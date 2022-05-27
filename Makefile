include .bingo/Variables.mk
FILES_TO_FMT	?= $(shell find . -name '*.go' -print)

GO111MODULE 	?= on
export GO111MODULE

GOPATH      	?= $(shell go env GOPATH)
TMP_GOPATH  	?= /tmp/
GOBIN       	?= ${GOPATH}/bin
export 			PATH :=${PATH}:/tmp:${GOBIN}

# Tools.
GIT             ?= $(shell which git)

# Support gsed on OSX (installed via brew), falling back to sed. On Linux
# systems gsed won't be installed, so will use sed as expected.
SED ?= $(shell which gsed 2>/dev/null || which sed)

define require_clean_work_tree
	@git update-index -q --ignore-submodules --refresh

    @if ! git diff-files --quiet --ignore-submodules --; then \
        echo >&2 "cannot $1: you have unstaged changes."; \
        git diff-files --name-status -r --ignore-submodules -- >&2; \
        echo >&2 "Please commit or stash them."; \
        exit 1; \
    fi

    @if ! git diff-index --cached --quiet HEAD --ignore-submodules --; then \
        echo >&2 "cannot $1: your index contains uncommitted changes."; \
        git diff-index --cached --name-status -r --ignore-submodules HEAD -- >&2; \
        echo >&2 "Please commit or stash them."; \
        exit 1; \
    fi

endef

help: ## Displays help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: all
all: format build

.PHONY: build
build: ## Build.
	@echo ">> Building"
	@go build ./cmd/contraget/

.PHONY: check-comments
check-comments: ## Checks Go code comments if they have trailing period (excludes protobuffers and vendor files). Comments with more than 3 spaces at beginning are omitted from the check, example: '//    - foo'.
	@echo ">> checking Go comments trailing periods\n\n\n"
	@./scripts/build-check-comments.sh

.PHONY: deps
deps: ## Ensures fresh go.mod and go.sum.
	@go mod tidy
	@go mod verify

.PHONY: format
format: ## Formats Go code including imports and cleans up white noise.
format: $(GOIMPORTS)
	@echo ">> formatting code"
	@$(GOIMPORTS) -w $(FILES_TO_FMT)
	@SED_BIN="$(SED)" scripts/cleanup-white-noise.sh $(FILES_TO_FMT)

.PHONY: test
test: ## Runs all Go unit tests.
export GOCACHE=/tmp/cache
test:
	@echo ">> running unit tests (without cache)"
	@rm -rf $(GOCACHE)
	@go test -v -timeout=30m $(shell go list ./... | grep -v /vendor/);

.PHONY: check-git
check-git:
ifneq ($(GIT),)
	@test -x $(GIT) || (echo >&2 "No git executable binary found at $(GIT)."; exit 1)
else
	@echo >&2 "No git binary found."; exit 1
endif

.PHONY:lint
lint: ## Runs various static analysis against our code.
lint: go-lint shell-lint
	@echo ">> detecting white noise"
	@find . -type f \( -name "*.md" -o -name "*.go" \) | SED_BIN="$(SED)" xargs scripts/cleanup-white-noise.sh
	$(call require_clean_work_tree,'detected white noise, run make lint and commit changes')

# PROTIP:
# Add
#      --cpu-profile-path string   Path to CPU profile output file
#      --mem-profile-path string   Path to memory profile output file
# to debug big allocations during linting.
.PHONY: go-lint
go-lint: check-git deps
	@echo ">> verifying modules being imported"
	@faillint -paths "errors=github.com/pkg/errors" ./...
	@faillint -paths "fmt.{Print,Printf,Println,Sprint}" -ignore-tests ./...
	@echo ">> linting all of the Go files GOGC=${GOGC}"
	@golangci-lint run
	@echo ">> detecting misspells"
	@find . -type f | grep -v tmp | grep -v node_modules | grep -v contracts | grep -v go.sum | grep -vE '\./\..*' | xargs misspell -error
	@echo ">> ensuring Copyright headers"
	@go run ./scripts/copyright
	$(call require_clean_work_tree,'detected file changes, run make lint and commit changes')

.PHONY:shell-lint
shell-lint: $(SHELLCHECK)
	@echo ">> linting all of the shell script files"
	@$(SHELLCHECK) --severity=error -o all -s bash $(shell find . -type f -name "*.sh" -not -path "*vendor*" -not -path "tmp/*" -not -path "*node_modules*")

