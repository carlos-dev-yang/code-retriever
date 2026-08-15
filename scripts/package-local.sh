#!/usr/bin/env bash
# Build a locally verified darwin/arm64 archive. This is deliberately not a
# release publisher: it refuses an unlicensed or dirty source tree.
set -euo pipefail

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
cd "$root"
fail() { printf 'package-local: %s\n' "$*" >&2; exit 1; }

[ "$(uname -s)" = Darwin ] || fail 'only darwin is locally verified'
[ "$(uname -m)" = arm64 ] || fail 'only arm64 is locally verified'
command -v clang >/dev/null || fail 'clang is required for bundled grammars'
command -v sandbox-exec >/dev/null || fail 'sandbox-exec is required for offline verification'
command -v otool >/dev/null || fail 'otool is required for linkage evidence'
command -v lipo >/dev/null || fail 'lipo is required for architecture evidence'
command -v file >/dev/null || fail 'file is required for binary format evidence'
command -v shasum >/dev/null || fail 'shasum is required for checksums'
command -v python3 >/dev/null || fail 'python3 is required for structural manifest validation'
[ -f LICENSE ] || fail 'project LICENSE is absent; public redistribution requires an owner license choice'
git diff --quiet || fail 'working tree has tracked modifications; commit provenance first'
git diff --cached --quiet || fail 'index has staged modifications; commit provenance first'
[ -z "$(git ls-files --others --exclude-standard)" ] || fail 'working tree has untracked files; commit or remove them first'

version=${CIDX_VERSION:-"dev-$(git rev-parse --short=12 HEAD)"}
commit=$(git rev-parse HEAD)
target=darwin_arm64
dist="$root/dist"
mkdir -p "$dist"
archive="$dist/cidx_${version}_${target}.tar.gz"
[ ! -e "$archive" ] || fail "refusing to overwrite existing archive: $archive"
[ ! -e "$dist/checksums.txt" ] || fail 'refusing to overwrite existing checksums.txt'
stage=$(mktemp -d "$dist/.package.XXXXXX")
expected=$(mktemp "$stage/expected.XXXXXX")
actual=$(mktemp "$stage/actual.XXXXXX")
trap 'rm -rf "$stage" "$expected" "$actual"' EXIT
mkdir -p "$stage/THIRD_PARTY_LICENSES"

awk -F '\t' 'NF && $1 !~ /^#/ { print $1 "\t" $2 }' packaging/third-party-licenses.tsv | sort -u >"$expected"
GOPROXY=off go list -deps -f '{{with .Module}}{{.Path}}{{"\t"}}{{.Version}}{{end}}' ./cmd/cidx | awk 'NF && $1 != "cidx"' | sort -u >"$actual"
if ! diff -u "$expected" "$actual"; then
  fail 'linked module set differs from packaging/third-party-licenses.tsv; update license review first'
fi

module_cache=$(go env GOMODCACHE)
notice="$stage/THIRD_PARTY_NOTICES"
printf 'cidx third-party notices\n\n' >"$notice"
while IFS=$'\t' read -r module module_version source_files; do
  case "$module" in ''|'#'*) continue ;; esac
  name=$(printf '%s@%s' "$module" "$module_version" | tr '/@' '__')
  old_ifs=$IFS
  IFS=,
  for source_file in $source_files; do
    source_path="$module_cache/$module@$module_version/$source_file"
    [ -f "$source_path" ] || fail "missing cached notice: $module@$module_version/$source_file"
    copy_name=$(printf '%s-%s' "$name" "$source_file" | tr '/' '_')
    cp "$source_path" "$stage/THIRD_PARTY_LICENSES/$copy_name"
    source_sha=$(shasum -a 256 "$source_path" | awk '{print $1}')
    printf '%s %s — source=%s — THIRD_PARTY_LICENSES/%s — sha256=%s\n' "$module" "$module_version" "$source_file" "$copy_name" "$source_sha" >>"$notice"
  done
  IFS=$old_ifs
done < packaging/third-party-licenses.tsv
cp LICENSE "$stage/LICENSE"

binary="$stage/cidx"
GOPROXY=off CGO_ENABLED=1 CC=clang GOOS=darwin GOARCH=arm64 go build -trimpath -buildvcs=true \
  -ldflags "-X cidx/internal/buildinfo.Version=$version -X cidx/internal/buildinfo.Commit=$commit -X cidx/internal/buildinfo.CGOEnabled=1" \
  -o "$binary" ./cmd/cidx
chmod 0755 "$binary"
"$binary" version --json >"$stage/build-info.json"
python3 - "$stage/build-info.json" <<'PY'
import json, re, sys
v=json.load(open(sys.argv[1]))
need=lambda ok,msg: (_ for _ in ()).throw(SystemExit(msg)) if not ok else None
need(v.get("version") not in (None,"","unknown","devel"), "invalid build version")
need(re.fullmatch(r"[0-9a-f]{40}", v.get("commit", "")) is not None, "invalid build commit")
need(v.get("source_modified") == "false", "build provenance is modified")
need((v.get("target_os"),v.get("target_arch"),v.get("cgo_enabled")) == ("darwin","arm64","1"), "invalid target/CGO")
need(v.get("sqlite_implementation_id") == "modernc.org/sqlite" and v.get("sqlite_version", "").startswith("v"), "invalid SQLite identity")
need(all("@v" in x for x in v.get("grammar_implementation_ids", [])) and len(v.get("grammar_implementation_ids", [])) == 3, "invalid grammar identities")
r=v.get("runtime", {})
need(r.get("fts5_available") is True and r.get("wal_available") is True, "runtime FTS/WAL unavailable")
need(r.get("registered_languages") == ["go","typescript","tsx"], "runtime grammar registry mismatch")
need(r.get("production_schema_minimum") == 1 and r.get("production_schema_maximum") == v.get("production_schema_version"), "invalid schema range")
PY
otool -L "$binary" >"$stage/linkage.txt"
[ -s "$stage/linkage.txt" ] || fail 'Mach-O linkage inspection produced no evidence'
file "$binary" >"$stage/binary-format.txt"
lipo -archs "$binary" >"$stage/binary-architectures.txt"
grep -qw arm64 "$stage/binary-architectures.txt" || fail 'built binary is not arm64'
go version -m "$binary" >"$stage/go-version-m.txt"
awk '$1 == "dep" { print $2 "\t" $3 }' "$stage/go-version-m.txt" | sort -u >"$stage/linked-modules.tsv"
if ! diff -u "$expected" "$stage/linked-modules.tsv"; then
  fail 'built binary module metadata differs from packaging allowlist'
fi
if grep -Eiq 'libsqlite3|tree-sitter' "$stage/linkage.txt"; then
  fail 'binary dynamically links SQLite or Tree-sitter instead of using bundled dependencies'
fi
printf '{\n  "build_info": ' >"$stage/build-manifest.json"
tr -d '\n' <"$stage/build-info.json" >>"$stage/build-manifest.json"
printf ',\n  "archive_target": "%s",\n  "cgo_policy": "CGO_ENABLED=1 with clang; Tree-sitter grammars require CGO",\n  "linkage_evidence": "linkage.txt",\n  "static_linkage": "not claimed"\n}\n' "$target" >>"$stage/build-manifest.json"
python3 - "$stage/build-info.json" "$stage/build-manifest.json" <<'PY'
import json, sys
i=json.load(open(sys.argv[1])); m=json.load(open(sys.argv[2]))
if m.get("build_info") != i or m.get("archive_target") != "darwin_arm64": raise SystemExit("build manifest mismatch")
PY
mkdir -p "$stage/docs"
cp docs/install.md docs/hosts.md docs/hooks.md docs/upgrade.md "$stage/docs/"

tar -czf "$archive" -C "$stage" cidx LICENSE THIRD_PARTY_NOTICES THIRD_PARTY_LICENSES build-info.json build-manifest.json linkage.txt binary-format.txt binary-architectures.txt go-version-m.txt linked-modules.tsv docs
(cd "$dist" && shasum -a 256 "$(basename "$archive")") >"$dist/checksums.txt"
printf 'created %s\n' "$archive"
printf 'checksum manifest: %s\n' "$dist/checksums.txt"
