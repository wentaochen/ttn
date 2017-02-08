# Infer GOOS and GOARCH
GOOS   ?= $(or $(word 1,$(subst -, ,${TARGET_PLATFORM})), $(shell echo "`go env GOOS`"))
GOARCH ?= $(or $(word 2,$(subst -, ,${TARGET_PLATFORM})), $(shell echo "`go env GOARCH`"))

MAIN ?= ./main.go

# build
go.build: $(RELEASE_DIR)/$(NAME)-$(GOOS)-$(GOARCH)

# Build the executable
$(RELEASE_DIR)/$(NAME)-%: $(shell $(GO_FILES)) vendor/vendor.json
	@$(log) "building" [$(GO_ENV) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO_ENV) $(GO) build $(GO_FLAGS) ... $(MAIN) ]
	@$(GO_ENV) GOOS=$(GOOS) GOARCH=$(GOARCH) $(GO) build -o "$(RELEASE_DIR)/$(NAME)-$(GOOS)-$(GOARCH)" $(GO_FLAGS) $(LD_FLAGS) -tags "$(GO_TAGS)" $(MAIN)

# Build the executable in dev mode (much faster)
go.dev: GO_FLAGS = -v
go.dev: GO_ENV =
go.dev: BUILD_TYPE = dev
go.dev: $(RELEASE_DIR)/$(NAME)-$(GOOS)-$(GOARCH)

## link the executable to a simple name
$(RELEASE_DIR)/$(NAME): $(RELEASE_DIR)/$(NAME)-$(GOOS)-$(GOARCH)
	@$(log) "linking binary" [ln -sfr $(RELEASE_DIR)/$(NAME)-$(GOOS)-$(GOARCH) $(RELEASE_DIR)/$(NAME)]
	@ln -sfr $(RELEASE_DIR)/$(NAME)-$(GOOS)-$(GOARCH) $(RELEASE_DIR)/$(NAME)

go.link: $(RELEASE_DIR)/$(NAME)

go.link-dev: GO_FLAGS = -v
go.link-dev: GO_ENV =
go.link-dev: BUILD_TYPE = dev
go.link-dev: go.link

## initialize govendor
vendor/vendor.json:
	@$(log) initializing govendor
	@govendor init

# vim: ft=make
