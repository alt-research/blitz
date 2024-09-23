#!/bin/bash

export BABYLON_HOME="/main/babylon"

export BABYLON_CHAIN_HOME="$HOME/.blitz/babylon-test/test01"
export CHAIN_ID="my-testnet-01"

$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME --keyring-backend test tx bank send -y $($BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME --keyring-backend test keys show main -a) bbn15l95xdshva75x4zycm5c59l64kre5j4x092vve 100000000ubbn --chain-id $CHAIN_ID

sleep 5

yarn test-deploy