# BLITZ: A Fast Finality Network for Arbitrum Orbit Chains using Babylon

This repo is an implementation of BLITZ — a fast finality network for Arbitrum Orbit chains that leverages BTC staking through Babylon. 

## Architecture

![Overview](<docs/images/architecture.png>)

## Dependencies

### Golang / Rust

The project requires Go version 1.21 and Rust `stable` version. You can install Go by following https://go.dev/doc/install, and install Rust by following https://www.rust-lang.org/tools/install.

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

The finality-gadget-operator can then be found in `./build/`.

## Build Docker

```bash
docker build . -t blitz-opt
```

## Usage

### Deploy orbit finality gadget contract

For this, use call:

```bash
yarn test-deploy
```

This will return the contract address.

### Register Consumer Finality Provider

Can see [Register Consumer Finality Provider](./docs/registrator.md)

### Boot finality provider for Orbit By Docker

change configs in:

- `docker/configs/finality-gadget-operator.yaml`
- `docker/configs/fpd.conf`

configs by contract address and fp address

Then we can initiate the database for the finality provider:

```bash
./build/finality-gadget-operator --config docker/configs/finality-gadget-operator.yaml fps restore {keyname} {btcpubkey}
```

then boot by

```bash
docker compose up
```

### Boot finality provider for Orbit using bin

Boot eotsd in finality-provider:

```bash
./build/eotsd start
```

Then boot blitz service:

```bash
 ./build/finality-gadget-operator --config finality-gadget-operator.yaml
```
