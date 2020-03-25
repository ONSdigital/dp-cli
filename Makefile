SHELL=bash

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TEMPLATES_DIR:=$(ROOT_DIR)/project_generation/content/templates
VERSION:=$(shell git describe --tags --dirty)

build:
	go build -ldflags="-X 'github.com/ONSdigital/dp-cli/command.appVersion=$(VERSION)' -X 'github.com/ONSdigital/dp-cli/project_generation.templatesPath=$(TEMPLATES_DIR)'" github.com/ONSdigital/dp-cli/cmd/dp

install:
	go install -ldflags="-X 'github.com/ONSdigital/dp-cli/command.appVersion=$(VERSION)' -X 'github.com/ONSdigital/dp-cli/project_generation.templatesPath=$(TEMPLATES_DIR)'" github.com/ONSdigital/dp-cli/cmd/dp

.PHONY: install echo