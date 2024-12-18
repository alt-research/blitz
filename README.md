# BLITZ: A Fast Finality Network for Arbitrum Orbit Chains using Babylon

We describe the design of BLITZ — a fast finality network for Arbitrum Orbit chains that leverages BTC staking through Babylon. To make BLITZ a reality, we also present a grant proposal to build BLITZ.

## Architecture

![Overview](<docs/images/architecture.png>)

## Dependencies

### Golang / Rust

This project requires Go version 1.21 and Rust `stable` version. You can install Go by following https://go.dev/doc/install, and install Rust by following https://www.rust-lang.org/tools/install.

Also install essential tools and packages that might be needed to compile and build the binaries. In Ubuntu / Debian systems:

```bash
sudo apt install build-essential
```

## Build

The contract:

```bash
bash ./scripts/build-contract.sh 
```

The wasm code will in:

```bash
./target/wasm32-unknown-unknown/release/nitro_finality_gadget.wasm 
```

The finality gadget:

```bash
make build
```

We can got the finality-gadget-operator in `./build/`.

## Build Docker

```bash
docker build . -t blitz-opt
```

## Usage

### Deploy orbit finality gadget contract

we can use call:

```bash
yarn test-deploy
```

we can got contract address in return.

### Register Consumer Finality Provider

Can see [Register Consumer Finality Provider](./docs/registrator.md)

### Boot finality provider for Orbit By Docker

change configs in:

- `docker/configs/finality-gadget-operator.yaml`
- `docker/configs/fpd.conf`

configs by contract address and fp address

then boot by

```bash
docker compose up
```

### Boot finality provider for Orbit By bin

Boot eotsd in finality-provider:

```bash
./build/eotsd start
```

Then boot blitz service:

```bash
 ./build/finality-gadget-operator --config finality-gadget-operator.yaml
```
