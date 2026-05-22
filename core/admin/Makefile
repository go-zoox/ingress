.PHONY: web web-dev tidy build

web:
	cd web && pnpm install && pnpm build

web-dev:
	cd web && pnpm dev

# Build UI and verify ingress binary.
build: web
	cd ../.. && go build -o core/admin/bin/ingress ./cmd/ingress

tidy:
	cd ../.. && go mod tidy
