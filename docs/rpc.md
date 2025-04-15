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

