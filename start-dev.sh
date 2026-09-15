#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
go build -o smeagol-wysiwyg .
exec ./smeagol-wysiwyg --host 0.0.0.0 testdata/vault
