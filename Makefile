SHELL := /bin/bash

CURR_DIR := $(shell pwd)
MODULES := $(shell go work edit -json .go.work | jq ".Use[].DiskPath")

build:
	@for dir in $(MODULES); do \
		cd "$(CURR_DIR)/$$dir" && go build ./...; \
	done

test:
	@for dir in $(MODULES); do \
		cd "$(CURR_DIR)/$$dir" && go test -v ./...; \
	done

.PHONY: _prepare_proto
_prepare_proto:
	@for pkg in \
		google.golang.org/protobuf/cmd/protoc-gen-go \
		google.golang.org/grpc/cmd/protoc-gen-go-grpc; \
	do GO111MODULE=on go install -v $$pkg@latest 2>&1; done

# make _proto pkg=openrtbadx files="a.proto b.proto" sedcmd="..."
.PHONY: _proto
_proto:
	@echo "update proto/$(pkg) package..."
	@rm -fr proto/$(pkg) && mkdir -p proto/$(pkg)
	@cd "$(CURR_DIR)/proto/$(pkg)" \
		&& go mod init github.com/mechiru/ab/proto/$(pkg) \
		&& for file in $(files); do \
				curl "https://developers.google.com/authorized-buyers/rtb/downloads/$$(echo $$file | sed -e 's!.proto!-proto.txt!')" -o $$file; \
			done \
		&& sed -i '$(sedcmd)' $(files) \
		&& export PATH=$$(go env GOPATH)/bin:$$PATH \
		&& protoc -I=. --go_out=. --go-grpc_out=. $(files) \
		&& go mod tidy

.PHONY: proto
proto: _prepare_proto
	@# https://developers.google.com/authorized-buyers/rtb/openrtb-guide
	@make _proto pkg=openrtb \
		files="openrtb.proto openrtb-adx.proto" \
		sedcmd="1s!^!syntax = \"proto2\";\noption go_package = \"./;openrtb\";\n!"

	@# https://developers.google.com/authorized-buyers/rtb/realtime-bidding-guide
	@make _proto pkg=networkbid \
		files="realtime-bidding.proto" \
		sedcmd="1s!^!syntax = \"proto2\";\noption go_package = \"./;networkbid\";\npackage com.google.protos.adx;\n!"
