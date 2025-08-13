BUILDDIR ?= $(CURDIR)/build
TOOLS_DIR := tools

WASMD_PKG   := github.com/CosmWasm/wasmd/cmd/wasmd

GO_BIN := ${GOPATH}/bin
BTCD_BIN := $(GO_BIN)/btcd

DOCKER := $(shell which docker)
CUR_DIR := $(shell pwd)
MOCKS_DIR=$(CUR_DIR)/testutil/mocks
MOCKGEN_REPO=github.com/golang/mock/mockgen
MOCKGEN_VERSION=v1.6.0
MOCKGEN_CMD=go run ${MOCKGEN_REPO}@${MOCKGEN_VERSION}

ldflags := $(LDFLAGS)
build_tags := $(BUILD_TAGS)
build_args := $(BUILD_ARGS)

ifeq ($(LINK_STATICALLY),true)
	ldflags += -linkmode=external -extldflags "-Wl,-z,muldefs -static" -v
endif

ifeq ($(VERBOSE),true)
	build_args += -v
endif

BUILD_TARGETS := build install
BUILD_FLAGS := --tags "$(build_tags)" --ldflags '$(ldflags)'

# Update changelog vars
ifneq (,$(SINCE_TAG))
	sinceTag := --since-tag $(SINCE_TAG)
endif
ifneq (,$(UPCOMING_TAG))
	upcomingTag := --future-release $(UPCOMING_TAG)
endif

all: build install

build: BUILD_ARGS := $(build_args) -o $(BUILDDIR)

$(BUILD_TARGETS): go.sum $(BUILDDIR)/
	CGO_CFLAGS="-O -D__BLST_PORTABLE__" go $@ -mod=readonly $(BUILD_FLAGS) $(BUILD_ARGS) ./...

$(BUILDDIR)/:
	mkdir -p $(BUILDDIR)/

build-docker:
	$(DOCKER) build --tag altlayer/blitz -f Dockerfile \
		$(shell git rev-parse --show-toplevel)

.PHONY: build build-docker

.PHONY: lint
lint:
	staticcheck ./...
	golangci-lint run

.PHONY: test
test:
	go test ./...

###############################################################################
###                                Protobuf                                 ###
###############################################################################

proto-all: proto-gen

proto-gen:
	make -C eotsmanager proto-gen
	make -C finality-provider proto-gen

.PHONY: proto-gen

mock-gen:
	mkdir -p $(MOCKS_DIR)
	$(MOCKGEN_CMD) -source=clientcontroller/api/interface.go -package mocks -destination $(MOCKS_DIR)/clientcontroller.go

.PHONY: mock-gen

update-changelog:
	@echo ./scripts/update_changelog.sh $(sinceTag) $(upcomingTag)
	./scripts/update_changelog.sh $(sinceTag) $(upcomingTag)

.PHONY: update-changelog
