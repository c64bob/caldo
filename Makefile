.PHONY: build dev tailwind templ test lint docker-build

BINARY := caldo

build:
	go build ./...

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
		echo "templ not found; skipping local templ generation"; \
	fi

test:
	go test ./...

lint:
	go vet ./...

docker-build:
	docker build .
