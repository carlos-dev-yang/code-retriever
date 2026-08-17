#!/usr/bin/env sh
set -eu

repo_root=$(git rev-parse --show-toplevel)
tool_source="$repo_root/tools/relationdiag/typescript-toolchain"
target="$repo_root/.cidx/test/tools/typescript-6.0.3"
target_parent="$repo_root/.cidx/test/tools"

case "$target" in "$repo_root"/.cidx/test/tools/*) ;; *) echo "unsafe target" >&2; exit 1 ;; esac
if [ -d "$target/node_modules/typescript" ]; then
  version=$(node -e "process.stdout.write(require('$target/node_modules/typescript/package.json').version)")
  [ "$version" = "6.0.3" ] || { echo "unexpected existing TypeScript version" >&2; exit 1; }
  exit 0
fi
mkdir -p "$target_parent"
stage=$(mktemp -d "$target_parent/.typescript-6.0.3.stage.XXXXXX")
cleanup() { rm -rf "$stage"; }
trap cleanup EXIT INT HUP TERM
cp "$tool_source/package.json" "$tool_source/package-lock.json" "$stage/"
npm ci --prefix "$stage" --ignore-scripts --no-audit --no-fund
version=$(node -e "process.stdout.write(require('$stage/node_modules/typescript/package.json').version)")
[ "$version" = "6.0.3" ] || { echo "materialized TypeScript version mismatch" >&2; exit 1; }
[ ! -e "$target" ] || { echo "toolchain target appeared during materialization" >&2; exit 1; }
mv "$stage" "$target"
trap - EXIT INT HUP TERM
