SHELL := /bin/bash
FILE_NAME := terraform-provider-confluent-kafka_v0.1.0
.SILENT: compile

prepare:
	@go mod vendor

test:
	@go mod vendor
	@go test

build_dev:
	DIR := ${HOME}/.terraform.d/plugins/registry.terraform.io/hashicorp/confluent-kafka/0.1.0/darwin_amd64
	@go build -o ${DIR}/${FILE_NAME} main.go

build:
	@mkdir -p bin
	@go build -o bin/${FILE_NAME} main.go

compile:
	@echo "Compiling for every OS and Platform"
	GOOS=linux GOARCH=amd64 go build -o bin/${FILE_NAME}_linux-amd64 main.go
	GOOS=linux GOARCH=arm go build -o bin/${FILE_NAME}_linux-arm main.go
	GOOS=darwin GOARCH=amd64 go build -o bin/${FILE_NAME}_darwin-amd64 main.go
	GOOS=windows GOARCH=amd64 go build -o bin/${FILE_NAME}_windows-amd64 main.go
	GOOS=darwin GOARCH=arm go build -o bin/${FILE_NAME}_darwin-arm main.go

all: prepare compile