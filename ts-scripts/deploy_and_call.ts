import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { calculateFee, GasPrice } from "@cosmjs/stargate";

// let ProofJSON = require("./scripts/res/proof.json");

import { fromHex } from "@cosmjs/encoding";

import * as fs from "fs";

// bbn14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sw76fy2
const rpcEndpoint = "http://10.1.1.39:26657";

// Example user from scripts/wasmd/README.md
const alice = {
  mnemonic:
    "chest flight nose manual safe table cliff boss stereo inflict gap mountain beyond cherry music obscure virtual risk edit slice sail narrow behave ceiling",
  address0: "bbn15l95xdshva75x4zycm5c59l64kre5j4x092vve",
};

async function main(hackatomWasmPath: string) {
  const gasPrice = GasPrice.fromString("0.005ubbn");
  const wallet = await DirectSecp256k1HdWallet.fromMnemonic(alice.mnemonic, {
    prefix: "bbn",
  });
  const client = await SigningCosmWasmClient.connectWithSigner(
    rpcEndpoint,
    wallet,
  );

  // Upload contract
  const wasm = fs.readFileSync(hackatomWasmPath);
  const uploadFee = calculateFee(4000_000, gasPrice);
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
    admin: alice.address0,
    consumer_id: "test1",
    activated_height: 1,
    is_enabled: true
  };
  const { contractAddress, events } = await client.instantiate(
    alice.address0,
    uploadReceipt.codeId,
    msg,
    "My instance",
    instantiateFee,
    { memo: `Create a hackatom instance` },
  );

  console.info(JSON.stringify(events, null, 2));
  console.info(`Contract instantiated at: `, contractAddress);
}

const repoRoot = process.cwd();
const hackatom = `${repoRoot}/target/wasm32-unknown-unknown/release/nitro_finality_gadget.wasm`;
await main(hackatom);
console.info("The show is over.");
