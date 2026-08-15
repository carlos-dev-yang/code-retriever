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

`serving_dimensions=512` is a fixture choice, not a project default.

```json
{"metric":"cosine","normalizer_id":"l2-v1","reducer_id":"prefix-v1","serving_dimensions":512,"source_profile_fingerprint":"923a0b84bf40d3880b5a081861ed6e17208380a5fa152847d66d2b98f222c0b3"}
```

```text
domain: cidx/vector-space-profile/v1
sha256: 01fd78bd204c5e01b010fba65047e6d6fa565bfcdf6c0d48db9019e709ee91e7
```

## Fixture E: vector-storage profile

`fixture-binary-v1` tests framing only. Phase 01 replaces it with the exact production codec/scorer ID after that contract is evidenced.

```json
{"storage_codec_id":"fixture-binary-v1","vector_space_profile_fingerprint":"01fd78bd204c5e01b010fba65047e6d6fa565bfcdf6c0d48db9019e709ee91e7"}
```

```text
domain: cidx/vector-storage-profile/v1
sha256: e2a6f905948719066e69817c9a82590f32ab1d1105a987233d2cf092a3abdbfa
```

## Required conformance cases for Phase 02

- Reordered object keys and insignificant source whitespace produce the same RFC 8785 bytes and digest.
- Explicit and defaulted configs that resolve to identical semantic objects produce the same digest.
- Array order remains significant.
- Unicode strings follow RFC 8785 escaping and ordering, without Unicode normalization not specified by the standard.
- Duplicate keys, unknown fields, lone surrogates/invalid UTF-8, NaN/Inf, and out-of-schema numbers fail before hashing.
- Changing one semantic value or the domain changes the digest.

## Independent recomputation record

On 2026-08-15, the original fixtures were independently reproduced. During Revision 4 reconciliation, the renamed vector-space payload and dependent storage payload were recomputed independently with `shasum -a 256` and OpenSSL SHA-256; both implementations produced `01fd78...91e7` and `e2a6f9...dbfa`. No repository file, secret, provider call, or environment path entered the payloads.
