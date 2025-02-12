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

async function main(hackatomWasmPath: string) {
  const gasPrice = GasPrice.fromString("0.00001ubbn");
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(alice.mnemonic, {
    prefix: "bbn",
  });
  const client = await SigningCosmWasmClient.connectWithSigner(
    rpcEndpoint,
    wallet,
  );

  // Upload contract
  const wasm = fs.readFileSync(hackatomWasmPath);
  const uploadFee = calculateFee(7000_000, gasPrice);
  const uploadReceipt = await client.upload(
    alice.address0,
    wasm,
    uploadFee,
    "Upload contract",
  );
  console.info("Upload succeeded. Receipt:", uploadReceipt);

  // Instantiate
  const instantiateFee = calculateFee(300_000, gasPrice);
  // This contract specific message is passed to the contract
  const msg = {
    admin: admin,
    consumer_id: "test1",
    activated_height: 10,
    is_enabled: true
  };
  const { contractAddress, events } = await client.instantiate(
    alice.address0,
    uploadReceipt.codeId,
    msg,
    "nitro_finality_gadget instance",
    instantiateFee,
    { memo: `Create a instance` },
  );

  console.info(JSON.stringify(events, null, 2));

  console.info(`Contract instantiated at: `, contractAddress);
}

const repoRoot = process.cwd();
const hackatom = `${repoRoot}/target/wasm32-unknown-unknown/release/nitro_finality_gadget.wasm`;
await main(hackatom);
console.info("The show is over.");
