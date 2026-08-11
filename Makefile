BIN=bin/blog-service
DOCKER_IMAGE?=marketing-digest-blog-service
WS_ROOT:=$(abspath ../../..)

.PHONY: build test lint run docker-build atlas-hash atlas-lint atlas-validate tidy

build:
	mkdir -p bin
	go build -o $(BIN) ./cmd/server

test:
	go test ./...

lint:
	go vet ./...

tidy:
	go mod tidy

run: build
	set -a && [ -f .env ] && . ./.env; set +a; ./$(BIN)

docker-build:
	docker build -f $(abspath Dockerfile) -t $(DOCKER_IMAGE) $(WS_ROOT)

atlas-hash:
	atlas migrate hash --dir file://migrations

atlas-lint:
	atlas migrate lint --env local --latest 1

atlas-validate:
	atlas migrate validate --dir file://migrations
