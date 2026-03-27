GITCOMMIT := $(shell git rev-parse HEAD)
GITDATE := $(shell git show -s --format='%ct')

LDFLAGSSTRING +=-X main.GitCommit=$(GITCOMMIT)
LDFLAGSSTRING +=-X main.GitDate=$(GITDATE)
LDFLAGS := -ldflags "$(LDFLAGSSTRING)"

MESSAGE_MANAGER_ABI_ARTIFACT := ./abis/MessageManager.sol/MessageManager.json
POOLMANAGER_ABI_ARTIFACT := ./abis/PoolManager.sol/PoolManager.json

flashloan-scanner:
	env GO111MODULE=on go build -v $(LDFLAGS) ./cmd/flashloan-scanner

clean:
	rm -f flashloan-scanner

test:
	go test -v ./...

lint:
	golangci-lint run ./...

bindings: binding-pool binding-msg

binding-pool:
	$(eval temp := $(shell mktemp))

	cat $(POOLMANAGER_ABI_ARTIFACT) \
    	| jq -r .bytecode.object > $(temp)

	cat $(POOLMANAGER_ABI_ARTIFACT) \
		| jq .abi \
		| abigen --pkg bindings \
		--abi - \
		--out bindings/pool_manager.go \
		--type PoolManager \
		--bin $(temp)

		rm $(temp)

binding-msg:
	$(eval temp := $(shell mktemp))

	cat $(MESSAGE_MANAGER_ABI_ARTIFACT) \
		| jq -r .bytecode.object > $(temp)

	cat $(MESSAGE_MANAGER_ABI_ARTIFACT) \
		| jq .abi \
		| abigen --pkg bindings \
		--abi - \
		--out bindings/message_manager.go \
		--type MessageManager \
		--bin $(temp)

		rm $(temp)


.PHONY: \
	flashloan-scanner \
	bindings \
	bindings-pool \
	bindings-msg \
	clean \
	test \
	lint

