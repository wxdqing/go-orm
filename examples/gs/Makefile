#
.PHONY: pri install

install: orm
	@go get -u google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	@go mod tidy

orm:
	@go install github.com/go-sprout/protoc-gen-go-orm@latest
	@go mod tidy

.PHONY: gen
gen:
	sh gen_pb.sh

pri:
	@go env -w GOPRIVATE=git.ghgame.local,git.tencent.com
	@go env -w GOINSECURE=git.ghgame.local
