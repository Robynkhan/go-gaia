.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/fairblock/go-fairblock/cmd/swarm
	go build -o bin/gfbc     github.com/fairblock/go-fairblock/cmd/gfbc
	go build -o bin/bootnode github.com/fairblock/go-fairblock/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/fairblock/go-fairblock/swarm/...
