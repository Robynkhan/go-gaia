# This Makefile is meant to be used by people that do not usually work
# with Go source code. If you know what GOPATH is then you probably
# don't need to bother with make.

.PHONY: gfbc android ios gfbc-cross swarm evm all test clean
.PHONY: gfbc-linux gfbc-linux-386 gfbc-linux-amd64 gfbc-linux-mips64 gfbc-linux-mips64le
.PHONY: gfbc-linux-arm gfbc-linux-arm-5 gfbc-linux-arm-6 gfbc-linux-arm-7 gfbc-linux-arm64
.PHONY: gfbc-darwin gfbc-darwin-386 gfbc-darwin-amd64
.PHONY: gfbc-windows gfbc-windows-386 gfbc-windows-amd64

GOBIN = $(shell pwd)/build/bin
GO ?= latest

gfbc:
	build/env.sh go run build/ci.go install ./cmd/gfbc
	@echo "Done building."
	@echo "Run \"$(GOBIN)/gfbc\" to launch gfbc."

swarm:
	build/env.sh go run build/ci.go install ./cmd/swarm
	@echo "Done building."
	@echo "Run \"$(GOBIN)/swarm\" to launch swarm."

all:
	build/env.sh go run build/ci.go install

android:
	build/env.sh go run build/ci.go aar --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/gfbc.aar\" to use the library."

ios:
	build/env.sh go run build/ci.go xcode --local
	@echo "Done building."
	@echo "Import \"$(GOBIN)/Gfbc.framework\" to use the library."

test: all
	build/env.sh go run build/ci.go test

clean:
	rm -fr build/_workspace/pkg/ $(GOBIN)/*

# The devtools target installs tools required for 'go generate'.
# You need to put $GOBIN (or $GOPATH/bin) in your PATH to use 'go generate'.

devtools:
	env GOBIN= go get -u golang.org/x/tools/cmd/stringer
	env GOBIN= go get -u github.com/jteeuwen/go-bindata/go-bindata
	env GOBIN= go get -u github.com/fjl/gencodec
	env GOBIN= go install ./cmd/abigen

# Cross Compilation Targets (xgo)

gfbc-cross: gfbc-linux gfbc-darwin gfbc-windows gfbc-android gfbc-ios
	@echo "Full cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-*

gfbc-linux: gfbc-linux-386 gfbc-linux-amd64 gfbc-linux-arm gfbc-linux-mips64 gfbc-linux-mips64le
	@echo "Linux cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-*

gfbc-linux-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/386 -v ./cmd/gfbc
	@echo "Linux 386 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep 386

gfbc-linux-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/amd64 -v ./cmd/gfbc
	@echo "Linux amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep amd64

gfbc-linux-arm: gfbc-linux-arm-5 gfbc-linux-arm-6 gfbc-linux-arm-7 gfbc-linux-arm64
	@echo "Linux ARM cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep arm

gfbc-linux-arm-5:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-5 -v ./cmd/gfbc
	@echo "Linux ARMv5 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep arm-5

gfbc-linux-arm-6:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-6 -v ./cmd/gfbc
	@echo "Linux ARMv6 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep arm-6

gfbc-linux-arm-7:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm-7 -v ./cmd/gfbc
	@echo "Linux ARMv7 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep arm-7

gfbc-linux-arm64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/arm64 -v ./cmd/gfbc
	@echo "Linux ARM64 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep arm64

gfbc-linux-mips:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips --ldflags '-extldflags "-static"' -v ./cmd/gfbc
	@echo "Linux MIPS cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep mips

gfbc-linux-mipsle:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mipsle --ldflags '-extldflags "-static"' -v ./cmd/gfbc
	@echo "Linux MIPSle cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep mipsle

gfbc-linux-mips64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64 --ldflags '-extldflags "-static"' -v ./cmd/gfbc
	@echo "Linux MIPS64 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep mips64

gfbc-linux-mips64le:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=linux/mips64le --ldflags '-extldflags "-static"' -v ./cmd/gfbc
	@echo "Linux MIPS64le cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-linux-* | grep mips64le

gfbc-darwin: gfbc-darwin-386 gfbc-darwin-amd64
	@echo "Darwin cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-darwin-*

gfbc-darwin-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/386 -v ./cmd/gfbc
	@echo "Darwin 386 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-darwin-* | grep 386

gfbc-darwin-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=darwin/amd64 -v ./cmd/gfbc
	@echo "Darwin amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-darwin-* | grep amd64

gfbc-windows: gfbc-windows-386 gfbc-windows-amd64
	@echo "Windows cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-windows-*

gfbc-windows-386:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/386 -v ./cmd/gfbc
	@echo "Windows 386 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-windows-* | grep 386

gfbc-windows-amd64:
	build/env.sh go run build/ci.go xgo -- --go=$(GO) --targets=windows/amd64 -v ./cmd/gfbc
	@echo "Windows amd64 cross compilation done:"
	@ls -ld $(GOBIN)/gfbc-windows-* | grep amd64
