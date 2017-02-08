
.PHONY: go.fmt go.fmt-staged go.vet go.vet-staged go.lint go.lint-staged go.quality go.quality-staged

# Fmt

## fmt all packages
go.fmt:
	@$(log) "formatting `$(GO_PACKAGES) | $(count)` go packages"
	@[[ -z "`$(GO_PACKAGES) | xargs go fmt | tee -a /dev/stderr`" ]]

## fmt stages packages
go.fmt-staged: GO_PACKAGES = $(STAGED_PACKAGES)
go.fmt-staged: go.fmt

# Vet

## vet all packages
go.vet:
	@$(log) "vetting `$(GO_PACKAGES) | $(count)` go packages"
	@$(GO_PACKAGES) | xargs $(GO) vet

## vet staged packages
go.vet-staged: GO_PACKAGES = $(STAGED_PACKAGES)
go.vet-staged: go.vet


# Linting

## lint all packages, exiting with non-zero if errors occur
GO_LINT_FILES = $(GO_FILES) | $(no_vendor) | $(no_mock) | $(no_pb)
go.lint:
	@$(log) "linting `$(GO_LINT_FILES) | $(count)` go files"
	@(for pkg in `$(GO_LINT_FILES)`; do $(GOLINT) $(GOLINT_FLAGS) $$pkg || export status=1; done; exit $$status)

## lint all packages, ignoring errors
go.lint-all: GOLINT_FLAGS =
go.lint-all: go.lint

# lint staged files
go.lint-staged: GO_LINT_FILES = $(STAGED_FILES) | $(only_go) | $(no_vendor) | $(no_mock) | $(no_pb)
go.lint-staged: go.lint

# Quality

## run all quality on all files
go.quality: go.fmt go.vet go.lint

## run all quality on staged files
go.quality-staged: go.fmt-staged go.vet-staged go.lint-staged

# vim: ft=make
