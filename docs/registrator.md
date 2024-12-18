# Register Consumer Finality Provider

## **Prerequisites**

To integrate with Euphrates 0.5.0, you need to install and set up the following software.[](https://docs.babylonchain.io/docs/user-guides/btc-staking-testnet/finality-providers/overview#prerequisites)

### Golang / Rust

This project requires Go version 1.21 and Rust `stable` version. You can install Go by following https://go.dev/doc/install, and install Rust by following https://www.rust-lang.org/tools/install.

Also install essential tools and packages that might be needed to compile and build the binaries. In Ubuntu / Debian systems:

```
sudo apt install build-essential
```

### Bitcoin

You can compile bitcoin v26.x from [source](https://github.com/bitcoin/bitcoin), or install one of the pre-built [binary images](https://bitcoincore.org/bin/bitcoin-core-26.2/) for your OS.

### Babylon node

Babylon node software provides CLI commands and queries for finality providers and BTC delegations. Note that you don’t need to run a Babylon node for integration purposes.

Babylon node software provides CLI commands and queries for finality providers and BTC delegations. Note that you don’t need to run a Babylon node for integration purposes.

```
git clone https://github.com/babylonlabs-io/babylon.git
```

Then, check out the `euphrates-0.5.0-rc.0` tag.

```
cd babylon
git checkout euphrates-0.5.0-rc.0
```

At the top-level directory of the project

```
make install

```

The above command will build and install the `babylond` binary to `$GOPATH/bin`.

### Finality provider

Finality providers are responsible for voting at a finality round on top of [CometBFT](https://github.com/cometbft/cometbft). Similar to any native PoS validator, a finality provider can receive voting power delegations from BTC stakers, and can earn commission from the staking rewards denominated in Babylon tokens.

To install finality provider software, clone the repository to your local machine from Github:

```
git clone https://github.com/babylonlabs-io/finality-provider.git
```

Then, check out the `euphrates-0.5.0-rc.0` tag.

```
cd finality-provider
git checkout euphrates-0.5.0-rc.0
```

At the top-level directory of the project

```
make install
```

The above command will build and install the following binaries to `$GOPATH/bin`:

- `fpd`: The daemon and CLI program for the finality-provider.
- `eotsd`: The daemon program for the EOTS manager.

### BTC staker

To get started, clone the repository to your local machine from Github, check out 

```
git clone https://github.com/babylonlabs-io/btc-staker.git
```

Then, check out the `euphrates-0.5.0-rc.0` tag.

```
cd btc-staker
git checkout euphrates-0.5.0-rc.0
```

At the top-level directory of the project

```
make install
```

The above command will build and install the following binaries to `$GOPATH/bin`:

- `stakerd`: The daemon program for the btc-staker
- `stakercli`: The CLI tool for interacting with stakerd.

### Accessing the Euphrates devnet

Let’s define the Babylon client environment to access the Euphrates devnet. Please notice that the below assumes / depends on the `bash` shell. Modify accordingly for other shells, or use `bash`:

```go
$ cat <<EOF >env_euphrates.sh
:

export binary="babylond"
export chainId="euphrates"
export homeDir="$HOME/.babylond"

export key="user"
export keyringBackend="--keyring-backend=test"
export feeToken="bbn"

export rpcUrl="https://rpc-euphrates.devnet.babylonlabs.io"
export nodeUrl="$rpcUrl"
export grpcUrl="grpc-euphrates.devnet.babylonlabs.io:443"
export faucetUrl="https://faucet-euphrates.devnet.babylonlabs.io"

alias babylond='babylond --home=$homeDir'
EOF
$ . ./env_euphrates.sh
```

### Getting test tokens

**Babylon BBN tokens.** 

1. Create a Babylon account by using

```bash
$ babylond keys add $key $keyringBackend
```

1. Then get some BBN test tokens. If you have access, you can by example contact Spyros Kekos (@Spyros) in BabylonLabs’s Slack workspace, or directly use the https://babylon-devnet.l2scan.co/faucet
Alternatively, you can try **hitting the faucet endpoint** directly:

```bash
$ curl $faucetUrl/claim \
  -H "Content-Type: multipart/form-data" \
  -d '{ "address": "<your_bbn_address>"}'
```

1. You can verify the account’s balance by using

```bash
$ babylond query bank balances <your_bbn_address> --node $nodeUrl
```

### Getting BTC test tokens

**Signet BTC.** The Euphrates devnet is connected to BTC Signet. There’s information on how to connect to Signet in the [BTC Signet wiki](https://en.bitcoin.it/wiki/Signet). After creating a wallet and a new BTC address in the signet network, you can visit Bitcoin’s signet faucets (e.g., [https://signetfaucet.com](https://signetfaucet.com/)) for signet BTC. Babylon also provides a BTC signet faucet in [Babylon discord server](https://discord.gg/babylonglobal).

### 4. Register your consumer system on Babylon

Starting with this version v0.5.0, the consumer system is registered automatically during IBC channel establishment. That’s what the `consumer_name` and `consumer_description` parameters of the babylon contract instantiation message are for.

**FAQs (these apply to all `babylond tx` commands):**

- If you encounter error like **`ERR** failure when running app err="rpc error: code = InvalidArgument desc = rpc error: code = InvalidArgument desc = Address cannot be empty: invalid request"` , try to add a flag `--from <key_name>` where `<key_name>` is the key name associated with your address.
- If you encounter error like `chain ID required but not specified`, try to add a flag `--chain-id <chain_id>` where `<chain_id>` can be obtained from [`https://rpc-euphrates.devnet.babylonlabs.io/block?height=1`](https://rpc-euphrates.devnet.babylonchain.io/block?height=1)

Then, you can query the Babylon node to see the blockchain is registered to Babylon.

```bash
$ babylond query btcstkconsumer registered-consumers -o json --node https://rpc-euphrates.devnet.babylonlabs.io
{
  "chain_ids": [
    "test-consumer-chain"
  ],
  "pagination": {
    "next_key": null,
    "total": "0"
  }
}
$ 
```

### 5. Become a finality provider for your consumer system

**Set up EOTS manager.** To become a finality provider, you need to set up a EOTS manager that manages the key pairs for your finality provider. Please follow steps at [EOTS Manager | Babylon Blockchain](https://docs.babylonchain.io/docs/user-guides/btc-staking-testnet/finality-providers/eots-manager) and adapt the configuration file attached in the appendix. 

**Set up finality provider daemon.** Then you can set up a finality provider. It will call EOTS manager for signing messages, and interact with Babylon. Please follow steps at [Finality Provider | Babylon Blockchain](https://docs.babylonchain.io/docs/user-guides/btc-staking-testnet/finality-providers/finality-provider) and adapt the configuration file attached in the appendix.

**Note**: Mind to use your Consumer system id as `chain-id` for the CLI commands. 

**Create and register your finality provider on Babylon.** After that, you can create a finality provider instance through the `fpd create-finality-provider` or `fpd cfp` command. The created instance is associated with a BTC public key which serves as its unique identifier and a Babylon account to which staking rewards will be directed.

```go
$ fpd create-finality-provider --key-name my-finality-provider --chain-id <your_chain_id> --moniker my-name
```

You can register a created finality provider in Babylon through the `fpd register-finality-provider`. The output contains the hash of the Babylon finality provider registration transaction.

```
$ fpd register-finality-provider --btc-pk d0fc4db48643fbb4339dc4bbf15f272411716b0d60f18bdfeb3861544bf5ef63
```

Then, you can query the Babylon node to see the finality provider is registered on Babylon

```go
$ babylond query btcstkconsumer finality-providers <your_chain_id>
{
  "finality_providers": [
    {
      "description": {
        "moniker": "Finality Provider 3",
        "identity": "",
        "website": "",
        "security_contact": "",
        "details": ""
      },
      "commission": "0.050000000000000000",
      "babylon_pk": {
        "key": "AugG2rD0aBx6Q4edjPRQ2w7eoRRYVV21zcmVP8O4qyAj"
      },
      "btc_pk": "1ef48bf0fa918ce2070b4e4952064a75e0f2ab914979ced622692fd708f23ae5",
      "pop": {
        "btc_sig_type": "BIP340",
        "babylon_sig": "tFcTCPmHM8mW6fCm2ApfGsngXK76jCFbxuCGe2OdxBJqjdBoONb6Gmuip0j18BaX5PJwNFGrm9l8RR9e8d31ww==",
        "btc_sig": "/Vg6uoW+DbgGk0xQhEXJH2nJQfigUzYP5nPPI2FUcmGCdNnqlww3yVipekiG/r6b02H3AbWFMQIkiPiy+uHHyw=="
      },
      "slashed_babylon_height": "0",
      "slashed_btc_height": "0",
      "height": "0",
      "voting_power": "0",
      "chain_id": "test-consumer-chain"
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "0"
  }
}
$ 
```

### 6. Stake BTC to your finality providers

**Set up BTC staker daemon.** To stake BTC to your finality providers, you need to set up a BTC staker daemon that generates Bitcoin staking transactions and sends staking requests to Babylon. Please follow steps at [Stake with BTC Staker CLI | Babylon Blockchain](https://docs.babylonchain.io/docs/user-guides/btc-staking-testnet/become-btc-staker) and adapt the configuation file attached in the appendix.

**Staking BTC to your finality providers.** When staking, specify the BTC public keys of finality providers using the `--finality-providers-pks` flag in the `stake` command. Note that **Babylon requires a BTC delegation to restake to at least 1 Babylon finality provider, apart from finality providers of other consumer systems.**

First, you could use the following command to list all Babylon finality providers.

```
$ stakercli daemon babylon-finality-providers
```

or 

```go
$ babylond query btcstaking finality-provider <fp_pk> --node $nodeUrl
```

Then, find the BTC address that has sufficient Bitcoin balance that you want to stake from.

```
stakercli daemon list-outputs
```

After that, stake Bitcoin to the finality provider of your choice. The `--staking-time` flag specifies the timelock of the staking transaction in BTC blocks. The `--staking-amount` flag specifies the amount in Satoshis to stake. For example, `<pk1>` could be the Bitcoin public key of your finality provider, while `<pk2>` is a Babylon finality provider of your choice.

```go
stakercli daemon stake \
  --staker-address <staker_btc_address> \
  --staking-amount 1000000 \
  --finality-providers-pks <babylon_fp_pk> \
  --finality-providers-pks <consumer_fp_pk> ... \
  --staking-time 10000 # ~70 days
```

Note that among public keys specified in `--finality-providers-pks`, at least one of them should be a Babylon finality provider. For example, you can let the BTC delegation to restake to a Babylon finality provider as well as a finality provider for your consumer system.

Then, you can query the Babylon node to see the BTC delegation

```go
babylond query btcstaking btc-delegations any --node https://rpc-euphrates.devnet.babylonlabs.io
{  "btc_delegations": [
    {
      "btc_pk": "6e25665ffcb10a82af5103263f4c1f33bb364244009bc4d4ef7696462de71cde",
      "fp_btc_pk_list": [
        "63b996ff158e5e2b82ed099817fb1b549b23648c2e43e96394ff691a8ff4128d",
        "1ef48bf0fa918ce2070b4e4952064a75e0f2ab914979ced622692fd708f23ae5"
      ],
      "start_height": "120",
      "end_height": "620",
      "total_sat": "1000000",
      "staking_tx_hex": "0100000000010107d3d802eb6e3d40552422cf4c4be38f429d4099102438470955dbb6a449041d0100000000ffffffff0240420f00000000002251204206b407838b11a232bc5d26c2e25661a7f93f52c21e9d0a4efea6d85022506ecf788b3b000000001600147cc587002b9a7181bd3ac3b228fb08e32af6320d0247304402202a6ca03e47c55a282483488d91b8b697d3a0b3fb4400370ca3038371745507a702203e79b45355512e2d1c200d699b07ab073bc0b5ad5a85a7e7f2a3ad1d0d21ee05012102a6069d1b48df8dc36024273f0f977d69b954afe6b570f875216a96f8636e538b00000000",
      "slashing_tx_hex": "01000000015507d254512ff96d318949a172122d93e31a13f312d44f17698b7ec481f4cbe10000000000ffffffff02a0860100000000001976a914010101010101010101010101010101010101010188acb8b70d00000000002251203d030fe574510e1b14d6314caf90fe13ae32bff13e1bf97d71df47d9d6f7e29e00000000",
      "delegator_slash_sig_hex": "2a69b7b478bb2d24dcb08f13bcd089998fd6f27a4ea9c016a5e26b224e039910d671c525e219a2c9a5fabaf9508f7b16641af3f73b7bf12a659304a061e31215",
      "covenant_sigs": [],
      "staking_output_idx": 0,
      "active": false,
      "status_desc": "PENDING",
      "unbonding_time": 3,
      "undelegation_response": {
        "unbonding_tx_hex": "02000000015507d254512ff96d318949a172122d93e31a13f312d44f17698b7ec481f4cbe10000000000ffffffff01ac300f0000000000225120151e497fb4a8e1a16db25eba8fa6858185085ab125161e31b6026d68b2de72f900000000",
        "delegator_unbonding_sig_hex": "",
        "covenant_unbonding_sig_list": [],
        "slashing_tx_hex": "01000000015402ecd010f7edc070f7aaac344ef98d4451879109f1d169e907f2003f36a94d0000000000ffffffff02de840100000000001976a914010101010101010101010101010101010101010188ace6a70d00000000002251203d030fe574510e1b14d6314caf90fe13ae32bff13e1bf97d71df47d9d6f7e29e00000000",
        "delegator_slashing_sig_hex": "3a0ade421330bf02bd8854543b90fb16895be79c0fb7d0c4e2367e630d50949e6d41d2bf48b3b8d024a6b9cf6e8a127177994e5ad90d4a6983bb5a7abd22a055",
        "covenant_slashing_sigs": []
      }
    }
  ],
  "pagination": {
    "next_key": null,
    "total": "0"
  }
}
```
