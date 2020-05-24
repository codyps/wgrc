.PHONY: all
all: cmd

GOPATH ?= $(HOME)/go

.PHONY: cmd
cmd: proto
	mkdir -p build
	go build -race -o build ./cmd/...

.PHONY: proto
proto:
	protoc api/stream.proto -I. --go_out=plugins=grpc:. --go_opt=paths=source_relative
