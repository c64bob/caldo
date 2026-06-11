.PHONY: build dev stage-caldav assets tailwind templ verify-assets test lint e2e e2e-webkit e2e-ci e2e-headed docker-build

BINARY := caldo
BINARY_DIR := bin
BINARY_PATH := $(BINARY_DIR)/$(BINARY)

build: templ assets verify-assets
	@mkdir -p $(BINARY_DIR)
	go build -o $(BINARY_PATH) ./cmd/caldo

dev:
	go run ./cmd/caldo

stage-caldav:
	go run ./cmd/stagecaldav

assets:
	./scripts/build-assets.sh

tailwind: assets

templ:
	go run github.com/a-h/templ/cmd/templ@v0.3.1020 generate

test:
	go test ./...

verify-assets:
	go test ./internal/assets -run TestLoadManifest -count=1

lint:
	go vet ./...

e2e:
	npm run test:e2e

e2e-webkit:
	npm run test:e2e:webkit

e2e-ci:
	npm run test:e2e:ci

e2e-headed:
	npm run test:e2e:headed

docker-build:
	@if command -v docker >/dev/null 2>&1; then \
		docker build .; \
	else \
		echo "docker not found in this environment; image builds are validated in CI"; \
	fi
