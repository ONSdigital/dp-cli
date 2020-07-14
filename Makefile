SHELL=bash

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
VERSION:=$(shell git describe --tags --dirty)
TEMPLATES_DIR:=$(ROOT_DIR)/project_generation/content/templates

GIT_PATH=github.com/ONSdigital/dp-cli
APP_PATH=$(GIT_PATH)/cmd/dp
LD_FLAGGER=-ldflags="-X '$(GIT_PATH)/command.appVersion=$(VERSION)' -X '$(GIT_PATH)/project_generation.templatesPath=$(TEMPLATES_DIR)'"

build:
	go build $(LD_FLAGGER) $(APP_PATH)

install:
	go install $(LD_FLAGGER) $(APP_PATH)

debug:
	go run -race $(LD_FLAGGER) $(APP_PATH)

test:
	go test -race -cover ./...

.PHONY: build install test debug
