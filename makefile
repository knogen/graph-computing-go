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
	go run main.go oae -t subject

wiki-entropy:
	go run main.go wikiEntropy -t subject

wiki-extract:
	go run main.go excractWikipediaDump

wiki-degree-stats:
	go run main.go wikiDegreeStats

openalex-degree-stats:
	go run main.go openalexDegreeStats