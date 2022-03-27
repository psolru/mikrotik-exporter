LOCAL_BIN:=$(CURDIR)/bin
MINIMOCK_BIN:=$(LOCAL_BIN)/minimock
GOLANGCI_BIN:=$(LOCAL_BIN)/golangci-lint

export VERSION=1.1.0

export GOSUMDB=sum.golang.org
export GONOPROXY=
export GONOSUMDB=
export GOPRIVATE=
export GOPROXY=

docker-publish:
	./scripts/docker-publish.sh

bin-deps:
	$(info #Installing binary dependencies...)
	tmp=$$(mktemp -d) && cd $$tmp && pwd && go mod init temp && \
	GOBIN=$(LOCAL_BIN) go install github.com/gojuno/minimock/v3/cmd/minimock@v3.0.10 && \
	GOBIN=$(LOCAL_BIN) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.44.2

lint:
	$(info #Running lint...)
	$(GOLANGCI_BIN) run --new-from-rev=origin/master --config=.golangci.yaml ./...

lint-full:
	$(GOLANGCI_BIN) run --config=.golangci.yaml ./...

mocks:
	$(MINIMOCK_BIN) -g -i github.com/ogi4i/mikrotik-exporter/collector.* -o ./collector/mocks -s _mock.go
	$(MINIMOCK_BIN) -g -i github.com/ogi4i/mikrotik-exporter/routeros.* -o ./routeros/mocks -s _mock.go
