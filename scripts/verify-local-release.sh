#!/usr/bin/env bash
# Verify one local archive without network access or user host configuration.
set -euo pipefail

[ $# -ge 1 ] || { printf 'usage: %s <archive> [checksums.txt]\n' "$0" >&2; exit 2; }
archive=$(CDPATH= cd -- "$(dirname -- "$1")" && pwd -P)/$(basename -- "$1")
checksums=${2:-"$(dirname -- "$archive")/checksums.txt"}
fail() { printf 'verify-local-release: %s\n' "$*" >&2; exit 1; }
[ "$(uname -s)" = Darwin ] || fail 'only darwin is locally verified'
[ "$(uname -m)" = arm64 ] || fail 'only arm64 is locally verified'
command -v sandbox-exec >/dev/null || fail 'sandbox-exec is unavailable; offline verification is unverified'
command -v git >/dev/null || fail 'git is required for the synthetic repository'
command -v sqlite3 >/dev/null || fail 'sqlite3 is required for the newer-schema negative check'
command -v codex >/dev/null || fail 'codex CLI is required for isolated project-config verification'
command -v python3 >/dev/null || fail 'python3 is required for structural transcript validation'
[ -f "$archive" ] && [ -f "$checksums" ] || fail 'archive or checksum manifest is absent'

profile='(version 1) (deny network*) (allow default)'
offline() { env -u VOYAGE_API_KEY GOPROXY=off sandbox-exec -p "$profile" "$@"; }
work=$(mktemp -d "${TMPDIR:-/tmp}/cidx-release.XXXXXX")
work=$(cd "$work" && pwd -P)
mcp_fd_open=0
codex_fd_open=0
cidx_server_pid=
codex_server_pid=
is_owned_live_child() {
  local pid=$1 parent state
  [ -n "$pid" ] || return 1
  read -r parent state < <(ps -o ppid= -o stat= -p "$pid" 2>/dev/null) || return 1
  [ "$parent" = "$$" ] || return 1
  case "$state" in Z*|*Z*) return 1 ;; esac
}
close_owned_fifo_fds() {
  if [ "$mcp_fd_open" -eq 1 ]; then
    exec 3>&-
    mcp_fd_open=0
  fi
  if [ "$codex_fd_open" -eq 1 ]; then
    exec 4>&-
    codex_fd_open=0
  fi
}
reap_owned_child() {
  local pid=$1 label=$2 term_ticks=${3:-20} kill_ticks=${4:-20}
  [ -n "$pid" ] || return 0
  if is_owned_live_child "$pid"; then
    kill -TERM "$pid" 2>/dev/null || true
    while is_owned_live_child "$pid" && [ "$term_ticks" -gt 0 ]; do
      sleep 0.1
      term_ticks=$((term_ticks - 1))
    done
    if is_owned_live_child "$pid"; then
      kill -KILL "$pid" 2>/dev/null || true
      while is_owned_live_child "$pid" && [ "$kill_ticks" -gt 0 ]; do
        sleep 0.1
        kill_ticks=$((kill_ticks - 1))
      done
    fi
  fi
  if is_owned_live_child "$pid"; then
    printf 'verify-local-release: owned %s process survived bounded cleanup\n' "$label" >&2
    return 0
  fi
  wait "$pid" 2>/dev/null || true
  printf 'verify-local-release: reaped owned %s process after failure\n' "$label" >&2
}
await_owned_shutdown() {
  local pid=$1 label=$2 graceful_ticks=${3:-50} terminate_ticks=${4:-20} kill_ticks=${5:-20}
  while is_owned_live_child "$pid" && [ "$graceful_ticks" -gt 0 ]; do
    sleep 0.1
    graceful_ticks=$((graceful_ticks - 1))
  done
  if is_owned_live_child "$pid"; then
    printf 'verify-local-release: %s did not stop after stdin closed; terminating owned child\n' "$label" >&2
    kill -TERM "$pid" 2>/dev/null || true
    while is_owned_live_child "$pid" && [ "$terminate_ticks" -gt 0 ]; do
      sleep 0.1
      terminate_ticks=$((terminate_ticks - 1))
    done
    if is_owned_live_child "$pid"; then
      printf 'verify-local-release: %s ignored TERM; killing owned child\n' "$label" >&2
      kill -KILL "$pid" 2>/dev/null || true
      while is_owned_live_child "$pid" && [ "$kill_ticks" -gt 0 ]; do
        sleep 0.1
        kill_ticks=$((kill_ticks - 1))
      done
    fi
    is_owned_live_child "$pid" && fail "$label did not stop after bounded shutdown"
  fi
  wait "$pid"
}
copy_evidence() {
  local destination=$1 artifact
  mkdir -p "$destination"
  for artifact in \
    version.json config-1024.json embed-plan-1024.json embed-1024.json status-1024.json \
    config-512.json index-512.json embed-plan-512.json embed-512.json status.json index.json index-a.json index-b.json \
    retired-256.stdout retired-256.stderr retired-binary.stdout retired-binary.stderr \
    mcp.jsonl mcp.stderr mcp-assertion.stderr \
    codex-app-server.jsonl codex-app-server.stderr codex-version.txt codex-version.stderr \
    relocated-status.json root-mismatch.stdout root-mismatch.stderr newer-schema.stdout newer-schema.stderr \
    invalid-config.stdout invalid-config.stderr; do
    if [ -f "$work/$artifact" ]; then
      cp "$work/$artifact" "$destination/" || return 1
    fi
  done
  return 0
}
cleanup() {
  local exit_code=$?
  trap - EXIT
  set +e
  close_owned_fifo_fds
  [ -z "$cidx_server_pid" ] || reap_owned_child "$cidx_server_pid" 'cidx server'
  [ -z "$codex_server_pid" ] || reap_owned_child "$codex_server_pid" 'Codex app-server'
  if [ "$exit_code" -ne 0 ] && [ -n "${CIDX_EVIDENCE_DIR:-}" ]; then
    if copy_evidence "$CIDX_EVIDENCE_DIR"; then
      printf 'local verifier failed; partial transcripts copied to %s\n' "$CIDX_EVIDENCE_DIR" >&2
    else
      printf 'local verifier failed; unable to copy partial transcripts to %s\n' "$CIDX_EVIDENCE_DIR" >&2
    fi
  fi
  rm -rf "$work"
  exit "$exit_code"
}
trap cleanup EXIT
archive_name=$(basename -- "$archive")
cp "$archive" "$work/$archive_name"
cp "$checksums" "$work/checksums.txt"
(cd "$work" && shasum -a 256 -c checksums.txt)
python3 - "$work/$archive_name" <<'PY'
import re,tarfile,sys
with tarfile.open(sys.argv[1], "r:gz") as bundle:
    members=bundle.getmembers()
    if not members:
        raise SystemExit("archive has no members")
    for member in members:
        if (member.uid, member.gid, member.uname, member.gname, member.mtime) != (0, 0, "root", "root", 0):
            raise SystemExit(f"non-neutral archive metadata: {member.name}")
    by_name={member.name:member for member in members}
    if by_name.get("cidx") is None or by_name["cidx"].mode & 0o777 != 0o755:
        raise SystemExit("archive did not preserve cidx executable mode")
    if by_name.get("LICENSE") is None or by_name["LICENSE"].mode & 0o777 != 0o644:
        raise SystemExit("archive did not preserve LICENSE mode")
    for name in ("linkage.txt", "binary-format.txt", "go-version-m.txt"):
        text=bundle.extractfile(name).read().decode()
        first=text.splitlines()[0] if text else ""
        if "/.package." in text or not re.match(r"^cidx(?: \([^\r\n]*\))?:", first):
            raise SystemExit(f"archive diagnostic is not portable: {name}")
PY
cp "$work/$archive_name" "$work/corrupt.tar.gz"
printf 'corrupt' >>"$work/corrupt.tar.gz"
if (cd "$work" && shasum -a 256 -c <(sed 's|cidx_.*\.tar\.gz|corrupt.tar.gz|' checksums.txt)); then
  fail 'corrupt archive unexpectedly verified'
fi
mkdir "$work/unpack"
tar -xzf "$work/$archive_name" -C "$work/unpack"
binary="$work/unpack/cidx"
[ -x "$binary" ] || fail 'archive lacks executable cidx'
chmod a-x "$binary"
if "$binary" version >/dev/null 2>&1; then
  fail 'non-executable artifact unexpectedly ran'
fi
chmod 0755 "$binary"
offline "$binary" version --json >"$work/version.json"
python3 - "$work/unpack/build-manifest.json" "$work/version.json" <<'PY'
import json,sys
m=json.load(open(sys.argv[1])); v=json.load(open(sys.argv[2])); r=v["runtime"]
assert m["build_info"] == v and m["archive_target"] == "darwin_arm64"
assert (v["target_os"],v["target_arch"],v["cgo_enabled"]) == ("darwin","arm64","1")
assert r["fts5_available"] and r["wal_available"] and r["registered_languages"] == ["go","typescript","tsx"]
assert r["production_schema_minimum"] == 1 and r["production_schema_maximum"] == v["production_schema_version"]
PY
for document in install.md hosts.md hooks.md upgrade.md; do
  [ -f "$work/unpack/docs/$document" ] || fail "archive missing docs/$document"
done

repo="$work/cidx phase14 한글"
mkdir -p "$repo"
offline git -C "$repo" init >/dev/null
printf 'package sample\nfunc Hello() string { return "hello" }\n' >"$repo/main.go"
printf 'export function hello(): string { return "hello" }\n' >"$repo/sample.ts"
printf 'export const View = () => <div>hello</div>\n' >"$repo/sample.tsx"
(cd "$repo" && offline "$binary" init)
cp "$repo/.cidx/config.json" "$work/config-1024.json"
python3 - "$work/config-1024.json" <<'PY'
import json,sys
c=json.load(open(sys.argv[1]))
assert c["embedding"]["serving_dimensions"] == 1024
assert c["embedding"]["storage_codec"] == "int8"
PY
(cd "$repo" && offline "$binary" index --reason manual >"$work/index.json")
[ ! -e "$repo/.cidx/db/embeddings.db" ] || fail 'free init/index unexpectedly created product source bank'

# Build a provider-free synthetic source bank for the indexed canonical inputs.
# These deterministic nonzero vectors are verifier fixtures, not provider or
# retrieval-quality evidence.
offline python3 - "$repo/.cidx/db/index.db" "$repo/.cidx/db/embeddings.db" <<'PY'
import binascii,hashlib,os,sqlite3,struct,sys
production_path,bank_path=sys.argv[1:]
production=sqlite3.connect(f"file:{production_path}?mode=ro", uri=True)
source_profile=production.execute("SELECT source_profile FROM meta WHERE id=1").fetchone()[0]
input_hashes=[row[0] for row in production.execute("SELECT DISTINCT canonical_input_sha256 FROM embedding_segments ORDER BY canonical_input_sha256")]
production.close()
assert len(source_profile) == 64 and input_hashes
bank=sqlite3.connect(bank_path)
bank.executescript("""
CREATE TABLE source_meta (id INTEGER PRIMARY KEY CHECK(id=1), schema_version INTEGER NOT NULL CHECK(schema_version=1), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')));
CREATE TABLE document_source_embeddings (source_profile_fingerprint TEXT NOT NULL, canonical_input_sha256 TEXT NOT NULL, dimensions INTEGER NOT NULL CHECK(dimensions=1024), checksum INTEGER NOT NULL, vector_f32_le BLOB NOT NULL CHECK(length(vector_f32_le)=4096), vector_sha256 TEXT NOT NULL, requested_model TEXT NOT NULL, response_model TEXT NOT NULL, request_id TEXT NOT NULL DEFAULT '', encoding TEXT NOT NULL CHECK(encoding='cidx-source-f32-le-v1'), created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')), PRIMARY KEY(source_profile_fingerprint,canonical_input_sha256));
INSERT INTO source_meta(id,schema_version) VALUES(1,1);
PRAGMA user_version=1;
""")
for ordinal,input_hash in enumerate(input_hashes):
    values=[0.0]*1024
    values[0]=1.0
    values[1+(ordinal%511)]=0.5
    blob=struct.pack("<1024f", *values)
    bank.execute("INSERT INTO document_source_embeddings(source_profile_fingerprint,canonical_input_sha256,dimensions,checksum,vector_f32_le,vector_sha256,requested_model,response_model,request_id,encoding) VALUES(?,?,?,?,?,?,?,?,?,?)", (source_profile,input_hash,1024,binascii.crc32(blob)&0xffffffff,blob,hashlib.sha256(blob).hexdigest(),"voyage-code-4","voyage-code-4","phase14-offline-fixture","cidx-source-f32-le-v1"))
bank.commit()
bank.close()
os.chmod(bank_path,0o600)
PY
(cd "$repo" && offline "$binary" embed --dry-run >"$work/embed-plan-1024.json")
(cd "$repo" && offline "$binary" embed --apply >"$work/embed-1024.json")
(cd "$repo" && offline "$binary" status --json >"$work/status-1024.json")
python3 - "$work/embed-plan-1024.json" "$work/embed-1024.json" "$work/status-1024.json" <<'PY'
import json,sys
plan,result,status=map(lambda p:json.load(open(p)),sys.argv[1:])
assert plan["source_input_count"] > 0 and plan["voyage_input_count"] == 0
assert result["requested_count"] == result["actual_tokens"] == result["failed_count"] == result["discarded_count"] == 0
assert result["succeeded_count"] == plan["source_input_count"]
assert status["vector_coverage_numerator"] == status["vector_coverage_denominator"] > 0
PY

# Switch only the selected serving dimension, reconcile, and reuse the same
# source-1024 rows locally. No provider key or network is available here.
python3 - "$repo/.cidx/config.json" "$work/config-512.json" <<'PY'
import json,sys
path,out=sys.argv[1:]
c=json.load(open(path)); c["embedding"]["serving_dimensions"]=512
with open(path,"w") as f: json.dump(c,f,indent=2); f.write("\n")
with open(out,"w") as f: json.dump(c,f,indent=2); f.write("\n")
assert c["embedding"]["storage_codec"] == "int8"
PY
(cd "$repo" && offline "$binary" index --reason manual >"$work/index-512.json")
(cd "$repo" && offline "$binary" embed --dry-run >"$work/embed-plan-512.json")
(cd "$repo" && offline "$binary" embed --apply >"$work/embed-512.json")
(cd "$repo" && offline "$binary" status --json >"$work/status.json")
offline python3 - "$repo/.cidx/db/index.db" "$work/embed-plan-512.json" "$work/embed-512.json" "$work/status.json" <<'PY'
import json,sqlite3,sys
db_path,plan_path,result_path,status_path=sys.argv[1:]
plan,result,status=[json.load(open(p)) for p in (plan_path,result_path,status_path)]
assert plan["source_input_count"] > 0 and plan["voyage_input_count"] == 0
assert result["requested_count"] == result["actual_tokens"] == result["failed_count"] == result["discarded_count"] == 0
assert result["succeeded_count"] == plan["source_input_count"]
assert status["vector_coverage_numerator"] == status["vector_coverage_denominator"] > 0
db=sqlite3.connect(f"file:{db_path}?mode=ro",uri=True)
active=db.execute("SELECT active_serving_profile FROM meta WHERE id=1").fetchone()[0]
rows=db.execute("SELECT DISTINCT dimensions,codec_id FROM vector_cache WHERE serving_profile=?",(active,)).fetchall()
db.close()
assert rows == [(512,"cidx-int8-symmetric-v1")]
PY

# Retired dimensions and codecs are negative checks only; failed init must not
# create repository state.
retired="$work/retired-profile"
mkdir "$retired"
offline git -C "$retired" init >/dev/null
if (cd "$retired" && offline "$binary" init --serving-dim 256 >"$work/retired-256.stdout" 2>"$work/retired-256.stderr"); then fail 'retired 256 dimension unexpectedly initialized'; fi
[ ! -e "$retired/.cidx" ] || fail 'retired 256 dimension mutated repository state'
if (cd "$retired" && offline "$binary" init --codec binary >"$work/retired-binary.stdout" 2>"$work/retired-binary.stderr"); then fail 'retired binary codec unexpectedly initialized'; fi
[ ! -e "$retired/.cidx" ] || fail 'retired binary codec mutated repository state'
(
  cd "$repo"
  offline "$binary" index --reason manual >"$work/index-a.json" &
  first=$!
  offline "$binary" index --reason manual >"$work/index-b.json" &
  second=$!
  wait "$first"
  wait "$second"
)
(cd "$repo" && offline "$binary" status --json >"$work/status.json")
[ -f "$repo/.cidx/db/embeddings.db" ] || fail 'provider-free rematerialization lost product source bank'
[ ! -d "$repo/.cidx/test" ] || fail 'normal runtime created development workspace state'
if find "$repo" -type f \( -name '*.wasm' -o -name '*.onnx' -o -name '*.gguf' \) -print -quit | grep -q .; then
  fail 'runtime created grammar or model download artifact'
fi

serve_repo="$work/cidx serve without source bank"
mkdir "$serve_repo"
offline git -C "$serve_repo" init >/dev/null
cp "$repo/main.go" "$repo/sample.ts" "$repo/sample.tsx" "$serve_repo/"
mkdir -p "$serve_repo/.cidx/db"
cp "$repo/.cidx/config.json" "$serve_repo/.cidx/config.json"
cp "$repo/.cidx/db/index.db" "$serve_repo/.cidx/db/index.db"
offline "$binary" status --root "$serve_repo" --json >"$work/relocated-status.json"
[ ! -e "$serve_repo/.cidx/db/embeddings.db" ] || fail 'source-bank-free serve fixture unexpectedly has product source state'

sha=$(shasum -a 256 "$serve_repo/main.go" | awk '{print $1}')
fifo="$work/mcp.stdin"
mkfifo "$fifo"
offline "$binary" serve --root "$serve_repo" <"$fifo" >"$work/mcp.jsonl" 2>"$work/mcp.stderr" &
cidx_server_pid=$!
exec 3>"$fifo"
mcp_fd_open=1
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"local-release","version":"1"}}}' >&3
for _ in $(seq 1 50); do
  grep -q '"id":1' "$work/mcp.jsonl" && break
  sleep 0.1
done
grep -q '"id":1' "$work/mcp.jsonl" || fail 'initialize response missing'
if grep '"id":1' "$work/mcp.jsonl" | grep -q '"error":'; then fail 'initialize failed'; fi
{
  printf '%s\n' '{"jsonrpc":"2.0","method":"notifications/initialized"}'
  printf '%s\n' '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}'
  printf '%s\n' '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"status","arguments":{}}}'
  printf '%s\n' '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"search","arguments":{"query":"Hello","mode":"fts","max_inline_bytes":0}}}'
  printf '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"read_span","arguments":{"path":"main.go","start_line":1,"end_line":2,"expected_sha256":"%s"}}}\n' "$sha"
  printf '%s\n' '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"reindex","arguments":{"dry_run":true}}}'
} >&3
for _ in $(seq 1 50); do
  complete=true
  for request_id in 2 3 4 5 6; do
    grep -q "\"id\":$request_id" "$work/mcp.jsonl" || complete=false
  done
  "$complete" && break
  sleep 0.1
done
exec 3>&-
mcp_fd_open=0
await_owned_shutdown "$cidx_server_pid" 'cidx server'
cidx_server_pid=
awk 'substr($0, 1, 1) != "{" { exit 1 }' "$work/mcp.jsonl" || fail 'serve stdout contained nonprotocol output'
[ ! -s "$work/mcp.stderr" ] || fail 'successful serve wrote diagnostics to stderr'
for name in status search read_span reindex; do grep -q "\"name\":\"$name\"" "$work/mcp.jsonl" || fail "MCP tool missing: $name"; done
grep -q '"id":6' "$work/mcp.jsonl" || fail 'dry reindex response missing'
if ! python3 - "$work/mcp.jsonl" 2>"$work/mcp-assertion.stderr" <<'PY'
import json,sys
frames=[json.loads(x) for x in open(sys.argv[1]) if x.strip()]
assert all(isinstance(x,dict) and x.get("jsonrpc") == "2.0" for x in frames)
assert [x.get("id") for x in frames].count(None) == 0
assert {x.get("id") for x in frames} == set(range(1,7)) and len(frames) == 6
assert all(sum(x.get("id") == i for x in frames) == 1 for i in range(1,7))
by_id={x["id"]:x for x in frames}
for i in range(1,7):
  assert i in by_id and "error" not in by_id[i], f"missing/error id {i}"
tools=by_id[2]["result"]["tools"]
assert [x["name"] for x in tools] == ["status","search","read_span","reindex"]
for i in range(3,7):
  r=by_id[i]["result"]
  assert not r.get("isError",False), f"application error {i}"
search=by_id[4]["result"]["structuredContent"]
assert search["requested_mode"] == search["effective_mode"] == "fts"
assert search["requested_max_inline_bytes"] == search["effective_max_inline_bytes"] == 0
assert search["query_embedding_used"] is False
results=search["results"]
hello=next((x for x in results if x["path"] == "main.go" and x["symbol"] == "Hello"), None)
assert hello is not None, "FTS result did not contain main.go Hello"
assert hello["score_source"] == "fts" and hello["content_source"] == "indexed_snapshot"
assert hello["body"] is None and hello["body_complete"] is False and hello["body_bytes"] == 0
assert hello["body_omission_reason"] == "NO_FITTING_INDEXED_BODY"
span=by_id[5]["result"]["structuredContent"]
assert span["body"] == 'package sample\nfunc Hello() string { return "hello" }\n'
assert by_id[6]["result"]["structuredContent"]["dry_run"] is True
PY
then
  cat "$work/mcp-assertion.stderr" >&2
  fail 'MCP transcript structural validation failed'
fi
[ ! -e "$serve_repo/.cidx/db/embeddings.db" ] || fail 'serve created product source bank'
[ ! -d "$serve_repo/.cidx/test" ] || fail 'serve created development state'

schema_repo="$work/newer-schema-root"
mkdir "$schema_repo"
offline git -C "$schema_repo" init >/dev/null
(cd "$schema_repo" && offline "$binary" init)
offline sqlite3 "$schema_repo/.cidx/db/index.db" 'PRAGMA user_version=999;'
if offline "$binary" status --root "$schema_repo" >"$work/newer-schema.stdout" 2>"$work/newer-schema.stderr"; then fail 'newer schema unexpectedly opened'; fi
[ ! -s "$work/newer-schema.stdout" ] || fail 'newer-schema failure wrote protocol/data to stdout'
grep -Eiq 'newer than supported|schema version' "$work/newer-schema.stderr" || fail 'newer-schema diagnostic was not actionable'

invalid="$work/invalid-config"
mkdir "$invalid"
offline git -C "$invalid" init >/dev/null
mkdir "$invalid/.cidx"
printf '{"version":1,"unknown_phase14_field":true}\n' >"$invalid/.cidx/config.json"
if offline "$binary" status --root "$invalid" >"$work/invalid-config.stdout" 2>"$work/invalid-config.stderr"; then fail 'unsupported config unexpectedly opened'; fi
[ ! -s "$work/invalid-config.stdout" ] || fail 'invalid-config failure wrote protocol/data to stdout'
grep -Eiq 'unknown|config' "$work/invalid-config.stderr" || fail 'invalid-config diagnostic was not actionable'

project="$work/codex-project"
mkdir -p "$project/.codex" "$work/codex-home"
printf '[mcp_servers.cidx]\ncommand = "%s"\nargs = ["serve", "--root", "%s"]\ncwd = "%s"\nenv_vars = ["VOYAGE_API_KEY"]\n' "$binary" "$serve_repo" "$serve_repo" >"$project/.codex/config.toml"
mkdir -p "$work/home"
printf '[projects."%s"]\ntrust_level = "trusted"\n' "$project" >"$work/codex-home/config.toml"
codex_fifo="$work/codex-app-server.stdin"
mkfifo "$codex_fifo"
HOME="$work/home" CODEX_HOME="$work/codex-home" offline codex app-server --disable plugins --strict-config --listen stdio:// <"$codex_fifo" >"$work/codex-app-server.jsonl" 2>"$work/codex-app-server.stderr" &
codex_server_pid=$!
exec 4>"$codex_fifo"
codex_fd_open=1
printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"clientInfo":{"name":"cidx-local-release","version":"1"},"capabilities":{"experimentalApi":true}}}' >&4
for _ in $(seq 1 50); do
  grep -q '"id":1' "$work/codex-app-server.jsonl" && break
  sleep 0.1
done
grep -q '"id":1' "$work/codex-app-server.jsonl" || fail 'Codex app-server initialize response missing'
printf '{"jsonrpc":"2.0","id":2,"method":"config/read","params":{"cwd":"%s","includeLayers":true}}\n' "$project" >&4
for _ in $(seq 1 50); do
  grep -q '"id":2' "$work/codex-app-server.jsonl" && break
  sleep 0.1
done
exec 4>&-
codex_fd_open=0
await_owned_shutdown "$codex_server_pid" 'Codex app-server'
codex_server_pid=
[ ! -s "$work/codex-app-server.stderr" ] || fail 'Codex app-server wrote diagnostics during project-config verification'
python3 - "$work/codex-app-server.jsonl" "$project" "$work/codex-home/config.toml" "$binary" "$serve_repo" <<'PY'
import json,os,sys
frames=[json.loads(line) for line in open(sys.argv[1]) if line.strip()]
project,home_config,binary,root=sys.argv[2:]
responses={frame.get("id"):frame for frame in frames if "id" in frame}
assert set(responses) == {1,2}
assert all("error" not in response for response in responses.values())
config=responses[2]["result"]
server=config["config"]["mcp_servers"]
assert list(server) == ["cidx"]
expected={"command":binary,"args":["serve","--root",root],"cwd":root,"env_vars":["VOYAGE_API_KEY"]}
assert {key:server["cidx"][key] for key in expected} == expected
assert server["cidx"]["enabled"] is True
layers=config["layers"]
project_layers=[layer for layer in layers if layer.get("name",{}).get("type") == "project"]
assert len(project_layers) == 1
layer=project_layers[0]
assert layer["name"]["dotCodexFolder"] == os.path.join(project,".codex")
assert layer.get("disabledReason") is None
assert layer["config"]["mcp_servers"] == {"cidx":expected}
assert "[mcp_servers" not in open(home_config).read()
PY
HOME="$work/home" CODEX_HOME="$work/codex-home" offline codex --version >"$work/codex-version.txt" 2>"$work/codex-version.stderr"
if [ -n "${CIDX_EVIDENCE_DIR:-}" ]; then
  copy_evidence "$CIDX_EVIDENCE_DIR"
  printf 'local verifier transcripts copied to %s\n' "$CIDX_EVIDENCE_DIR"
fi
printf 'verified local darwin/arm64 archive; no model or assistant invocation was performed\n'
