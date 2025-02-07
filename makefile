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
	go run main.go oae -t structural

wiki-entropy:
	go run main.go wikiEntropy -t complexity

wiki-extract:
	go run main.go excractWikipediaDump

wiki-degree-stats:
	go run main.go wikiDegreeStats

openalex-degree-stats:
	go run main.go openalexDegreeStats

wikipedia-google-distance:
	go run main.go wikiGoogleDistance

wikiInDegree:
	go run main.go wikiInDegree