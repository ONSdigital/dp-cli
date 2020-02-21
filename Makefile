SHELL=bash

ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))
TEMPLATES_DIR:=$(ROOT_DIR)/project-generation/content/templates

build:
	go build -ldflags="-X 'github.com/ONSdigital/dp-cli/project_generation.templatesPath=$(TEMPLATES_DIR)'"

install:
	go install -ldflags="-X 'github.com/ONSdigital/dp-cli/project_generation.templatesPath=$(TEMPLATES_DIR)'"

echo:
	echo $(TEMPLATES_DIR)

.PHONY: install echo