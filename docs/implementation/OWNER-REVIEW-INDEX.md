# Owner Review Index — Packaging, FTS, Assistant Evaluation, and Remaining Work

- Date: 2026-08-21
- Language: Korean (owner review). File names and identifiers stay English.
- Status: evaluation-only freeze; not `core_retrieval`, not `release_candidate`
- Authoritative ledger: [`STATUS.md`](STATUS.md)
- This file is the owner entry point for the packaging work, natural-language
  FTS correction, paired assistant diagnostic, and remaining work.

Do not treat this index as promotion evidence. If it disagrees with
`STATUS.md` or a named evidence file, those files win.

---

## 1. 한 줄

검색 순위와 MCP는 그대로 두고, **이미 찾은 파일의 형제 심볼을 평가용으로
4개/4096바이트까지 붙이는 계약**만 채택했다. 그래프 한 홉을 기본 결과에
넣는 것은 기각했다. 기존 chi/RHF critical/general v2 결과는 고정했다.
자연어 질문을 모두 `AND`로 묶어 후보를 0개로 만들던 Phase 06 lexical
planner 교정과 동일 v2 재실행도 끝났다. 후보 0건은 `32/44 -> 0/44`, 완전
정답@5는 `10/44 -> 30/44`가 됐다. 동일 chi/RHF paired assistant A/B V3도
완료했다. 정확도는 baseline `11 complete + 1 partial`에서 cidx `12 complete`로
유지·개선됐지만 모델 토큰은 `37.2%` 늘었다. 다음 순서는 새 저장소나 동일
A/B 반복이 아니라 MCP 검색 응답량을 계측하고 줄이는 것이다.

---

## 2. 지금 상태

| 항목 | 값 |
| --- | --- |
| 제품 | 로컬 코드 검색 MCP. 도구 4개: `status`, `search`, `read_span`, `reindex` |
| 서빙 | 기본 `1024/int8`, 옵션 `512/int8`. 소스 뱅크는 1024-f32 |
| Phase 00–05, 08–11, 13 | `done` |
| Phase 06 | `done` — natural-language planner와 독립 lexical lane 교정 완료 |
| Phase 07 | `done` — 동일 v2 새 run과 단계별 진입/순위 진단 완료 |
| Phase 12 `core_retrieval` | `blocked` |
| Phase 14 `release_candidate` | `blocked` (로컬 darwin/arm64 패키지는 있음) |
| Assistant A/B V3 | 완료 — cidx 12/12 complete, 모델 토큰 +37.2%, unchanged rerun 기각 |
| 정확한 다음 작업 | MCP search 응답의 text/structured/metadata/body 비용 분리 계측과 compact contract 제안 |
| 라이브 패키징 판정 | `CONTINUE_SIBLING_PACKAGING` |
| Voyage | 이번 작업에서 0회 |

### 2.1 최신 critical/general 질문 세트 결과

기존 chi/RHF 질문과 exact-identifier 질문을 새 v2로 합쳤고, 이전 파일과
run은 그대로 보관했다. 각 새 run은 질문 세트 버전·canonical digest·cohort
taxonomy 버전/digest를 직접 기록한다.

- FTS: lexical anchor `10/12`, semantic-only `0/24`, mixed `0/8`
- simple control: 전체 `28/44`, semantic-only `13/24`, mixed `5/8`
- 해석: semantic-only 24개와 mixed 8개는 전부 FTS 후보가 0개였다. 따라서
  이 결과는 BM25 순위가 낮다는 뜻이 아니라, 모든 정규화 토큰을 `AND`로
  묶은 후보 진입 방식이 자연어를 배제했다는 뜻이다.
- 교정 후 최종: 전체 `30/44`, lexical anchor `12/12`, semantic-only
  `15/24`, mixed `3/8`. 후보 0건은 `0/44`다.
- candidate depth 20에서 필요한 그룹을 모두 가진 질문은 `38/44`다. 그중
  8개는 최종 top 5에서 밀렸고, 6개는 후보 단계부터 불완전하다.
- 기존 FTS가 맞힌 10개는 하나도 잃지 않았고 20개를 추가 회복했다.
- 후속 assistant A/B V3는 완료했다. 결과와 다음 교정은
  [FTS/assistant 작업 일지](FTS-REMEDIATION-AND-ASSISTANT-AB-JOURNAL.md)와
  [V3 결과](evidence/phase-14/assistant-ab-v3-result.md)에 기록했다.

정확한 실행 ID와 artifact digest는
[critical/general v2 report](evidence/phase-07/critical-general-question-set-v2.md)에
있다.

교정 과정, 중간 run, 최종 digest와 잔여 한계는
[planner v2 rerun report](evidence/phase-07/natural-language-lexical-rerun-v2.md)에
있다.

개선 계약과 ChatGPT/Grok의 동일-맥락 검토 결과는
[natural-language FTS query-planner review](evidence/phase-07/natural-language-fts-query-planner-review-r4.md)에
있다. 핵심은 `symbol`, `path`, `descriptive FTS`, `dense`를 독립 후보 lane으로
두고, dense를 FTS 후보로 제한하지 않는 것이다. 명시된 동일-result 필수
조건만 `AND`로 유지하며, 자연어 설명은 OR 기반 후보 진입 후 별도 순위를
측정한다.

### 2.2 Assistant A/B V3 결과

- 기존 저장소와 선정된 12질문만 사용했다. 새 저장소와 Voyage 호출은 없다.
- treatment 12/12가 정확한 cidx MCP를 사용했고 총 48회 호출했다.
- baseline은 `11 complete + 1 partial`, cidx FTS는 `12 complete`이며 역전 실패는 없다.
- 전체 모델 토큰은 `1,223,579 -> 1,678,341`로 `37.2%` 증가했다.
- uncached input은 `241,613 -> 363,843`로 `50.6%` 증가했다.
- 양쪽 모두 complete인 11쌍의 model-total ratio 중앙값은 `1.378`, 감소/동률은 `3/11`이다.
- 모든 treatment에서 모델에 노출된 저장소 출력 바이트가 늘었다. 큰 inline 검색
  결과와 반복 `search/read_span`이 우선 조사 대상이다.
- ChatGPT와 Grok 모두 unchanged rerun을 기각했다. 응답량 교정이 1순위,
  도구 가이드가 2순위, lexical/semantic routing은 그 이후다.

---

## 3. 이 작업이 무엇을 한 건지

cidx는 답을 쓰지 않는다. 질문에 대해 함수/타입 덩어리(parent)를 순위대로
준다. 기본은 dense top 5다.

닫힌 40쿼리(go-git, Zustand, Memos)에서 검색이 빠지는 이유는 대부분
“못 찾아서”가 아니었다.

- 6개: 이미 나온 **같은 파일**의 다른 심볼
- 2개: 다른 파일, dense 14등·40등
- 1개 (`gg-g09`): 134등. 이번 라운드에서 그래프로 살릴 대상 아님

그래서 실험은 순위를 고정한 채 **패키징만** 비교했다.

| Arm | 페이로드 | 결정 셀 |
| --- | --- | --- |
| A | top 5만 (지금 제품) | identity/순서만 |
| B | A + 같은 파일 형제 | **4개 / 4096바이트** |
| C | A + 한 홉 파일/심볼 클러스터 | 4파일 / 4096바이트 |
| D | B+C | C 실패로 권한 없음 |

프로덕션 search/MCP/store는 변경하지 않았다. 구현은
`internal/relationdiag`과 `cidx dev relations packaging`에만 있다.

---

## 4. 라이브 결과

명령:

```text
env -u VOYAGE_API_KEY go run ./cmd/cidx dev relations packaging \
  --contract testdata/retrieval/relation-packaging-experiment-contract-v1.json \
  --output-dir .cidx/test/experiments/relation-packaging-v1
```

| | 완전한 쿼리 | 의미 |
| --- | ---: | --- |
| A | **27/40** | 토폴로지 기준선 (Stage F 31/40보다 빡센 정의) |
| B | **32/40** | 지정 sibling 6개 중 **5개** 회복 |
| C | **36/40** | nearby 2개 회복 + 금지된 `gg-g09`까지 들어옴 |

- top 5 identity/순서 불변
- B에서 labeled isolated extra 0
- 기존 완료 쿼리 회귀 0
- `gg-g06-commit-object`만 형제 cap에 안 닿음 (같은 파일 extra ~141개)
- C는 nearby는 살리지만 `object.Change`(134등)를 모든 격자에서 같이 넣음 → 한 홉 게이트 실패

자세한 쿼리 표는
[packaging experiment](evidence/phase-07/relation-packaging-experiment-r4.md).

27 vs 31: Stage F는 top 5의 grade-2 라벨이 그룹을 덮으면 완료로 센다.
이번 실험은 그룹에 적힌 `source_parent_ids`가 페이로드에 있어야 완료다.

---

## 5. 채택 / 기각

채택 (평가 전용, MCP 아님):

- 같은 파일 형제 **count 4 / 4096 body bytes**
- 계약: [`testdata/retrieval/relation-sibling-packaging-adopted-v1.json`](../../testdata/retrieval/relation-sibling-packaging-adopted-v1.json)
- digest `d0b288b321cee2b60a794a0a38d7134395381491c9ede8b02d1af09ff2d65250`

기각 / 권한 없음:

- 한 홉 기본 push
- Arm D
- 제품 그래프 경로
- 검색 순위, RRF, FTS, MCP 스키마 변경
- 닫힌 32케이스·40쿼리 결과 덮어쓰기. 질문·코호트를 바꾸면 새 질문
  세트 버전과 새 run으로 남긴다.
- 어시스턴트 최종답 A/B

닫아서 confirmation에 쓰면 안 되는 세트:

1. chi v5.3.1 + React Hook Form v7.85.0 — 32질문
2. go-git v5.19.1 + Zustand v5.0.14 + Memos v0.30.0 — 40질문 Stage E/F

---

## 6. 문서 지도 (여기서 찾아가기)

### 6.1 먼저 읽을 것

| 순서 | 문서 | 역할 |
| ---: | --- | --- |
| 1 | **이 파일** | 개요·결과·찾아가기 |
| 2 | [STATUS.md](STATUS.md) | 페이즈 상태의 권위 장부 |
| 3 | [remaining-work handoff](evidence/revision-4/remaining-work-review-handoff-r4.md) | 재개 체크리스트, 오너 결정 목록 |
| 4 | [packaging experiment](evidence/phase-07/relation-packaging-experiment-r4.md) | 라이브 40쿼리 숫자 |

### 6.2 계약과 산출물

| 문서 | 역할 |
| --- | --- |
| [experiment contract](../../testdata/retrieval/relation-packaging-experiment-contract-v1.json) | 실험 격자. digest `cb726ace…4c28` |
| [adopted sibling contract](../../testdata/retrieval/relation-sibling-packaging-adopted-v1.json) | 채택된 평가 셀 4/4096 |
| [RELATION-PACKAGING-NEXT.md](RELATION-PACKAGING-NEXT.md) | 실험 권한·게이트 문장 |
| `.cidx/test/experiments/relation-packaging-v1/` | 라이브 JSONL/decision (gitignore) |

### 6.3 왜 패키징인가

| 문서 | 역할 |
| --- | --- |
| [overlap/selection diagnostic](evidence/phase-07/relation-overlap-noise-diagnostic-r4.md) | 6 sibling / 2 nearby / 1 far |
| [Stage E/F](evidence/phase-07/relation-calibration-stage-ef-r4.md) | 40쿼리 닫힘, 정책 미선택 |
| [assistant validation handoff](RELATION-ASSISTANT-VALIDATION-HANDOFF.md) | A/B 연기, 패키징이 다음 질문 |

### 6.4 그래프는 왜 제품이 아닌가

시간순. 전부 평가 sidecar. 제품 검색에 넣지 않기로 닫힘.

| 문서 | 결론 |
| --- | --- |
| [usage graph](evidence/phase-07/relation-usage-graph-diagnostic-r4.md) | 증거는 찾음. selector는 30/32 |
| [edge metadata](evidence/phase-07/relation-edge-metadata-diagnostic-r4.md) | 메타 dense-first 31/32. graph-first 기각 |
| [value parameter](evidence/phase-07/relation-value-parameter-diagnostic-r4.md) | 공통 패턴 분류. X08 미해결. 정책 보류 |
| [anchor/edge strength](evidence/phase-07/relation-anchor-edge-strength-diagnostic-r4.md) | 형식 32/32. 노이즈 번들 많음 |
| [frontier cap](evidence/phase-07/relation-frontier-cap-diagnostic-r4.md) | per-bucket top-2는 복잡도 제어일 뿐 |
| [graph-only Pareto](evidence/phase-07/relation-graph-only-pareto-diagnostic-r4.md) | 32/32지만 useful 7/17. 제품 기각 |
| [graph journal](RELATION-GRAPH-EXPERIMENT-JOURNAL.md) | chi/RHF 그래프 조사 전체 |

엣지 메타(zone, role, flow, tier, 횟수, dense 순위)는 이미 있다. `gg-g09`를
못 가린 것은 메타 부족이 아니라, 한 홉이면 넣는 규칙이 그 정보를 안 썼기
때문이다.

### 6.5 제품·평가 본선

| 문서 | 역할 |
| --- | --- |
| [implementation plan index](README.md) | 페이즈 00–14 |
| [FTS/assistant work journal](FTS-REMEDIATION-AND-ASSISTANT-AB-JOURNAL.md) | AND 결함부터 V3 A/B와 다음 작업까지 시간순 재개 지침 |
| [EVALUATION-CONTRACT.md](EVALUATION-CONTRACT.md) | 스테이지별 분모, 프로모션 게이트 |
| [EVALUATION-EMBEDDING-EXECUTION-PLAN.md](EVALUATION-EMBEDDING-EXECUTION-PLAN.md) | 유료 임베딩 순서 |
| [06 FTS search](06-fts-search.md) | 완료 — 자연어 후보 진입 개선 |
| [lexical planner review](evidence/phase-07/natural-language-fts-query-planner-review-r4.md) | 현재 구현·평가 목표와 ChatGPT/Grok 검토 |
| [lexical planner v2 rerun](evidence/phase-07/natural-language-lexical-rerun-v2.md) | 단계별 교정, 실행 lineage, 최종 `30/44`, 남은 6+8 실패 분리 |
| [07 lexical evaluation](07-lexical-evaluation.md) | 동일 v2 재실행 완료; assistant A/B handoff |
| [12 retrieval evaluation](12-retrieval-evaluation.md) | `core_retrieval` — confirmation 대기 |
| [14 packaging/hosts](14-packaging-and-host-integration.md) | `release_candidate` — 12 + 호스트 대기 |
| [chi/RHF freeze](evidence/phase-07/dual-ai-calibration-freeze-r4.md) | 32케이스 닫힘 |
| [Phase 07 evidence index](evidence/phase-07/README.md) | 07 증거 목록 |
| [Revision 4 evidence](evidence/revision-4/README.md) | R4 화해 경계 |
| [root README](../../README.md) | 제품 소개, Start here |

### 6.6 코드

| 경로 | 역할 |
| --- | --- |
| `internal/relationdiag/packaging.go` | 실험 엔진·게이트 |
| `internal/relationdiag/packaging_adopted.go` | 채택 계약 상수 |
| `internal/relationdiag/packaging_test.go` | 픽스처 게이트 + 계약 freeze 테스트 |
| `internal/devlab/relation_packaging.go` | `cidx dev relations packaging` |
| `cmd/cidx` | 엔트리. `dev`만 위 CLI를 노출 |

---

## 7. 일부러 안 한 것

| 작업 | 이유 |
| --- | --- |
| 새 confirmation 코퍼스 | 현재 순서에서는 하지 않음. Phase 12 promotion 확인은 별도 후속 |
| Phase 12 공식 프로모션 | confirmation 없이 `core_retrieval` 불가 |
| Phase 14 출시 후보 | 12 + 호스트/어시스턴트 증거 필요 |
| 형제를 MCP에 올리기 | 별도 제품 설계 전까지 평가 계약만 |
| 한 홉을 순위/role로 다시 맞추기 | 닫힌 세트 튜닝 |
| unchanged 어시스턴트 A/B 반복 | V3에서 정확도 보존과 구조적 응답량 증가가 이미 확인됨. 먼저 응답량을 교정해야 함 |

---

## 8. 오너가 정해야 재개되는 것

핸드오프 [§5](evidence/revision-4/remaining-work-review-handoff-r4.md#5-owner-decisions-required-before-work-resumes)와 같다.

현재 lexical 교정과 assistant A/B V3에는 남은 오너 결정이 없다. 다음 작업은
retrieval 정책을 다시 바꾸거나 새 저장소를 추가하는 것이 아니라, 현재 MCP
`search` 응답에서 text/structured content/metadata/inline body가 각각 얼마나
모델 입력을 늘리는지 계측하는 것이다. 공개 MCP 응답 스키마 변경이 필요하면
설계·Phase 13/14·호환성 계획을 먼저 갱신하고 오너 결정을 받아야 한다.

응답량 교정 후에는 새 V4 manifest로 전체 24-turn paired batch를 실행한다.
V1–V3 결과는 보존하며 selective rerun을 하지 않는다. 작은 `k`, 작은 inline
budget, 호출 횟수 제한은 실험 후보이지 아직 제품 기본값이 아니다.

아래 결정은 assistant A/B 이후의 별도 Phase 12/14 범위로 남는다.

1. confirmation 저장소, 핀된 커밋, 라이선스
2. 90쿼리 플로어(Go/TS/TSX 각 30 + hard-negative 18) 유지 여부
3. 형제 4/4096을 나중에 MCP에 넣을지 (기본: 아니오)

위 Phase 12/14 결정은 assistant A/B의 선행조건이 아니다. 다만 그 결정을
받기 전에는 새 confirmation 질문 작성, 코퍼스 클론, Voyage 실행, 추가
검색 정책 튜닝을 시작하지 않는다.

Confirmation을 돌리게 되면 절차는 핸드오프
[§6](evidence/revision-4/remaining-work-review-handoff-r4.md#6-confirmation-intake-do-not-execute-yet).

---

## 9. 로컬 아티팩트 (커밋되지 않음)

```text
.cidx/test/experiments/relation-packaging-v1/decision.json
.cidx/test/experiments/relation-calibration-review-v1/stage-f-ba44-a/
.cidx/test/experiments/relation-calibration-review-v1/frozen-ba44/
.cidx/test/states/{go-git,zustand,memos}-1024-int8/evaluations/relation-completion-stage-b-*-v2/
.cidx/test/assistant-ab/runs/assistant-ab-v3-20260820T143000Z/
```

`.cidx/credentials.env`는 읽거나 커밋하지 않는다.
