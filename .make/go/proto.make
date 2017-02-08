## get tools required for development
proto.dev-deps:
	@$(log) "fetching proto tools"
	@command -v protoc-gen-gogofast > /dev/null || ($(log) Installing protoc-gen-gogofast && $(GO) get -u github.com/gogo/protobuf/protoc-gen-gogofast)
	@command -v protoc-gen-grpc-gateway > /dev/null || ($(log) Installing protoc-gen-grpc-gateway && $(GO) get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway)
	@command -v protoc-gen-ttndoc > /dev/null || ($(log) Installing protoc-gen-ttndoc && $(GO) install github.com/TheThingsNetwork/ttn/utils/protoc-gen-ttndoc)

PROTO_FILES = find . -name '*.proto' | grep -v '.git' | grep -v 'vendor'
COMPILED_PROTO_FILES = $(patsubst %.proto, %.pb.go, $(shell $(PROTO_FILES)))

PROTOC_IMPORTS= -I/usr/local/include -I$(GOPATH)/src -I$(shell dirname $(PWD)) -I$(GOPATH)/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis

PROTOC = protoc $(PROTOC_IMPORTS) \
--gogofast_out=Mgoogle/api/annotations.proto=github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis/google/api,plugins=grpc:$(GOPATH)/src \
--grpc-gateway_out=:$(GOPATH)/src `pwd`/

proto.clean:
	@$(log) cleaning `echo "$(COMPILED_PROTO_FILES)" | $(count)` proto files
	@rm -f $(COMPILED_PROTO_FILES)

proto.compile: $(COMPILED_PROTO_FILES)

%.pb.go: %.proto
	@$(log) "compiling proto $<"
	@$(PROTOC)$<
