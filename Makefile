GO              := $(shell command -v go 2> /dev/null)
GIT             := $(shell command -v git 2> /dev/null)
REVIVE          := $(shell command -v revive 2> /dev/null)
PROTOC          := $(shell command -v protoc 2> /dev/null)
PROTOCGENGO     := $(shell command -v protoc-gen-go 2> /dev/null)
PROTOCGENGOGRPC := $(shell command -v protoc-gen-go-grpc 2> /dev/null)

VERSION         := $(shell grep "\[[0-9]\+\.[0-9]\+\.[0-9]\+\]" CHANGELOG.md | head -n 1 | grep -o '[0-9]\+\.[0-9]\+\.[0-9]\+')
BUILD           := $(shell git rev-parse HEAD)


go-check:
	@[ "${GO}" ] || ( echo ">> Go is not installed"; exit 1 )

revive-check: go-check
	@[ "${REVIVE}" ] || ( echo ">> Installing revive" && go install github.com/mgechev/revive@latest )

go-format: go-check
	@echo ">> formatting code"
	@$(GO) fmt ./...

go-vet: go-check
	@echo ">> vetting code"
	@$(GO) vet ./...

go-lint: revive-check go-format go-vet
	@echo ">> linting code"
	@revive -formatter stylish --config ./.revive.toml ./...

go-tests: go-check
	@echo ">> running tests"
	@go test ./...

git-check:
	@[ "${GIT}" ] || ( echo ">> git is not installed"; exit 1 )

protoc-check:
	@[ "${PROTOC}" ] || ( echo ">> protoc compiler is not installed, see http://google.github.io/proto-lens/installing-protoc.html"; exit 1 )
	@[ "${PROTOCGENGO}" ] || ( echo ">> Installing protoc-gen-go" && go install google.golang.org/protobuf/cmd/protoc-gen-go@latest )
	@[ "${PROTOCGENGOGRPC}" ] || ( echo ">> Installing protoc-gen-go-grpc" && go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest )

protos-clean:
	@echo ">> Cleaning gRPC generated code"
	@rm -f tinternal/pkg/foobar/*.pb.go

protos-compile: protoc-check
	@echo ">> Generating gRPC go code"
	@protoc --proto_path=internal/pkg/foobar --go_out=internal/pkg/foobar --go_opt=paths=source_relative --go-grpc_out=internal/pkg/foobar --go-grpc_opt=paths=source_relative internal/pkg/foobar/service.proto

protos-update: protos-clean protos-compile
