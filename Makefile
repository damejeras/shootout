.PHONY: all
all: build

.PHONY: build
build: ## Build docker images required by the operator.
	docker build --tag shootout-shooter -f images/shooter/Dockerfile .
	docker build --tag shootout-arbiter -f images/arbiter/Dockerfile .