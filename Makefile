.PHONY: all
all: build/cmd

GOPATH ?= $(HOME)/go

build/cmd: proto
	mkdir -p build
	go build -o build ./cmd

.PHONY: proto
proto:
	#protoc api/stream.proto -I. --go_out=plugins=grpc:$(GOPATH)/src
	protoc api/stream.proto -I. --go_out=plugins=grpc:. --go_opt=paths=source_relative
