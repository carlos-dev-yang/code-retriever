# Phase 07 chi/RHF Behavior-Cohort Working Set — Revision 4

- Status: `draft_authoring`
- Date: 2026-08-16
- Authority: machine-prepared working set for review; not frozen truth and not
  promotion evidence
- Scope: 12 Go, 12 TypeScript, and 8 TSX behavior-oriented calibration
  candidates
- Provider calls: `0`

## Purpose and counting rule

The existing 12 exact-identifier cases remain pipeline reference only. This
worksheet starts the first behavior-oriented set from the real generation-2
parent inventories. Every row has exactly one proposed `task:*` tag and one
proposed `signal:*` tag. `Direct` and `Support` identify provisional parent
judgments, not rank expectations. Lines are review coordinates; the final
dataset will bind content hash, qualified symbol, and byte range from the
post-fix inventory.

The set is intentionally small enough for rapid exploratory retrieval. It is
not the later 30-per-language confirmation floor. Query wording, answer mode,
and judgments may change after the first approved dense run, but difficult
cases may not be deleted merely because they score poorly.

## Go working slice

| ID | Natural-language query | Proposed task / signal | Mode | Direct parent(s) | Useful support |
| --- | --- | --- | --- | --- | --- |
| G01 | How does a request entering a top-level router reset and reuse its route context before the handler chain runs? | `task:single_parent_behavior` / `signal:semantic_only` | `SINGLE` | `mux.go:60` `chi.Mux.ServeHTTP` | none required |
| G02 | How does routing choose between a matching handler, method-not-allowed, and not-found responses? | `task:lifecycle_state_error_or_configuration` / `signal:semantic_only` | `SINGLE` | `mux.go:445` `chi.Mux.routeHTTP` | `tree.go:382` `chi.node.FindRoute` |
| G03 | After a successful trie lookup, how are captured URL parameters and the matched pattern committed to `RouteContext`? | `task:single_parent_behavior` / `signal:semantic_only` | `SINGLE` | `tree.go:382` `chi.node.FindRoute` | `tree.go:407` `chi.node.findRoute` |
| G04 | How does mounting a subrouter inherit fallback handlers and shift the remaining path before delegating? | `task:single_parent_behavior` / `signal:semantic_only` | `SINGLE` | `mux.go:288` `chi.Mux.Mount` | none required |
| G05 | How are middleware functions wrapped so registration order is preserved before the endpoint runs? | `task:single_parent_behavior` / `signal:semantic_only` | `SINGLE` | `chain.go:34` `chi.chain` | `chain.go:10` `chi.Middlewares.Handler` |
| G06 | How are Basic credentials compared without a timing-sensitive password check, and how is a failed challenge returned? | `task:delegated_or_cross_parent_flow` / `signal:mixed` | `BEST_N` | `middleware/basic_auth.go:9` `middleware.BasicAuth` | `middleware/basic_auth.go:30` `middleware.basicAuthFailed` |
| G07 | How does response compression choose an accepted encoder and avoid compressing disallowed content types? | `task:delegated_or_cross_parent_flow` / `signal:semantic_only` | `BEST_N` | `middleware/compress.go:212` `middleware.Compressor.selectEncoder`; `middleware/compress.go:276` `middleware.compressResponseWriter.isCompressible` | `middleware/compress.go:187` `middleware.Compressor.Handler`; `middleware/compress.go:31` `middleware.Compress` |
| G08 | How are ordinary panics logged and converted to 500 while abort-handler panics are rethrown and upgraded connections avoid the status write? | `task:lifecycle_state_error_or_configuration` / `signal:semantic_only` | `SINGLE` | `middleware/recoverer.go:17` `middleware.Recoverer` | none required |
| G09 | How does the deprecated client-IP middleware choose between forwarding headers before mutating `RemoteAddr`? | `task:delegated_or_cross_parent_flow` / `signal:mixed` | `BEST_N` | `middleware/realip.go:16` `middleware.RealIP` | `middleware/realip.go:39` `middleware.realIP` |
| G10 | How are concurrent requests limited, queued with a timeout, cancelled, and given an optional Retry-After value? | `task:delegated_or_cross_parent_flow` / `signal:semantic_only` | `BEST_N` | `middleware/throttle.go:43` `middleware.ThrottleWithOpts` | `middleware/throttle.go:145` `middleware.throttler.setRetryAfterHeaderIfNeeded` |
| G11 | How does the timeout middleware cancel downstream work and decide whether to write a gateway-timeout response? | `task:lifecycle_state_error_or_configuration` / `signal:mixed` | `SINGLE` | `middleware/timeout.go:9` `middleware.Timeout` | none required |
| G12 | How is the first matching header middleware selected with wildcard support and a default fallback? | `task:delegated_or_cross_parent_flow` / `signal:semantic_only` | `BEST_N` | `middleware/route_headers.go:77` `middleware.HeaderRouter.Handler` | `middleware/route_headers.go:141` `middleware.Pattern.Match` |

Reserve candidates are URL suffix routing through `middleware.URLFormat` and
router path-conflict detection through `chi.Mux.Mount`. They are not added to
the denominator until a new working-set version explicitly promotes them.

## TypeScript working slice

| ID | Natural-language query | Proposed task / signal | Mode | Direct parent(s) | Useful support |
| --- | --- | --- | --- | --- | --- |
| T01 | How is one form control created or replaced, subscribed to root form-state changes, and exposed through a proxy that tracks accessed state? | `task:delegated_or_cross_parent_flow` / `signal:semantic_only` | `BEST_N` | `src/useForm.ts:18` `module.useForm` | `src/logic/createFormControl.ts:142` `module.createFormControl`; pending `getProxyFormState` parent |
| T02 | How does a controlled field subscribe to value and form state, translate change/blur events, and unregister or mark itself unmounted? | `task:lifecycle_state_error_or_configuration` / `signal:semantic_only` | `BEST_N` | `src/useController.ts:26` `module.useController` | `module.useWatch` after overload repair; `src/useFormState.ts:14` `module.useFormState` |
| T03 | How does the field-array hook keep generated IDs aligned while array operations update control state and subscribers? | `task:lifecycle_state_error_or_configuration` / `signal:mixed` | `SINGLE` | `src/useFieldArray.ts:47` `module.useFieldArray` | none required until blind pooling finds a tighter array helper |
| T04 | How are nested dirty fields recursively marked or cleared while registered array leaves remain atomic? | `task:single_parent_behavior` / `signal:semantic_only` | `SINGLE` | `src/logic/getDirtyFields.ts:58` `module.getDirtyFields` | helpers contained in the same source area |
| T05 | How does deep equality handle cycles, dates, empty custom-prototype objects, arrays, and the special `ref` key? | `task:single_parent_behavior` / `signal:semantic_only` | `SINGLE` | `src/utils/deepEqual.ts:9` `module.deepEqual` | none required |
| T06 | Which values are recursively cloned, and which browser or custom objects deliberately retain their original identity? | `task:single_parent_behavior` / `signal:semantic_only` | `SINGLE` | `src/utils/cloneObject.ts:5` `module.cloneObject` | none required |
| T07 | How is the current value extracted differently for file inputs, radios, multiple selects, checkboxes, and ordinary fields? | `task:single_parent_behavior` / `signal:mixed` | `SINGLE` | `src/logic/getFieldValue.ts:11` `module.getFieldValue` | input-specific value helpers are grade-1 pool candidates |
| T08 | Where are registration, validation, reset, subject notification, and form-state transitions coordinated for the whole form? | `task:lifecycle_state_error_or_configuration` / `signal:semantic_only` | `SINGLE` | `src/logic/createFormControl.ts:142` `module.createFormControl` | none required; large-parent diagnostic |
| T09 | What API contract exposes both public form operations and the internal state, field, subject, validation, and array hooks used by the implementation? | `task:interface_type_or_api_contract` / `signal:mixed` | `SINGLE` | `src/types/form.ts:870` `module.Control` | none required |
| T10 | How does the type system construct dotted field paths recursively while avoiding infinite traversal through already-seen object types? | `task:interface_type_or_api_contract` / `signal:semantic_only` | `BEST_N` | `src/types/path/eager.ts:58` `module.Path`; `src/types/path/eager.ts:38` `module.PathInternal` | related array-path types are grade-1 pool candidates |
| T11 | How are required, min/max, length, pattern, native, and custom validation rules combined for one field, including all-criteria mode? | `task:delegated_or_cross_parent_flow` / `signal:semantic_only` | `BEST_N` | pending path-derived `validateField` parent | `src/logic/getValidateError.ts:5` `module.getValidateError` |
| T12 | How are observers added, notified, individually unsubscribed, and cleared from the form-state subject? | `task:lifecycle_state_error_or_configuration` / `signal:semantic_only` | `SINGLE` | pending path-derived `createSubject` parent | `src/utils/createSubject.ts:11` `module.Subject` type |

`schemaErrorLookup` is retained as a reserve behavior candidate. T01, T11, and
T12 must not enter an executable dataset until the path-derived default-export
policy is decided and their final parent identities exist. T02 additionally
waits for the real-corpus overload repair so one `useWatch` API is not labeled
against eight duplicate parents.

## TSX working slice

| ID | Natural-language query | Proposed task / signal | Mode | Direct parent(s) | Useful support |
| --- | --- | --- | --- | --- | --- |
| X01 | How does the Controller render-prop component obtain field, field-state, and form-state values for a controlled input? | `task:delegated_or_cross_parent_flow` / `signal:semantic_only` | `BEST_N` | `src/controller.tsx:4` `module.Controller` | `src/useController.ts:26` `module.useController` |
| X02 | How are all form methods memoized into React context and later retrieved by deeply nested inputs? | `task:delegated_or_cross_parent_flow` / `signal:semantic_only` | `BEST_N` | `src/useFormContext.tsx:55` `module.FormProvider`; `src/useFormContext.tsx:14` `module.useFormContext` | `FormProviderProps` is a grade-1 pool candidate |
| X03 | How does the Form component convert submitted values, call a URL or function action, and turn failures into root server errors? | `task:lifecycle_state_error_or_configuration` / `signal:semantic_only` | `BEST_N` | `src/form.tsx:17` `module.Form` | `src/utils/formData.ts:3` `module.jsonToFormData` |
| X04 | How does the Watch render-prop component subscribe to selected values or a computed result and render the returned value? | `task:delegated_or_cross_parent_flow` / `signal:semantic_only` | `BEST_N` | `src/watch.tsx:9` `module.Watch` | repaired single `module.useWatch` parent |
| X05 | How does the FieldArray component obtain field-array operations and pass them to its render callback? | `task:delegated_or_cross_parent_flow` / `signal:mixed` | `BEST_N` | `src/fieldArray.tsx:15` function `module.FieldArray` | `src/useFieldArray.ts:47` `module.useFieldArray` |
| X06 | How does the FormState render-prop component subscribe only to the requested form-state slice? | `task:delegated_or_cross_parent_flow` / `signal:semantic_only` | `BEST_N` | `src/formStateSubscribe.tsx:21` function `module.FormState` | `src/useFormState.ts:14` `module.useFormState` |
| X07 | How does Form choose between a headless render callback and a native form element, especially when the action is a function? | `task:single_parent_behavior` / `signal:semantic_only` | `SINGLE` | `src/form.tsx:17` `module.Form` | none required |
| X08 | What props contract controls the name, exactness, disabled state, control, and render callback of the FormState component? | `task:interface_type_or_api_contract` / `signal:mixed` | `SINGLE` | `src/formStateSubscribe.tsx:11` `module.FormStateProps` | function `module.FormState` is a grade-1 pool candidate |

The type and function named `module.FieldArray`, and the type and function named
`module.FormState`, are intentional same-name/different-kind cases. Their
non-direct counterpart should be retained as an ambiguity candidate during
blind pooling rather than automatically judged irrelevant.

## Structural gates before JSON publication

1. Implement the accepted deterministic path-derived retrieval-label exception
   for anonymous default-export function-like declarations in the existing
   `symbol` and `qualified_symbol` fields, with no alias/schema expansion.
2. Repair overload grouping across export modifiers and associated JSDoc so
   `useWatch`, `insert`, and `mockZodResolver` each produce one function parent
   per logical overload set.
3. Increment the TypeScript chunker and index profile versions and run one full
   provider-free reindex of both corpora.
4. Rebind every row to the new inventory's exact content hash, qualified
   symbol, byte range, and parent kind.
5. Side-panel review the actual questions and provisional direct/support
   assignments, then publish a versioned draft `EvaluationDataset` with
   deterministic digests.

No current row is permission for document or query embedding.

## Generation-3 binding result

All five structural gates above are complete. The current draft calibration
datasets are:

```text
testdata/retrieval/behavior-go-chi-v5.3.1-draft-v1.json
  12 Go cases
  sha256 a51a0642491611e5c6517a360bf38fc8ef2e40f4c9df0cd43030a9562ae9e6a6

testdata/retrieval/behavior-react-hook-form-v7.85.0-draft-v1.json
  12 TypeScript + 8 TSX cases
  sha256 b8a2facc95eab83fc348752c54d0075cb4a0f2b38ca0b5ae1f90815f0b702fb2
```

Every case is bound to one corrected generation-3 inventory through exact
repository-relative path, indexed content SHA-256, qualified symbol, and byte
range. All 32 case digests match the documented RFC 8785 framing, and every
direct/support judgment resolves to exactly one production parent. The files
remain `draft`; the advisory question review is not either formal human label
pass, and no hard-negative or confirmation denominator is claimed yet.

## Side-panel advisory review and accepted corrections

The 32 rows were submitted, without hashes, source bodies, local paths, or
credentials, to the existing ChatGPT and Grok side-panel conversations. Both
reviewers treated unresolved parser labels as structural gates rather than
query failures. The source-checked resolution was:

- narrow G01 to the context-pool lifecycle owned entirely by
  `Mux.ServeHTTP`;
- narrow G03 to the `RouteContext` commit owned by `node.FindRoute`, keeping
  the internal trie walk as support;
- make G05 a single-parent composition case owned by `chain`;
- keep G07 as a required multi-parent sequential flow (`BEST_N`), not a claim
  to enumerate every compression implementation;
- remove the overly broad `createFormControl` support judgment from T03 and
  mark its signal mixed; and
- retain ambiguity candidates such as `ServeHTTP` versus `routeHTTP`, `Route`
  versus `Mount`, `Compress` versus `Compressor.Handler`, `Controller` versus
  `useController`, and `Path` versus runtime path helpers during later blind
  pooling.

One adviser suggested adding a mixed-language query immediately. That was not
accepted because the user explicitly deferred mixed-language corpus work until
the chi/RHF slices close, and the two independent repositories do not contain
a genuine cross-language behavior path. The review is advisory preparation,
not either required human label pass.
