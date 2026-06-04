#!/bin/bash

install() {
    echo "Installing dependencies..."
    go get -u google.golang.org/protobuf/cmd/protoc-gen-go@latest
   	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   	go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   	go mod tidy
}

orm() {
    echo "Installing protoc-gen-go-orm..."
    go install github.com/wxdqing/protoc-gen-go-orm@latest
    go mod tidy
}

gen() {
  sh gen_pb.sh
}

pri() {
  	go env -w GOPRIVATE=git.ghgame.local,git.tencent.com
  	go env -w GOINSECURE=git.ghgame.local
}

all() {
    pri
    install
}

# 根据命令行参数执行相应操作
parse_args() {
    if [ -z "$1" ]; then
        all
    elif [ "$1" == "install" ]; then
        install
    elif [ "$1" == "orm" ]; then
        orm
    elif [ "$1" == "gen" ]; then
        gen
    else
        echo "Unknown command: $1"
    fi
}

# 调用解析参数函数
parse_args "$@"
