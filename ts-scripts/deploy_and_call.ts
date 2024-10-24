import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import { DirectSecp256k1HdWallet } from "@cosmjs/proto-signing";
import { calculateFee, GasPrice } from "@cosmjs/stargate";

// let ProofJSON = require("./scripts/res/proof.json");

import { fromHex } from "@cosmjs/encoding";

import * as fs from "fs";

// bbn14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9sw76fy2
const rpcEndpoint = "http://10.1.1.49:26657";

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
    admin: alice.address0,
    params: {
      covenant_pks: [],
      covenant_quorum: 1,
      btc_network: "regtest",
      max_active_finality_providers: 100,
      min_pub_rand: 1,
      min_slashing_tx_fee_sat: 1000,
      slashing_rate: "String::from(\"0.1\")",
      slashing_address: "String::from(\"n4cV57jePmAAue2WTTBQzH3k3R2rgWBQwY\")"
    }
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

  // Execute contract

  const executeFee = calculateFee(755_000, gasPrice);
  const result = await client.execute(
    alice.address0,
    contractAddress,
    {
      btc_staking: {
        new_fp: [{
          commission: "0.00",
          addr: 'bbn1tvt7058xrce9r3helj9ym0xa08jvaedd543pdm',
          btc_pk_hex: '28252efa5097e0b007dca2d11308c5670e6e822ba39d8dd0ab0c88111ec2b7e3',
          consumer_id: 'test1'
        }],
        active_del: [],
        slashed_del: [],
        unbonded_del: [],
      },
    },
    executeFee,
  );

  console.info(result);
  console.info(JSON.stringify(result.events, null, 2));


  console.info(`Contract instantiated at: `, contractAddress);
}

const repoRoot = process.cwd();
const hackatom = `${repoRoot}/contracts/wasm/btc_staking.wasm`;
await main(hackatom);
console.info("The show is over.");
