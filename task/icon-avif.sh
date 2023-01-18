#!/bin/bash -u
# This script convert all PNG icons to AVIF format.
# Requires Node.js 14.15.0+.
# see https://github.com/lovell/avif-cli

wd=$(realpath -s "$(dirname "$0")/../frontend/icon")
cd "$wd"

npx avif --input="chakram/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="delta/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="junior/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="oxygen/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="senary/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="senary2/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="tulliana/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="ubuntu/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="whistlepuff/*.png" --quality=50 --effort=9 --overwrite --verbose
npx avif --input="xrabbit/*.png" --quality=50 --effort=9 --overwrite --verbose

cd ~-
