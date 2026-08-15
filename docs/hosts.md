# Codex project MCP setup

Only Codex project-scoped stdio configuration is documented for v1. The
current configuration contract uses a trusted repository's
`.codex/config.toml` with `[mcp_servers.<name>]`, `command`, and `args`.
`cwd` and `env_vars` are optional supported fields. See the official
[Codex MCP documentation](https://developers.openai.com/codex/mcp/).

The local verifier targets `codex-cli 0.148.0-alpha.9`. Its isolated
`CODEX_HOME/config.toml` trusts only the temporary project with
`[projects."<absolute-project-path>"]` and `trust_level = "trusted"`; it then
uses the model-free `codex app-server --strict-config --listen stdio://`
`config/read` protocol with that project's `cwd`. It verifies the effective
project layer and the exact `cidx` command, args, `cwd`, and `env_vars`, without
copying the MCP entry into `CODEX_HOME`. No user project or home configuration
is read or changed.

Use an absolute repository root. This example keeps the binary on `PATH`:

```toml
[mcp_servers.cidx]
command = "cidx"
args = ["serve", "--root", "/absolute/path/to/repository"]
cwd = "/absolute/path/to/repository"
```

When it is not on `PATH`, use an absolute command path instead. TOML strings
preserve paths containing spaces; quote the entire path as one string.

```toml
[mcp_servers.cidx]
command = "/absolute/path with spaces/cidx"
args = ["serve", "--root", "/absolute/repository with spaces"]
cwd = "/absolute/repository with spaces"
```

For optional explicitly approved hybrid use, forward only the environment
variable name. Never write its value into the project file:

```toml
env_vars = ["VOYAGE_API_KEY"]
```

cidx never writes, merges, registers, or removes host configuration. Restart
the host after changing the project configuration or cidx binary. The MCP
server writes protocol frames only to stdout; diagnostics belong on stderr.
It exposes exactly `status`, `search`, `read_span`, and `reindex`. A caller
must always provide `search.max_inline_bytes`; the server clamps it to the
project hard maximum (64 KiB by default, 1 MiB executable ceiling). `read_span`
has no line-count cap and returns `SPAN_TOO_LARGE` rather than a partial body.

The local verifier is designed to parse this project-scoped configuration
through the local Codex CLI in an isolated `HOME`/`CODEX_HOME`, with network
blocked. It also directly exercises the four-tool stdio handshake. That is
configuration and transport evidence only: no Codex model or assistant task is
run, and no assistant-use or release-candidate claim follows from it.
