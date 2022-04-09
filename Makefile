.PHONY: all
all: build

.PHONY: build
build: ## Build local docker images required by the operator.
	docker build --tag shootout-shooter -f images/shooter/Dockerfile .
	docker build --tag shootout-arbiter -f images/arbiter/Dockerfile .

.PHONY: test
test: ## Test Go code locally
	go test ./...
