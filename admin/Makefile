.PHONY: web web-dev tidy build

web:
	cd web && pnpm install && pnpm build

web-dev:
	cd web && pnpm dev

# Build UI and verify ingress binary includes admin command.
build: web
	cd .. && go build -o admin/bin/ingress ./cmd/ingress

tidy:
	cd .. && go mod tidy
