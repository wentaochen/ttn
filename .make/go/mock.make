mock.dev-deps:
	@$(log) "fetching mock tools"
	@command -v mockgen > /dev/null || ($(log) Installing mockgen && $(GO) get -u github.com/golang/mock/mockgen)

