#!/usr/bin/env bash

# golangci-lint 代码检查
CMD_GOLANGCI_LINT=golangci-lint
if type ${CMD_GOLANGCI_LINT} &>/dev/null; then
  GO111MODULE=on ${CMD_GOLANGCI_LINT} run --exclude 'redundant `return` statement' ./... --fast
else
  echo -e "\033[31m'${CMD_GOLANGCI_LINT}' not found\033[0m"
fi

# golint 代码检查
CMD_GO_LINT=golint
if type ${CMD_GO_LINT} &>/dev/null; then
  GO111MODULE=on ${CMD_GO_LINT} -set_exit_status ./...
else
  echo -e "\033[31m'${CMD_GO_LINT}' not found\033[0m"
fi

set -e
# build
env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build --tags=jsoniter -o metric-index main.go
