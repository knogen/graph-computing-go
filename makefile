SHELL := /bin/bash

ifneq (,$(wildcard ./.env))
    include .env
    export
endif



update:
	go get -u ./...

generate_grpc:
	protoc --go_out=.  --go-grpc_out=.  protos/wikiTextParser.proto 

openalex-entropy:
	go run main.go oae

wiki-entropy:
	go run main.go wikiEntropy

wiki-extract:
	go run main.go excractWikipediaDump