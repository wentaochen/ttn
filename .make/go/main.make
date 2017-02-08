
# Programs
GO = go
GOLINT = golint

ifeq ($(GIT_BRANCH), $(GIT_TAG))
	TTN_VERSION = $(GIT_TAG)
	GO_TAGS += prod
else
	TTN_VERSION = $(GIT_TAG)-dev
	GO_TAGS += dev
endif

# Flags
## go
GO_FLAGS = -a -installsuffix cgo
GO_ENV = CGO_ENABLED=0
LD_FLAGS = -ldflags "-w -X main.version=$(TTN_VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildDate=$(BUILD_DATE) -X main.gitBranch=$(GIT_BRANCH)"

## golint
GOLINT_FLAGS = -set_exit_status

## test
GO_TEST_FLAGS = -cover

# Filters

## select only go files
only_go = grep '.go$$'

## select/remove vendored files
no_vendor = grep -v 'vendor'
only_vendor = grep 'vendor'

## select/remove mock files
no_mock = grep -v '_mock.go'
only_mock = grep '_mock.go'

## select/remove protobuf generated files
no_pb = grep -Ev '.pb.go$$|.pb.gw.go$$'
only_pb = grep -E '.pb.go$$|.pb.gw.go$$'

## select/remove test files
no_test = grep -v '_test.go$$'
only_test = grep '_test.go$$'

## filter files to packages
to_packages = sed 's:/[^/]*$$::' | sort | uniq

## make packages local (prefix with ./)
to_local = sed 's:^:\./:'


# Selectors

## find all go files
GO_FILES = find . -name '*.go' | grep -v '.git'

## local go packages
GO_PACKAGES = $(GO_FILES) | $(no_vendor) | $(to_packages)

## external go packages (in vendor)
EXTERNAL_PACKAGES = $(GO_FILES) | $(only_vendor) | $(to_packages)

## staged local packages
STAGED_PACKAGES = $(STAGED_FILES) | $(only_go) | $(no_vendor) | $(to_packages) | $(to_local)

## packages for testing
TEST_PACKAGES = $(GO_FILES) | $(no_vendor) | $(only_test) | $(to_packages)

# Rules

## get tools required for development
go.dev-deps:
	@$(log) "fetching go tools"
	@command -v govendor > /dev/null || ($(log) Installing govendor && $(GO) get -u github.com/kardianos/govendor)
	@command -v golint > /dev/null || ($(log) Installing golint && $(GO) get -u github.com/golang/lint/golint)

## install dependencies
go.deps:
	@$(log) "fetching go dependencies"
	@govendor sync -v

## install packages for faster rebuilds
go.install:
	@$(log) "installing go packages"
	@$(GO_PACKAGES) | xargs $(GO) install -v

## clean build files
go.clean:
	@$(log) "cleaning release dir" [rm -rf $(RELEASE_DIR)]
	@rm -rf $(RELEASE_DIR)

## run tests
go.test:
	@$(log) testing `$(TEST_PACKAGES) | $(count)` go packages
	@$(GO) test $(GO_TEST_FLAGS) `$(TEST_PACKAGES)`

# vim: ft=make
