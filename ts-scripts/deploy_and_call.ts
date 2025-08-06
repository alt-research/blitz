import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { calculateFee, GasPrice } from "@cosmjs/stargate";
import 'dotenv/config';

import * as fs from "fs";

const rpcEndpoint = "https://rpc.edge-devnet.babylonlabs.io:443";

let mnemonic = process.env.BBN_DEPLOY_MNEMONIC || ""
let address = process.env.BBN_DEPLOY_ADDRESS || ""

console.log("use " + address)

const alice = {
  mnemonic: mnemonic,
  address0: address,
};

const admin = address

async function main(hackatomWasmPath: string) {
  const gasPrice = GasPrice.fromString("0.002ubbn");
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
    bsn_id: "dodo-testnet-53457",
    is_enabled: true,
    min_pub_rand: 1000,
    rate_limiting_interval: 10000,
    max_msgs_per_interval: 10,
    bsn_activation_height: 0,
    finality_signature_interval: 1
  };
  const { contractAddress, events } = await client.instantiate(
    alice.address0,
    uploadReceipt.codeId,
    msg,
    "nitro finality contract instance",
    instantiateFee,
    { memo: `Create a instance for nitro finality contract` },
  );

  console.info(JSON.stringify(events, null, 2));

  console.info(`Contract instantiated at: `, contractAddress);
}

const repoRoot = process.cwd();
const code = `${repoRoot}/depends/rollup-bsn-contracts/artifacts/finality.wasm`;
await main(code);
console.info("The show is over.");
