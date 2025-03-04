# How to run a finality provider operator node

## About Finality provider

For finality provider and consumer finality provider, can see:

- [finality-provider](https://github.com/babylonlabs-io/finality-provider)
- [Finality Provider Operation](https://github.com/babylonlabs-io/finality-provider/blob/main/docs/finality-provider-operation.md)

TODO: Babylon had no newest doc for public, :-(

## How to be a Consumer finality provider

Can see [Register Consumer Finality Provider](./registrator.md)

### Boot finality provider for Orbit By Docker

change configs in:

- `docker/configs/finality-gadget-operator.yaml`
- `docker/configs/fpd.conf`

configs by contract address and fp address:

In finality-gadget-operator.yaml:

```yaml
# fp_addr is the bech32 chain address identifier of the finality provider.
fp_addr: "bbn1vd74qq3m49605j4k7ltnpxx3rweqvukcfp3esf"

# btc_pk is the BTC secp256k1 PK of the finality provider encoded in BIP-340 spec
btc_pk: "0x1648cb2885f24b25df13d49641cf9af8ebaece3753269e7d6ee33982953fda0b"
```

For fpd.conf:

```yaml
; the contract address of the op-finality-gadget
OPFinalityGadgetAddress = bbn1dvepyy7s2nkfep05c4v6tfkmzqyvz7x3nj6ddj3kkr8nfsmmylhq6a5yp4
```

Then we should init the database for fp:

```bash
./build/finality-gadget-operator --config docker/configs/finality-gadget-operator.yaml fps restore {keyname} {btcpubkey}
```

then boot by

```bash
docker compose up
```

### Boot finality provider for Orbit By bin

First, we should perpare the configurations.

In finality-gadget-operator.yaml:

```yaml
finalityProviderHomePath: "/fpd/"

# fp_addr is the bech32 chain address identifier of the finality provider.
fp_addr: "bbn1vd74qq3m49605j4k7ltnpxx3rweqvukcfp3esf"

# btc_pk is the BTC secp256k1 PK of the finality provider encoded in BIP-340 spec
btc_pk: "0x1648cb2885f24b25df13d49641cf9af8ebaece3753269e7d6ee33982953fda0b"

```

the `finalityProviderHomePath` should be the path to register finality provider, it will contain this files:

```yaml
keyring-test/
data/
logs/
fpd.conf
finality-gadget-operator.yaml
```

For fpd.conf, it same as the finality-provider 's config, note we will use the config in `opstackl2`, we need change this configuration:

```yaml
; the contract address of the op-finality-gadget
OPFinalityGadgetAddress = bbn1dvepyy7s2nkfep05c4v6tfkmzqyvz7x3nj6ddj3kkr8nfsmmylhq6a5yp4
```

Boot eotsd in finality-provider:

```bash
./build/eotsd start
```

Then boot blitz service:

```bash
 ./build/finality-gadget-operator --config finality-gadget-operator.yaml
```
