import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { calculateFee, GasPrice } from "@cosmjs/stargate";
import 'dotenv/config';

import * as fs from "fs";

const rpcEndpoint = "https://rpc-euphrates.devnet.babylonlabs.io:443";

let mnemonic = process.env.BBN_DEPLOY_MNEMONIC || ""
let address = process.env.BBN_DEPLOY_ADDRESS || ""

console.log("use " + address)

// Just for test account, can got from https://babylon-devnet.l2scan.co/faucet
const alice = {
  mnemonic: mnemonic,
  address0: address,
};

const admin = address
const contractAddress = "bbn13wf4hwycv048vdcew4v58avutruhg4mzz3fuj66unydlzf8ctscqqx8gkv"

async function main() {
  const gasPrice = GasPrice.fromString("0.00001ubbn");
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(alice.mnemonic, {
    prefix: "bbn",
  });
  const client = await SigningCosmWasmClient.connectWithSigner(
    rpcEndpoint,
    wallet,
  );

  // Set image id
  // Execute contract
  const reset = calculateFee(3500_000, gasPrice);
  const resetResult = await client.execute(
    alice.address0,
    contractAddress,
    {
      reset: {}
    },
    reset,
  );

  console.info(JSON.stringify(resetResult.events, null, 2));
}

const repoRoot = process.cwd();
await main();
console.info("The show is over.");
