.PHONY: web web-dev tidy build

# Output: core/admin/static/dist (embedded with -tags adminui; see static/embed_dist.go)
web:
	cd web && pnpm install && pnpm build

web-dev:
	cd web && pnpm dev

# Build UI and verify ingress binary (adminui embeds core/admin/static/dist).
build: web
	cd ../.. && go build -tags adminui -o core/admin/bin/ingress ./cmd/ingress

tidy:
	cd ../.. && go mod tidy
