BINARY := netfin
IMAGE ?= netfin:local

.PHONY: build test docker-build clean

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -o bin/$(BINARY) ./cmd/sidecar

test:
	go test ./...

docker-build:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(IMAGE) .

clean:
	rm -rf bin
