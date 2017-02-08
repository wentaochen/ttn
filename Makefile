RELEASE_DIR = release

# BEGIN Deps

deps: go.deps

dev-deps: go.dev-deps proto.dev-deps mock.dev-deps
	@command -v forego > /dev/null || go get github.com/ddollar/forego
	@git config http.https://gopkg.in.followRedirects true

# END Deps

# BEGIN Proto

.PHONY: proto protodoc

proto: proto.compile

protodoc:
	@$(log) "compiling proto documentation"
	protoc $(PROTOC_IMPORTS) --ttndoc_out=logtostderr=true,.lorawan.DevAddrManager=all:$(GOPATH)/src $(PWD)/api/protocol/lorawan/device_address.proto
	protoc $(PROTOC_IMPORTS) --ttndoc_out=logtostderr=true,.handler.ApplicationManager=all:$(GOPATH)/src $(PWD)/api/handler/handler.proto
	protoc $(PROTOC_IMPORTS) --ttndoc_out=logtostderr=true,.discovery.Discovery=all:$(GOPATH)/src $(PWD)/api/discovery/discovery.proto

# END Proto

# BEGIN Mocks

.PHONY: mocks

mocks:
	mockgen -source=./api/networkserver/networkserver.pb.go -package networkserver NetworkServerClient > api/networkserver/networkserver_mock.go
	mockgen -source=./api/discovery/client.go -package discovery Client > api/discovery/client_mock.go

# END Mocks

# BEGIN Build

.PHONY: build ttn ttnctl dev ttn-dev ttnctl-dev

build:
	@$(MAKE) ttn
	@$(MAKE) ttnctl

ttn ttn-dev: NAME = ttn
ttn ttn-dev: MAIN = ./main.go

ttnctl ttnctl-dev: NAME = ttnctl
ttnctl ttnctl-dev: MAIN = ./ttnctl/main.go

ttn ttnctl: go.build

dev:
	@$(MAKE) ttn-dev
	@$(MAKE) ttnctl-dev

ttn-dev ttnctl-dev: go.dev

# END Build

# BEGIN Test

.PHONY: test quality quality-staged

test: go.test

quality: go.quality

quality-staged: go.quality-staged

# END Test

clean: go.clean

include ./.make/*.make
include ./.make/go/*.make

# BEGIN Coverage

.PHONY: cover-clean cover-deps cover coveralls

GO_COVER_FILE ?= coverage.out
GO_COVER_DIR ?= .cover

GO_COVER_PACKAGES = $(shell find . -name "*_test.go" | grep -vE ".git|.env|vendor|ttnctl|cmd|api" | sed 's:/[^/]*$$::' | sort | uniq)
GO_COVER_FILES = $(patsubst ./%, $(GO_COVER_DIR)/%.out, $(shell echo "$(GO_COVER_PACKAGES)"))

cover-clean:
	rm -rf $(GO_COVER_DIR) $(GO_COVER_FILE)

cover-deps:
	@command -v goveralls > /dev/null || go get github.com/mattn/goveralls

cover: $(GO_COVER_FILE)

$(GO_COVER_FILE): cover-clean $(GO_COVER_FILES)
	echo "mode: set" > $(GO_COVER_FILE)
	cat $(GO_COVER_FILES) | grep -vE "mode: set|/server.go|/manager_server.go" | sort >> $(GO_COVER_FILE)

$(GO_COVER_DIR)/%.out: %
	@mkdir -p "$(GO_COVER_DIR)/$<"
	go test -cover -coverprofile="$@" "./$<"

coveralls: cover-deps $(GO_COVER_FILE)
	goveralls -coverprofile=$(GO_COVER_FILE) -service=travis-ci -repotoken $$COVERALLS_TOKEN

# END Coverage

# BEGIN Docs

.PHONY: docs

docs:
	cd cmd/docs && HOME='$$HOME' go run generate.go > README.md
	cd ttnctl/cmd/docs && HOME='$$HOME' go run generate.go > README.md

# END Docs

# BEGIN Docker

.PHONY: docker

docker: TARGET_PLATFORM=linux-amd64
docker: ttn
	docker build -t thethingsnetwork/ttn -f Dockerfile .

# END Docker
