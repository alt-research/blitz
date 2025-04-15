# Finality Rpc Provider

For blitz, we can use the json rpc to got the finality block number by blitz.

In `finality-gadget-operator.yaml` config:

```yaml
# Configuration for finality gadget operator.

###############################################################
# The Common configs ##########################################
###############################################################
common:
  rpc_server_ip_port_address: "0.0.0.0:8290"
  rpc_vhosts: ["*"]
  rpc_cors: []

```

To confirm the finality block, the operator need to connect btc, so we need config btc info in `finality-gadget-operator.yaml` config

```yaml
###############################################################
# The babylon configs ######################################
###############################################################
babylon:
  finality_gadget:
    # path to the DB file
    dbfilepath: "data.db"
    bbnchainid: "euphrates-0.6.0"
    bbnrpcaddress: "https://rpc-euphrates.devnet.babylonlabs.io:443"
    bitcoinrpchost: "10.1.1.120:38332"
    bitcoinrpcuser: "name"
    bitcoinrpcpass: "password"
    bitcoindisabletls: true
    fgcontractaddress: "bbn1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjqczkw9f"
```
