export GO111MODULE=on
export GOFLAGS=-mod=vendor

.PHONY: build
build: cmd/bashgpt

cmd/bashgpt: vendor FORCE
	go build -o bashgpt cmd/bashgpt/*.go

vendor: go.mod
	go mod download
	go mod vendor

cmd/bashgpt/version.txt:
	git describe --tags > cmd/bashgpt/version.txt
	truncate -s -1 cmd/bashgpt/version.txt

.PHONY: clean
clean:
	rm -f bashgpt unit-tests.xml

.PHONY: deps
deps:
	rm -rf go.mod go.sum vendor
	go mod init github.com/mikeb26/bashgpt
	GOPROXY=direct go mod tidy
	go mod vendor

FORCE:
