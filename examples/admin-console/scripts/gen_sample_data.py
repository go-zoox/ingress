#!/usr/bin/env python3
"""Generate access.log and error.log for examples/admin-console (~90 days of traffic)."""

from __future__ import annotations

import random
from datetime import datetime, timedelta
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
END = datetime.now().replace(microsecond=0)
START = END - timedelta(days=90)

CLIENT_IPS = [
    "203.0.113.44",
    "198.51.100.8",
    "192.0.2.99",
    "203.0.113.12",
    "10.0.0.5",
    "198.51.100.22",
    "203.0.113.88",
    "192.0.2.17",
]

ROUTES = [
    {
        "host": "api.example.com",
        "target": "api.internal:8080",
        "paths": [
            ("GET", "/api/users", 200, (8, 45), 0.08),
            ("GET", "/api/users/42", 200, (6, 30), 0.06),
            ("POST", "/api/login", 200, (10, 80), 0.02),
            ("POST", "/api/login", 401, (5, 15), 0.0),
            ("GET", "/search", 200, (20, 120), 0.05),
            ("GET", "/search", 400, (3, 8), 0.0),
            ("POST", "/api/orders", 201, (15, 90), 0.0),
            ("GET", "/v2/users", 200, (5, 25), 0.62),
            ("GET", "/v2/health", 200, (2, 8), 0.58),
            ("GET", "/public", 200, (4, 18), 0.48),
            ("GET", "/public/docs", 200, (3, 12), 0.44),
        ],
        "weight": 42,
        "cache_rate": 0.05,
    },
    {
        "host": "cdn.example.com",
        "target": "minio.internal:9000",
        "paths": [
            ("GET", "/assets/app.js", 200, (2, 12), 0.78),
            ("GET", "/assets/style.css", 200, (1, 6), 0.82),
            ("GET", "/assets/logo.png", 200, (3, 18), 0.71),
            ("GET", "/assets/vendor.js", 200, (4, 22), 0.75),
            ("GET", "/favicon.ico", 404, (1, 4), 0.15),
        ],
        "weight": 28,
        "cache_rate": 0.72,
    },
    {
        "host": "assets.cdn.example.com",
        "target": "minio.internal:9000",
        "paths": [
            ("GET", "/static/main.js", 200, (2, 10), 0.88),
            ("GET", "/static/theme.css", 200, (1, 5), 0.86),
        ],
        "weight": 8,
        "cache_rate": 0.85,
    },
    {
        "host": "tunnel-a.inlets.example.com",
        "target": "tunnel-a.tunnel:443",
        "paths": [
            ("GET", "/", 200, (30, 200)),
            ("GET", "/ws", 101, (5, 20)),
            ("GET", "/", 502, (8000, 15000)),
            ("GET", "/api", 504, (60000, 65000)),
        ],
        "weight": 6,
        "cache_rate": 0.0,
    },
    {
        "host": "tunnel-b.inlets.example.com",
        "target": "tunnel-b.tunnel:443",
        "paths": [
            ("GET", "/", 200, (25, 180)),
            ("GET", "/health", 200, (10, 40)),
        ],
        "weight": 4,
        "cache_rate": 0.0,
    },
    {
        "host": "admin.internal",
        "target": "handler",
        "paths": [
            ("GET", "/healthz", 200, (1, 3), 0.52),
            ("GET", "/", 200, (1, 4), 0.38),
        ],
        "weight": 5,
        "cache_rate": 0.0,
    },
    {
        "host": "legacy.example.com",
        "target": "redirect",
        "paths": [
            ("GET", "/", 301, (0, 1), 0.42),
            ("GET", "/old-page", 301, (0, 1), 0.35),
        ],
        "weight": 3,
        "cache_rate": 0.0,
    },
    {
        "host": "waf-demo.example.com",
        "target": "httpbin.org:443",
        "paths": [
            ("GET", "/admin", 403, (1, 4)),
            ("GET", "/../etc/passwd", 403, (1, 3)),
            ("GET", "/search?q=1' OR '1'='1", 403, (2, 6)),
            ("GET", "/", 200, (20, 80)),
        ],
        "weight": 4,
        "cache_rate": 0.0,
    },
]

USER_AGENTS = [
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
    "curl/8.0",
    "ingress-admin/1.0",
    "kube-probe/1.29",
    "scanner/1.0",
    "Go-http-client/1.1",
    "PostmanRuntime/7.36.0",
]

TLS = [
    ("-", "-"),
    ("TLS 1.3", "TLS_AES_128_GCM_SHA256"),
    ("TLS 1.2", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"),
]

ERROR_TEMPLATES = [
    "[prepareCache] failed to clear cache: redis: connection refused",
    "[waf block] phase=request client_ip={ip} host={host} method={method} path={path}",
    "service proxy: dial tcp 10.0.0.12:8080: connect: connection refused host={host} path={path}",
    "upstream 502 host={host} target={target} duration={dur}ms",
    "upstream 504 host={host} target={target} duration={dur}ms",
    "[reload] validate failed: rules[{n}].host: invalid regex",
    "tls handshake error host={host} err=remote error: tls: bad certificate",
    "handler panic recovered host={host} path={path}",
]


def weighted_choice(routes: list[dict]) -> dict:
    weights = [r["weight"] for r in routes]
    return random.choices(routes, weights=weights, k=1)[0]


def random_timestamp(start: datetime, end: datetime, *, business_hours: bool = True) -> datetime:
    span = int((end - start).total_seconds())
    if span <= 0:
        return end
    base = start + timedelta(seconds=random.randint(0, span))
    if business_hours and span > 6 * 3600:
        hour = random.randint(8, 21)
        base = base.replace(hour=hour, minute=random.randint(0, 59), second=random.randint(0, 59))
        if base < start:
            base = start + timedelta(seconds=random.randint(0, span))
        if base > end:
            base = end - timedelta(seconds=random.randint(0, min(span, 3600)))
    return base


def pick_timestamps(n: int) -> list[datetime]:
    """Mix historical spread with recent bursts for dashboard metrics."""
    out: list[datetime] = []
    recent_15m = END - timedelta(minutes=15)
    recent_1h = END - timedelta(hours=1)
    recent_24h = END - timedelta(hours=24)
    recent_7d = END - timedelta(days=7)

    buckets = [
        (int(n * 0.55), START, recent_7d, True),
        (int(n * 0.20), recent_7d, recent_24h, True),
        (int(n * 0.15), recent_24h, recent_1h, False),
        (int(n * 0.07), recent_1h, recent_15m, False),
        (int(n * 0.03), recent_15m, END, False),
    ]
    assigned = 0
    for count, lo, hi, biz in buckets:
        take = min(count, n - assigned)
        if take <= 0:
            break
        for _ in range(take):
            out.append(random_timestamp(lo, hi, business_hours=biz))
        assigned += take
    while len(out) < n:
        out.append(random_timestamp(START, recent_7d))
    out.sort()
    return out


def format_access_line(
    ts: datetime,
    route: dict,
    method: str,
    path: str,
    status: int,
    dur_ms: int,
    *,
    path_cache_rate: float | None = None,
) -> str:
    ip = random.choice(CLIENT_IPS)
    rate = route["cache_rate"] if path_cache_rate is None else path_cache_rate
    cache_hit = 1 if random.random() < rate else 0
    waf_block = 1 if status == 403 and route["host"] == "waf-demo.example.com" else 0
    tls_proto, tls_cipher = random.choice(TLS)
    if route["target"].endswith(":443"):
        tls_proto, tls_cipher = TLS[1] if random.random() < 0.7 else TLS[2]
    upstream_status = status if route["target"] != "redirect" else status
    upstream_len = random.choice([48, 1024, 2048, 8192, -1])
    if status >= 500:
        upstream_len = -1
    referer = random.choice(["-", "https://portal.example.com/", f"https://{route['host']}/"])
    ua = random.choice(USER_AGENTS)
    xff = ip if random.random() < 0.8 else f'"{ip}, 10.0.0.1"'
    q = ""
    if path == "/search" and status != 403:
        q = "?q=" + random.choice(["test", "ingress", "api", "docs"])
    return (
        f"{ts.strftime('%Y/%m/%d %H:%M:%S')} {ip} {route['host']} -> {route['target']} "
        f'"{method} {path}{q} HTTP/1.1" {status} {dur_ms}ms '
        f"cache_hit={cache_hit} waf_block={waf_block} real_ip={ip} referer={referer} "
        f"ua={ua} xff={xff} tls_protocol={tls_proto} tls_cipher={tls_cipher} "
        f"upstream_status={upstream_status} upstream_response_length={upstream_len} "
        f"upstream_response_time={dur_ms}ms"
    )


def parse_path_entry(entry: tuple) -> tuple[str, str, int, tuple[int, int], float | None]:
    if len(entry) >= 5:
        method, path, status, dur, cache_rate = entry
        return method, path, status, dur, cache_rate
    method, path, status, dur = entry
    return method, path, status, dur, None


def generate_access_lines(n: int = 4200) -> list[str]:
    random.seed(42)
    timestamps = pick_timestamps(n)
    lines: list[str] = []
    for ts in timestamps:
        route = weighted_choice(ROUTES)
        entry = random.choice(route["paths"])
        method, path, status, (lo, hi), path_cache_rate = parse_path_entry(entry)
        dur = random.randint(lo, hi)
        lines.append(
            format_access_line(ts, route, method, path, status, dur, path_cache_rate=path_cache_rate)
        )
    return lines


def generate_error_lines(n: int = 220) -> list[str]:
    random.seed(43)
    lines: list[str] = []
    timestamps = pick_timestamps(n)
    for ts in timestamps:
        route = weighted_choice(ROUTES)
        entry = random.choice(route["paths"])
        method, path, _, _, _ = parse_path_entry(entry)
        tpl = random.choice(ERROR_TEMPLATES)
        ip = random.choice(CLIENT_IPS)
        msg = tpl.format(
            ip=ip,
            host=route["host"],
            method=method,
            path=path,
            target=route["target"],
            dur=random.randint(5000, 65000),
            n=random.randint(0, 4),
        )
        lines.append(f"{ts.strftime('%Y/%m/%d %H:%M:%S')} {msg}")
    return lines


def main() -> None:
    access = generate_access_lines()
    errors = generate_error_lines()
    (ROOT / "access.log").write_text("\n".join(access) + "\n", encoding="utf-8")
    (ROOT / "error.log").write_text("\n".join(errors) + "\n", encoding="utf-8")
    print(f"wrote {len(access)} access lines, {len(errors)} error lines -> {ROOT}")


if __name__ == "__main__":
    main()
