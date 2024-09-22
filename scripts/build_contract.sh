#!/bin/bash

cargo build --release --target=wasm32-unknown-unknown --lib

# Use wasm-opt to optimizer, can move to build.rs in future
wasm-opt -O --signext-lowering -o ./target/wasm32-unknown-unknown/release/nitro_finality_gadget.wasm ./target/wasm32-unknown-unknown/release/nitro_finality_gadget.wasm 

# Use llvm-strip to delete the debug infos, but we keep the func name
# can use `twiggy top -n 30 ./bin/test-wasm/res/eth-validation.wasm` to show the size.
llvm-strip --keep-section=name ./target/wasm32-unknown-unknown/release/nitro_finality_gadget.wasm

