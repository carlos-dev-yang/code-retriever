# Phase 00 Canonical Profile and Hash Fixtures

- State: `complete`
- Canonical JSON: RFC 8785 JSON Canonicalization Scheme (JCS)
- Digest framing: UTF-8 domain bytes, one NUL byte, then canonical payload bytes

The domain-separated identities are already fixed conceptually:

```text
canonical_input_sha256 =
  SHA256("cidx/input/v1" || NUL || canonical_input_utf8)

canonical_text_profile_fingerprint =
  SHA256("cidx/canonical-text-profile/v1" || NUL || canonical_json(resolved profile))

embedding_source_profile_fingerprint =
  SHA256("cidx/embedding-source-profile/v1" || NUL || canonical_json(resolved source profile))

vector_space_profile_fingerprint =
  SHA256("cidx/vector-space-profile/v1" || NUL || canonical_json(resolved vector-space profile))

vector_storage_profile_fingerprint =
  SHA256("cidx/vector-storage-profile/v1" || NUL || canonical_json(resolved storage profile))
```

RFC 8785 provides the cross-language rule. Config/schema validation runs before canonicalization and rejects duplicate/unknown fields, NaN/Inf, invalid Unicode, and unsupported numeric shapes. Resolved profile schemas use integers and booleans for semantic numeric values wherever possible. Implementations must not hash Go map serialization, pretty JSON, source JSON key order, or environment-specific paths.

## Fixture A: canonical input

The payload is 153 UTF-8 bytes and ends with one LF. The body contains one tab before `return`.

```text
path: internal/math/add.go
kind: function
qualified_symbol: example.Add
signature: func Add(a, b int) int
body:
func Add(a, b int) int {
	return a + b
}
```

```text
domain: cidx/input/v1
sha256: 0a9e9610ed55c43bf8470f190c988e00d959fe889260ba2e6c8dc668adf784fc
```

## Fixture B: canonical-text profile

Exact RFC 8785 bytes:

```json
{"formatter_id":"cidx-canonical-text","formatter_version":1,"projection_order":["path","kind","qualified_symbol","signature","body"]}
```

```text
domain: cidx/canonical-text-profile/v1
sha256: eabf6198a0be430d4d5c15f1f036d99e8fbde12381a060be0debb34c211c7016
```

## Fixture C: embedding-source profile

Exact RFC 8785 bytes:

```json
{"adapter_version":1,"input_type_mapping":{"document":"document","query":"query"},"model":"voyage-code-4","output_dtype":"float","provider":"voyage-official","source_dimensions":1024,"truncation":false}
```

```text
domain: cidx/embedding-source-profile/v1
sha256: 923a0b84bf40d3880b5a081861ed6e17208380a5fa152847d66d2b98f222c0b3
```

## Fixture D: vector-space profile

`target_dimensions=512` is a fixture choice, not a project default.

```json
{"metric":"cosine","normalizer_id":"l2-v1","reducer_id":"prefix-v1","source_profile_fingerprint":"923a0b84bf40d3880b5a081861ed6e17208380a5fa152847d66d2b98f222c0b3","target_dimensions":512}
```

```text
domain: cidx/vector-space-profile/v1
sha256: 7499afe97dad0c21f6bd2aef10a19e2a47036de4490de49b80a4426ce22bdd14
```

## Fixture E: vector-storage profile

`fixture-binary-v1` tests framing only. Phase 01 replaces it with the exact production codec/scorer ID after that contract is evidenced.

```json
{"storage_codec_id":"fixture-binary-v1","vector_space_profile_fingerprint":"7499afe97dad0c21f6bd2aef10a19e2a47036de4490de49b80a4426ce22bdd14"}
```

```text
domain: cidx/vector-storage-profile/v1
sha256: 80f910329088b323e1834e5cc44431c78562f71d8f231405712b5f3c164db007
```

## Required conformance cases for Phase 02

- Reordered object keys and insignificant source whitespace produce the same RFC 8785 bytes and digest.
- Explicit and defaulted configs that resolve to identical semantic objects produce the same digest.
- Array order remains significant.
- Unicode strings follow RFC 8785 escaping and ordering, without Unicode normalization not specified by the standard.
- Duplicate keys, unknown fields, lone surrogates/invalid UTF-8, NaN/Inf, and out-of-schema numbers fail before hashing.
- Changing one semantic value or the domain changes the digest.

## Independent recomputation record

On 2026-08-15, one recursive sorted-object JSON/Digest implementation and a separate `printf` plus system `shasum -a 256` pipeline produced the same five digests above. No repository file, secret, provider call, or environment path entered the payloads.
