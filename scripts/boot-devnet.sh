#!/bin/bash

export BABYLON_HOME="/main/babylon"

export BABYLON_CHAIN_HOME="$HOME/.blitz/babylon-test/test01"
export CHAIN_ID="my-testnet-01"

rm -rf $BABYLON_CHAIN_HOME
mkdir -p $BABYLON_CHAIN_HOME

$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME init my-node --chain-id $CHAIN_ID
$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME prepare-genesis testnet $CHAIN_ID
$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME --keyring-backend test keys add main
$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME --keyring-backend test keys add validator
$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME add-genesis-account $($BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME --keyring-backend test keys show main -a) 1824636766368ubbn
$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME add-genesis-account $($BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME --keyring-backend test keys show validator -a) 1824636766368ubbn
$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME create-bls-key $($BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME --keyring-backend test keys show validator -a)
$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME gen-helpers create-bls
$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME --keyring-backend test gentx validator 1824636766367ubbn --chain-id $CHAIN_ID
$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME collect-gentxs

# got the bls key file name
blsfile=""
for file in $BABYLON_CHAIN_HOME/config/*; do
    if [[ -f "$file" && "${file##*/}" == gen-bls-* ]]; then
        blsfile="${file##*/}"
    fi
done

echo $blsfile

$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME gen-helpers add-bls $BABYLON_CHAIN_HOME/config/$blsfile
$BABYLON_HOME/build/babylond --home $BABYLON_CHAIN_HOME --keyring-backend test start
