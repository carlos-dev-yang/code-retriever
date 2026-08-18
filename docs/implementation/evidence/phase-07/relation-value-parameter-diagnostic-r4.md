# Phase 07 Value-Parameter Contract Diagnostic — Revision 4

- Phase: `07-lexical-evaluation`
- State: measured calibration diagnostic; policy decision deferred by the owner
- Date: 2026-08-18
- Implementation commit: `7879ab7315bd215fab34d5756b6416158b6c382d`
- Selection policy: `query-edge-value-parameter-dense-first-v1`
- Provider operations: zero
- Question and label changes: none

## Purpose and boundary

RHF X08 asks for the public `FormState` component's props contract. The exact
`FormState`/`FormStateProps` name collision is unique, but the underlying
structure is not: all six reviewed public RHF React components declare an
explicit `*Props` value-parameter type. The v2 relation graph classified those
uses as generic `TYPE_LOCAL`.

The v3 development sidecar adds the mechanically derived
`TYPE_VALUE_PARAMETER` role. It is distinct from generic `TYPE_PARAMETER` and
does not contain a React, component-name, query-ID, corpus, or `*Props` suffix
exception. Go input parameters in declarations, methods, function types,
function literals, and interface methods use the same role; receivers and
result parameters remain excluded.

Only the new calibration selector consumes the role. It inserts one
`value_parameter_mismatch` field immediately after the existing qualifier
field when normalized query tokens contain `props` or `contract`. The earlier
metadata-dense-first and graph-first algorithms and their immutable v2
artifacts remain historical evidence. Production index/search schemas, FTS,
vectors, RRF, MCP, questions, and labels are unchanged.

## Implementation and validation

The implementation uses sidecar schema 3, protocol
`cidx.relation-diagnostic.v3`, and metadata policy
`occurrence-context-ast-compiler-v2`. Independent Terra code review found one
missing Go owner class; `func_literal` and interface `method_elem` were added
with the same input-parameter containment guard. The focused re-review was
`CLEAR`.

The main commit-boundary validation passed:

```text
env -u VOYAGE_API_KEY GOPROXY=off go test -count=1 -race ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go vet ./internal/relationdiag ./internal/devlab
env -u VOYAGE_API_KEY GOPROXY=off go build -o /tmp/cidx-relationdiag-v3-boundary ./cmd/cidx
node --check tools/relationdiag/typescript-resolver.mjs
gofmt -l internal/relationdiag internal/devlab/relations.go
go mod tidy -diff
git diff --check
```

No broad new test matrix was added. The owner explicitly prioritized real
corpus execution over additional test scaffolding; the two fresh graphs and
all-32 replay below are the measured boundary.

The clean executable records commit `7879ab7315bd215fab34d5756b6416158b6c382d`,
`source_modified=false`, and SHA-256
`db2568547f28ad0e4d22f9cece044706fad347094d0ad5c13d67626d6d99c45b`.

## Fresh graph evidence

| Corpus | Parents | Occurrences | Resolved unique | `TYPE_VALUE_PARAMETER` resolved | Logical graph SHA-256 | Database SHA-256 |
| --- | ---: | ---: | ---: | ---: | --- | --- |
| chi | 452 | 6,737 | 1,740 | 89 | `5dcd27636fbfa2d4e4b40aec076af8899d10f375d42f4cd6866c6e6a3ca14f35` | `7408ef9f5eca62857345ab9bc6c6f1d698d29e92c386b6836216dd8089754b9d` |
| React Hook Form | 322 | 26,011 | 3,267 | 513 | `592b5f63faad488b7b721f127283e2196a353db34ec7adaeca1dbbeb1e0c334f` | `3756cf34cc33d529402f93a1c8b0a3f69524576a139355d0c2bd1f0f29d2bc41` |

The six mechanically verified RHF component contracts are:

| Source | Target | Path | Byte range | Metadata |
| --- | --- | --- | --- | --- |
| `module.Controller` | `module.ControllerProps` | `src/controller.tsx` | `1616..1631` | `SIGNATURE / TYPE_VALUE_PARAMETER / DECLARATION` |
| `module.FieldArray` | `module.FieldArrayProps` | `src/fieldArray.tsx` | `1217..1232` | same |
| `module.Form` | `module.FormProps` | `src/form.tsx` | `1077..1086` | same |
| `module.FormProvider` | `module.FormProviderProps` | `src/useFormContext.tsx` | `2849..2866` | same |
| `module.FormState` | `module.FormStateProps` | `src/formStateSubscribe.tsx` | `688..702` | same |
| `module.Watch` | `module.WatchProps` | `src/watch.tsx` | `1582..1592` | same |

The X08 exact relation remains unique with relation ID
`4baa57b19ec63201ccb8af423b5fb56f4f8f8a154b0cebe4e7dffbfbbd36d43e`.
All declared G09/X08/T09/T10 forward and reverse probes passed.

## All-32 replay result

The replay reused the frozen exhaustive `dense_1024_int8` top 20. It made no
Voyage request, persisted no query vector, preserved every primary dense top
five, and changed no question or relevance label.

| Corpus | Queries | Baseline complete | Augmented complete | Hard-negative attachments | `walkXFF` attachments | Internal safety gate |
| --- | ---: | ---: | ---: | ---: | ---: | --- |
| chi | 12 | 11 | 12 | 0 | 0 | eligible |
| React Hook Form | 20 | 19 | 19 | 0 | 0 | eligible |
| total | 32 | 30 | 31 | 0 | 0 | eligible |

This is identical complete-answer coverage to the accepted v2 metadata
dense-first diagnostic: G09 remains recovered and X08 remains
`RELATION_ADMISSION`. Only two selected relation IDs changed, both because the
query contains `contract`:

- RHF T09 remains complete from its protected dense results, while the related
  selector changes to `module.createFormControl -> module.InternalFieldName`;
- RHF X08 changes to that same relation and remains incomplete.

For X08, the correct `FormState -> FormStateProps` fact is reachable and has
the new role. Its prefix begins `[0,0,-3,-6,-4,...]`. The selected unrelated
`createFormControl -> InternalFieldName` value-parameter fact begins
`[0,0,-4,-2,-4,...]`. Therefore the generic role removes one structural
ambiguity but leaves 513 resolved RHF value-parameter type facts; the existing
context-overlap ordering still cannot identify the component's contract.

The built-in safety gate is green because no declared hard negative is added.
The predeclared usefulness condition that X08 become complete is not met. Per
the owner's instruction, this document does not accept or reject the policy;
the policy decision, any further selector design, and any question replacement
are deferred. No additional key adjustment was made after seeing the result.

## Immutable local artifacts

```text
.cidx/test/states/chi-1024-int8/evaluations/relation-graph-chi-value-param-v3-7879ab7
.cidx/test/states/react-hook-form-1024-int8/evaluations/relation-graph-rhf-value-param-v3-7879ab7
.cidx/test/states/chi-1024-int8/evaluations/relation-diagnostic-chi-value-param-v1-7879ab7
.cidx/test/states/react-hook-form-1024-int8/evaluations/relation-diagnostic-rhf-value-param-v1-7879ab7
```

Canonical entry-list checksums:

| Artifact | SHA-256 |
| --- | --- |
| chi graph | `3d92ee683005c21237f6f6b046be68110ce599221794ae24fd4b71b5b31c3836` |
| RHF graph | `d58ba5d2b6e7f25ad4d83247b94fb30c7f0f5d390d693191662b5a06ad17f0f2` |
| chi replay | `923f6d9231a78ea4e0550d4bb201e0ba14b667fb61da16d6def149e678712a10` |
| RHF replay | `d04d377bcfaa54c4188b2298f217d93b0e534e1d9f480ad9fb4fc205fea9b1cc` |

Every recorded entry digest and byte count was recomputed after execution.
The artifact entry sets contain no WAL/SHM file, credential, vector, query
vector, absolute checkout path, or provider operation. Independent Terra
artifact review was intentionally stopped when the owner deferred the policy
decision; the checksums and measured facts above are main-agent evidence, not
an independent artifact-acceptance claim.

## Handoff

Keep Phase 07 `in_progress`. The 32 exposed cases remain calibration only and
the separate unexposed confirmation set remains required. Resume this policy
decision only on explicit owner direction; do not tune another ordering on X08
or silently treat the green internal safety gate as usefulness acceptance.
