# Assistant Session Policy Trace V2

- Status: implementation support for the owner-authorized forced-cidx prompt
  diagnostic; not yet frozen for execution
- Scope: ignored host-side evaluation artifact only
- Implementation: `scripts/assistant_session_policy_trace.py`
- Protocol: `forced-cidx-policy-v2`

## Boundary

This trace is separate from `scripts/assistant_session_trace.py`. The latter is
the frozen `passive-v1` replay authority for the prior availability run and is
not changed by this work. The new trace does not change MCP calls, SQLite,
search ranking, source files, prompt contents, grading, or product behavior.

The trace retains no source body, shell output, command text, assistant answer,
truth, or grade. It stores only command hashes, action classes and ordinals,
source-relative locations, source hashes, byte counts, locator metadata, and
final citation locations.

## Command classes

`ordinary_repository_discovery` includes the policy's named discovery commands
`rg`, `grep`, `git grep`, `find`, and `fd`, plus recognized equivalents (`ag`,
`ack`, `pt`, `fdfind`, `locate`, `ls`, `tree`, `git ls-files`, and
`git ls-tree`). The trace records the command family but never its arguments.

Pipeline classification is purpose-aware at the mechanically observable
boundary. Filtering build or test output, such as `go test ./... | grep FAIL`,
is non-search verification. Filtering a named source file, such as
`cat file.go | grep term`, remains a known-file read. A text-search stage whose
upstream producer cannot be proven to be either of those is
`ambiguous_repository_discovery`; it fails directed-policy compliance rather
than being allowed to create false compliance.

Invoking `cidx` itself through a shell executable (including a path-qualified
binary or `command -v`/`which`/`type` probe) is a separate
`shell_cidx_attempt`. Merely printing the word `cidx` is not. This action is
retained as policy noncompliance even if proper MCP search/read calls also
occur, preventing a mixed path from being reported as fully compliant.

`known_file_read` covers `sed`, `cat`, `head`, `tail`, `awk`, `nl`, `less`, and
`more` when no ordinary discovery family occurs in the same command. Every
other command execution is represented as `non_search_repository_action`; this
makes compilation, tests, formatting, and environment checks observable without
treating them as candidate discovery.

For each command classified as discovery, ambiguous discovery, or known-file
reading, the module reuses the v1 verifier to retain only source-verified output
ranges and bytes. It retains safe plain source paths only when every sequential
command segment has a recognized path-list output contract. Source text that
happens to look like a path is therefore not attributed as another discovered
file. Ranges are not trusted merely because a command printed text.

## Mechanically derived policy fields

The `policy_stage.directed_policy` object is arm-neutral. The reducer applies
it to the directed arm rather than allowing the trace to decide experiment
membership. A directed turn would mechanically comply only when it has a
`cidx.search`, no ordinary repository-discovery command, its first discovery is
`cidx.search`, and at least one cidx locator is selected through an exact
`read_span`.

Noncompliance stays an observed outcome: it is never a retry, execution
failure, exclusion, or grade decision. Completed observations are joined to
started actions by event identity rather than list position. Incomplete search
and read attempts stay visible as attempts, receive no search/read credit, and
make a directed turn noncompliant.

`selected_locator_read_span_count` is a count of successful read attempts, not
repeated locator rows: each credited read must exactly match a locator (`path`,
start line, end line, and indexed SHA-256) returned by a **prior** cidx search in
the same trace. Locator `read_status` uses the same chronological rule. The
trace also records ordinary and ambiguous discovery before/after the first
cidx search, cidx and ordinary unique source bytes, non-temporal overlap,
temporally reacquired bytes, exact cidx reads, and overlapping cidx reads.
This supports the separate policy, evidence, and exploration surfaces
defined in `ASSISTANT-FORCED-CIDX-PROMPT-EVALUATION-V1.md` without a weighted
score.

The builder performs a recursive fail-closed key audit before returning. The
stored trace cannot contain exact fields for source body, command text, shell
output, query text, assistant answer, truth, or grade; only hashes, identities,
locations, counts, and classifications cross this boundary.

## Integration contract

The runner/reducer integration must select this module only for the new
`forced-cidx-policy-v2` manifest protocol, store the wrapper and delegated
`assistant_session_trace.py` hashes plus the protocol/schema version in the new
run manifest, and rebuild the stored trace before grading. It must continue to
select and hash the unmodified v1 builder for the earlier frozen run.

## Focused checks actually run

- Python compilation passed for the trace, runner, and scorer modules.
- `ls -R` classified as ordinary discovery.
- `go test ./... | grep FAIL` classified as non-search verification.
- `cat a.go | grep term` classified as a known-file read.
- an unknown producer piped to `grep` classified as ambiguous discovery.
- `find . | grep term` classified as ordinary discovery with a path-list
  contract.
- direct, path-qualified, and executable-probe shell cidx calls classified as
  `shell_cidx_attempt`, while `echo cidx` did not; a trace containing both a
  proper selected cidx read and a shell cidx call remained noncompliant.
- an incomplete read followed by a completed read retained both attempts and
  mapped the completed observation to the correct event ordinal.
- a read occurring before its matching search was not credited and its locator
  remained `unread`.
- source text equal to another existing filename did not create a false
  discovered path.
- ordinary source read before cidx counted as overlap but not reacquisition;
  the same read after cidx counted as temporal reacquisition.
- an existing V6 cidx search plus exact read replayed as one compliant search,
  one selected read, and a body-free trace.
