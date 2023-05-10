BUILD_DIR := $(CURDIR)/build

.PHONY: all test clean

all: build-opera-docker-image norma

pull-hello-world-image:
	docker image pull hello-world

build-opera-docker-image:
	docker build . -t opera

generate-abi: load/contracts/abi/Counter.abi # requires installed solc and Ethereum abigen - check README.md

load/contracts/abi/Counter.abi: load/contracts/Counter.sol
	cd load/generator; solc -o ../contracts/abi --overwrite --pretty-json --optimize --abi --bin ../contracts/Counter.sol
	abigen --type Counter --pkg abi --abi load/contracts/abi/Counter.abi --bin load/contracts/abi/Counter.bin --out load/contracts/abi/Counter.go

generate-mocks: # requires installed mockgen
	go generate ./...

norma: build-opera-docker-image
	go build -o $(BUILD_DIR)/norma ./driver/norma

test: pull-hello-world-image build-opera-docker-image
	go test ./... -v

clean:
	rm -rvf $(CURDIR)/build