FROM golang:1.23.4-alpine AS builder

# Version to build. Default is the Git HEAD.
ARG VERSION="HEAD"

# Use muslc for static libs
ARG BUILD_TAGS="muslc"


RUN apk add --no-cache --update openssh git make build-base linux-headers libc-dev \
    pkgconfig zeromq-dev musl-dev alpine-sdk libsodium-dev \
    libzmq-static libsodium-static gcc


# Build
WORKDIR /go/src/github.com/alt-research/blitz
# Cache dependencies
COPY go.mod go.sum /go/src/github.com/alt-research/blitz/
RUN go mod download
# Copy the rest of the files
COPY ./ /go/src/github.com/alt-research/blitz/

# Cosmwasm - Download correct libwasmvm version
RUN WASMVM_VERSION=$(grep github.com/CosmWasm/wasmvm go.mod | cut -d' ' -f2) && \
    wget https://github.com/CosmWasm/wasmvm/releases/download/$WASMVM_VERSION/libwasmvm_muslc.$(uname -m).a \
    -O /lib/libwasmvm_muslc.$(uname -m).a && \
    # verify checksum
    wget https://github.com/CosmWasm/wasmvm/releases/download/$WASMVM_VERSION/checksums.txt -O /tmp/checksums.txt && \
    sha256sum /lib/libwasmvm_muslc.$(uname -m).a | grep $(cat /tmp/checksums.txt | grep libwasmvm_muslc.$(uname -m) | cut -d ' ' -f 1)

RUN CGO_LDFLAGS="$CGO_LDFLAGS -lstdc++ -lm -lsodium" \
    CGO_ENABLED=1 \
    BUILD_TAGS=$BUILD_TAGS \
    LINK_STATICALLY=true \
    make build

# FINAL IMAGE
FROM alpine:3.16 AS app

RUN apk add bash curl jq

COPY --from=builder /go/src/github.com/alt-research/blitz/build/finality-gadget-operator /bin/finality-gadget-operator
COPY --from=builder /go/src/github.com/alt-research/blitz/build/finality-gadget-signer /bin/finality-gadget-signer

ENTRYPOINT [ "finality-gadget-operator" ]