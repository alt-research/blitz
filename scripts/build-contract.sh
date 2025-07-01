#!/bin/bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

cd $SCRIPT_DIR/../depends/rollup-bsn-contracts/ && bash scripts/optimizer.sh 