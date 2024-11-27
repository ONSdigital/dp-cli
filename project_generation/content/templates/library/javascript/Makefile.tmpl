GREEN  := $(shell tput -Txterm setaf 2)
YELLOW := $(shell tput -Txterm setaf 3)
WHITE  := $(shell tput -Txterm setaf 7)
CYAN   := $(shell tput -Txterm setaf 6)
RESET  := $(shell tput -Txterm sgr0)

.PHONY: all
all: delimiter-AUDIT audit delimiter-UNIT-TESTS test delimiter-LINTERS lint delimiter-FINISH ## Runs multiple targets, audit, lint and test

.PHONY: audit
audit: ## Runs checks for security vulnerabilities on dependencies (including transient ones)
	npm audit

.PHONY: build
build: ## Builds binary of library code
	npm run build

.PHONY: clean
clean: ## Cleans up build directory
	npm run clean

.PHONY: install
install: ## Installs dependencies
	npm install

.PHONY: delimiter-%
delimiter-%:
	@echo '===================${GREEN} $* ${RESET}==================='

.PHONY: lint
lint: ## Used in ci to run linters against JS code
	npm run lint

.PHONY: lint-fix
lint-fix: ## Used in ci to run linters against JS code
	npm run lint:fix

.PHONY: test
test: ## Runs unit tests including checks for race conditions and returns coverage
	npm test

.PHONY: help
help: ## Show help page for list of make targets
	@echo ''
	@echo 'Usage:'
	@echo '  ${YELLOW}make${RESET} ${GREEN}<target>${RESET}'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} { \
		if (/^[a-zA-Z_-]+:.*?##.*$$/) {printf "    ${YELLOW}%-20s${GREEN}%s${RESET}\n", $$1, $$2} \
		else if (/^## .*$$/) {printf "  ${CYAN}%s${RESET}\n", substr($$1,4)} \
		}' $(MAKEFILE_LIST)
