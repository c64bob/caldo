.PHONY: build dev tailwind templ verify-assets test lint docker-build

BINARY := caldo
BINARY_DIR := bin
BINARY_PATH := $(BINARY_DIR)/$(BINARY)

build: templ tailwind verify-assets
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_PATH) ./cmd/caldo

dev:
	go run ./cmd/caldo

tailwind:
	@if command -v tailwindcss >/dev/null 2>&1; then \
		tailwindcss -i ./web/static/tailwind.input.css -o ./web/static/tailwind.output.css --minify; \
	else \
		echo "tailwindcss not found; skipping local tailwind build"; \
	fi

templ:
	@if command -v templ >/dev/null 2>&1; then \
		templ generate; \
	else \
		echo "templ not found; running pinned generator via go run"; \
		go run github.com/a-h/templ/cmd/templ@v0.3.865 generate; \
	fi

test:
	go test ./...

verify-assets:
	go test ./internal/assets -run TestLoadManifest -count=1

lint:
	go vet ./...

docker-build:
	@if command -v docker >/dev/null 2>&1; then \
		docker build .; \
	else \
		echo "docker not found in this environment; image builds are validated in CI"; \
	fi
