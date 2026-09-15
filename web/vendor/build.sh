#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

npm install --no-save @milkdown/core@7.3.6 @milkdown/preset-commonmark@7.3.6 @milkdown/plugin-listener@7.3.6 2>&1

npx esbuild entry.js \
  --bundle \
  --format=esm \
  --platform=browser \
  --target=es2022 \
  --outfile=milkdown.js \
  --minify

echo "Done: $(wc -c < milkdown.js) bytes"
