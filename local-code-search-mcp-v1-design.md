# 로컬 코드 검색 MCP v1 설계

상태: 초안. 구현 전 의사결정 문서.
날짜: 2026-08-14
가칭 바이너리: `cidx` (이름은 저장소를 열 때 바꿔도 된다)
위치: 이 문서는 Knowledge Base 제품 문서가 아니다. 구현은 별도 저장소에서 한다.

---

## 1. 이 문서가 하는 일

로컬에서 코드베이스를 인덱싱하고, MCP로 함수 단위 원문을 돌려주는 작은 검색기를 만들기로 한 이유를 적는다. 처음에 본 오픈소스와 무엇이 달랐는지, 기존 지식 베이스를 왜 쓰지 않는지, v1에서 무엇을 잠글지, 공개 저장소 구조를 어떻게 나눌지를 한곳에 둔다.

제품 한 줄은 이것이다.

> Go/TypeScript 함수를 AST로 잘라 SQLite와 FTS5에 로컬 인덱싱하고, 명시적으로 승인할 때만 OpenAI로 임베딩하며, 렉시컬과 벡터 검색을 섞어 원문을 돌려준다.

로컬 AST+FTS 인덱스 기본은 수동이고 커밋 훅은 옵션이다. 커밋 훅은 외부 API를 호출하지 않는다. 파일 감시로 임베딩하거나, dirty 파일을 위한 이중 인덱스, HNSW, 다언어 파서, 리뷰/위키/리스크 툴은 v1에 넣지 않는다.

---

## 2. 처음에 본 것

출발점은 [code-review-graph](https://github.com/tirth8205/code-review-graph)(이하 CRG)였다. 개발 중에 AI가 코드를 그래프처럼 찾아보게 해서 토큰을 줄인다는 접근이다. Tree-sitter로 저장소를 파싱하고, 함수·호출·import를 SQLite 그래프에 넣은 뒤 MCP로 제공한다. 로컬 우선, 증분 업데이트, 리뷰 시 영향 반경을 계산한다는 설명이 붙어 있다.

표면만 보면 로컬 코딩 보조에 가까워 보였다. 인덱스를 한 번 만들어 두고 에이전트가 필요한 조각만 읽게 한다는 문제 설정 자체는 타당하다. 다만 제품이 그 문제를 푸는 방식과, 그걸 증명한다고 내세운 숫자가 어색했다.

### 2.1 벤치마크가 어색했던 이유

헤드라인은 중앙값 약 65배, 최대 376배 토큰 감소다. 분모는 저장소 전체 소스 토큰이고, 분자는 질문당 검색 히트 몇 개와 이웃 엣지다. FastAPI가 376배로 가장 크게 나온 것도 분모가 약 95만 토큰이기 때문이다. 실제 에이전트는 저장소 전체를 읽지 않는다. CRG도 이걸 알고 식별자 grep 후 상위 3파일을 읽는 `agent_baseline`을 따로 만들었지만, 그 숫자는 공개하지 않았다.

리뷰에 가까운 벤치마크는 반대 방향이다. 커밋에서 바뀐 파일만 읽는 비용과 그래프 리뷰 JSON을 비교하면, 작은 커밋에서는 그래프 응답이 더 크다. 비율이 1 미만으로 나오는 것을 버그가 아니라고 적어 두었다.

임팩트 recall 1.0은 같은 그래프에서 만든 정답이다. 바뀐 파일과 그 파일로 들어가는 call/import 엣지를 정답으로 두고 그 엣지를 다시 따라가니 놓칠 수가 없다. 같은 커밋에서 같이 바뀐 다른 파일로 채점하는 co-change 모드는 `predicted_files = 0`으로 깨져 있어 인용하지 않는다고 되어 있다.

멀티홉 0.909는 6개 저장소, 수작업 11문항이다. 당일 점수가 0.545에서 올라간 이유는 본문 임베딩이 아니라 식별자를 단어로 쪼개고, 쿼리에서 뽑은 식별자가 이름에 있으면 2.0배 가산점을 준 것이다. 평가셋에 맞춘 휴리스틱에 가깝다.

검색 품질은 스스로 MRR 0.35라고 적었고, Express는 모듈 네이밍 때문에 0히트가 난다. 실행 플로우 탐지 recall은 33%다.

### 2.2 임베딩과 검색이 약했던 이유

FAQ에 그대로 있다. 임베딩은 선택이고, 하는 일도 시작 노드를 찾는 보조다. 넣는 텍스트는 함수 시그니처 수준, 노드당 약 10토큰이고 본문은 넣지 않는다. 예전에는 `"{name} {kind} in {parent}"`만 넣었고, 지금은 점 표기, 식별자 분리, 모듈 디렉터리 정도를 더한다. File 노드는 임베딩하지 않는다.

그래서 “authentication이 어떻게 동작하나” 같은 질문은 본문 의미가 아니라 이름·경로 어휘에 걸린다. 그 정도면 임베딩을 쓸 이유가 거의 없다. 본문을 함수 단위로 나누고, 식별자와 함께 검색할 수 있어야 의미가 있다.

FTS5/BM25도 “있다”와 “본문을 찾는다”는 다른 이야기다. 인덱스는 노드 메타데이터 위주이고, 점수를 끌어올린 것도 BM25가 아니라 식별자 부스트다. 코드 본문 BM25와 벡터를 융합했다고 보기 어렵다.

### 2.3 그래프를 점프하는 모델

옵시디언이나 GraphRAG 계열과 같은 그림이다. 작은 맵에서는 멋지고, 노드가 수백·수천이 되면 시드가 빗나간 뒤로는 길을 잃는다. 관계를 더 많이 붙인다고 답이 좋아지지 않고, 그래프만 검색하면 의미 대응이 약하다. 이미 지식 베이스 쪽에서 GraphRAG와 retrieval 실험을 보며 같은 결론을 낸 적이 있다. 검색 커버리지와 생성 모델이 실제로 쓰는 컨텍스트는 별개다. 후보를 넓히는 것과 답을 맞게 만드는 것은 다른 문제다.

CRG에서 값어치가 있는 부분은 에이전트가 문서를 점프하는 연출이 아니다. Tree-sitter로 뽑은 호출·import·테스트 관계의 결정적 조회다. “이 함수를 누가 부르나”는 유사도 문제가 아니라 구조 질의다. 그 한 가지는 본문 임베딩으로 대체되지 않는다. 다만 그 위에 Leiden 커뮤니티, betweenness, surprise scoring, 위키 생성, 약 30개 MCP 툴을 얹고 토큰을 65배 줄인다고 포장한 제품이다.

파이썬 구현 자체는 큰 문제가 아니었다. Django급 약 3천 파일 빌드가 약 40초, 검색은 밀리초다. 느린 쪽은 인터프리터가 아니라 에이전트가 MCP를 여러 번 돌며 홉하는 루프와, 훅마다 프로세스를 띄우는 비용이다.

### 2.4 CRG가 넓게 연 것들, 그리고 왜 따라가지 않는지

그 문장은 프로그래밍 언어 런타임 이야기가 아니다. CRG가 제품에 올려 둔 대상 언어 커버리지와 MCP 툴 개수, 검색과 상관없는 그래프 부가기능이다.

**언어 약 35개**는 파서 표면이다. Python, JS/TS, Go뿐 아니라 Rust, Java, C/C++, C#, VB.NET, Ruby, Kotlin, Swift, PHP, Scala, Solidity, Dart, R, Perl, Lua, Objective-C, 셸, Elixir, Zig, PowerShell, Julia, ReScript, GDScript, Nix, Verilog, SQL, Terraform, Ansible, Vue/Svelte, Astro, Jupyter, Perl XS까지 열어 둔다. Tree-sitter 문법만 넣으면 “지원”이라고 쓰기 쉽다. 실제로 필요한 것은 언어마다 함수·메서드·타입을 어떻게 자를지, 심볼과 정규명을 어떻게 만들지, 생성 코드와 매크로를 어떻게 제외할지, 그 언어 저장소에서 hit@5가 나오는지다. 넓게 연 뒤 Express 검색 0히트, 플로우 recall 33%가 나온 상태가 그 결과다. 공개 저장소에서 처음부터 이 표면을 열면 이슈의 대부분이 “내 언어에서 함수가 안 잘려요”가 된다.

**MCP 툴 30개**는 검색 하나가 아니다. 그래프 빌드, 후처리, 최소 컨텍스트, 임팩트 반경, 리뷰 컨텍스트, callers/callees, BFS/DFS, 의미 검색, 임베딩 계산, 큰 함수 찾기, 실행 플로우, 커뮤니티, 아키텍처 개요, 변경 리스크, 허브/브리지, 지식 공백, 의외의 연결, 추천 질문, 리팩터 미리보기/적용, 위키, 멀티레포 검색이 한 서버에 붙어 있다. 에이전트는 툴 스키마만으로도 토큰을 쓰고, 틀린 툴을 고르면 그 응답을 읽고 다시 호출한다. CRG가 줄이겠다는 토큰이 여기서 다시 센다. 그래서 그들 자신도 `--tools`로 부분집합을 켜게 만들어 두었다.

**커뮤니티 탐지, 리스크 점수, 위키**는 검색 품질과 거의 무관한 그래프 장식이다. Leiden으로 군집을 나누고, 디프에 점수를 매겨 PR 코멘트를 달며, 군집을 마크다운으로 풀어 쓴다. 호출 그래프가 틀리면 군집도 틀리고, 위키는 금방 낡는다. 허브/브리지, surprise scoring, knowledge gap, Obsidian/Neo4j보내기, 메모리 루프도 같은 층이다.

파서 언어를 잔뜩 열고, MCP 툴을 잔뜩 열고, 군집·리스크·위키를 얹는 순간 검색기가 아니라 CRG가 된다. 공개할수록 README가 길어지고 별은 늘 수 있지만, 실제 쓰는 경로인 `search → 원문`은 약해진다.

---

## 3. 지식 베이스를 왜 개조하지 않는지

한동안은 기존 지식 베이스를 로컬용으로 줄여 쓰는 편이 낫지 않나 생각했다. 이미 있는 축이 CRG보다 검색 문제에 가깝기 때문이다. 원문을 청크로 나누고, 본문을 임베딩하고, Item/Chunk를 세대로 고정하며, 그래프는 답을 만드는 출처가 아니라 시드 뒤의 라우팅으로 본다.

그 판단에서 바꾼 것은 “검색 원리를 버리자”가 아니다. **제품을 하나로 합치지 말자**는 쪽이다.

지식 베이스는 Workspace, Collection, 문서 수명주기, 벡터 세대, 권한, 워커, Postgres를 가진 독립 시스템이다. 로컬 코딩 보조 검색기는 그 계약이 필요 없다. 스키마를 공유하거나 패키지를 가져오면, 작은 도구가 큰 제품의 배포 단위를 따라가게 된다. 코딩 에이전트 훅, MCP 호스트 설정, PR 리뷰 같은 것을 지식 베이스에 넣는 것도 반대 방향의 오염이다.

가져올 것은 아이디어뿐이다.

- 검색 단위는 본문 청크다. 이름만 임베딩하지 않는다.
- 렉시컬과 벡터를 같이 쓴다. 평균이 조금 오른다고 바로 복잡도를 넣지 않는다.
- 로컬 인덱스 프로필과 임베딩 프로필을 따로 잠근다. 프로필이 바뀌면 파일 해시가 같아도 해당 층을 무효화한다.
- 결과는 모델이 다시 쓰지 않는다. 원문이 근거다.
- 그래프는 주경로가 아니다.

구현은 Go 단일 바이너리, SQLite, MCP로 처음부터 다시 짠다. 지식 베이스 저장소에 패키지를 추가하지 않는다.

---

## 4. 의사결정 기록

### 4.1 로컬의 의미

로컬은 검색기와 인덱스가 사용자 기계에서 돈다는 뜻이다. 생성 LLM을 로컬에 올리는 것도, 임베딩 모델을 기기에 넣는 것도 아니다. 코딩 보조 검색기에 로컬 임베딩 런타임을 넣으면 설치만 무거워진다. 임베딩 입력은 공식 OpenAI API로 보낸다.

코드 검색 결과는 AI로 재정리하지 않는다. 원문과 줄번호가 근거이고, 한 번 더 정리하면 토큰만 늘고 위치가 어긋난다.

### 4.2 OpenAI API와 임베딩 프로필

공식 검증 모델은 OpenAI만 둔다. Voyage는 코드 쪽 성능은 있어도 키를 가진 사용자가 적고, Qwen은 셀프호스트 쪽, Gemini 임베딩은 가격과 안정성을 공식 기본값으로 두기 부담스럽다.

API를 쓰면 엔진을 내장할 일이 없다. v1 프로덕션 경로는 공식 OpenAI Embeddings API만 지원한다. 고정하는 것은 모델 바이너리가 아니라 임베딩 프로필이다.

- provider: OpenAI Embeddings API
- model: `text-embedding-3-small` 또는 `text-embedding-3-large`
- dimensions: 256 (두 모델 모두 Matryoshka)
- metric: cosine
- 입력 텍스트: 경로 + 시그니처 + 본문
- 저장/서빙 벡터: int8

인덱스를 처음 만들 때 모델, 차원, 계량, 임베딩 텍스트 포맷, 양자화 포맷을 함께 잠근다. 이 중 하나가 바뀌면 AST+FTS는 그대로 사용하고 기존 벡터만 전부 `pending`으로 돌려 다시 임베딩한다. v1 프로덕션 설정에는 custom endpoint나 `base_url`을 열지 않는다.

small과 large를 같이 여는 이유는 품질이 아니라 비용이다. 입력 백만 토큰 기준 small은 약 $0.02, large는 약 $0.13로 대략 6.5배다. 차원이 256이어도 요금은 같다. 기본값은 small, 품질이 아쉬울 때 large로 인덱스를 다시 짓는다.

256차원 int8은 용량을 무시하지 않기 위해 고른 값이다. 청크 10만 개 기준 약 25MB, 20만 개면 약 50MB다. 프로덕션 DB에는 int8만 저장한다. 양자화 품질을 비교하는 평가에서만 API의 float32 응답을 별도 평가 산출물로 저장할 수 있으며, 서버는 그 파일을 읽지 않는다. 20만 청크는 지원 보장이 아니라 초기 성능 측정용 상한 예시다.

### 4.3 청크와 렉시컬

검색 단위는 파일이 아니라 AST 함수·메서드·타입이다. 3천 줄 파일이어도 함수가 잘게 나뉘면 검색된다. 반대로 짧은 파일에 함수 하나가 수백 줄이면 청크가 뭉개진다. 문서에 “파일을 작게 유지하세요”라고 쓰지 않는다. 쓸 문장은 이쪽이다.

- 검색 단위는 AST 함수/메서드/타입이다.
- vendor, generated, minified, lockfile은 인덱스에서 뺀다.
- 한 함수가 수백 줄을 넘기면 그 청크는 검색과 토큰 모두 불리하다.

렉시컬은 함수명, 타입명, 메서드명만 높은 가중으로 넣는다. 상수와 변수는 exported 여부와 무관하게 독립 청크나 고가중 심볼로 만들지 않는다. 다만 `export const handler = () => ...`처럼 변수 선언에 묶인 함수 표현식은 함수 청크로 보고 `handler`를 함수 심볼로 쓴다. 함수 본문에 들어 있는 지역 이름은 `body`의 낮은 가중으로만 검색된다. FTS5 컬럼은 `symbols`(가중 높음)와 `body`(가중 낮음)로 나눈다. camelCase/snake_case는 원문과 분해형을 같이 넣는다.

타입 청크와 그 안의 메서드 청크는 같은 본문을 중복해서 넣지 않는다. 언어별 청커 계약에서 타입의 구조 정보와 메서드 본문 경계를 고정한다.

### 4.4 언어

공식 파서는 Go와 TypeScript/TSX만 둔다. 운영 중인 프로젝트에 Python이 없고, 공개 첫 릴리스에서 언어를 넓히면 파서 민원이 검색 품질을 먹는다. JavaScript는 같은 문법으로 받기 쉽지만, 첫 평가 코퍼스는 `.go` / `.ts` / `.tsx`로 닫는다.

### 4.5 인덱스 시점과 비용

매 저장마다 임베딩하면 API 비용이 바로 센다. 감시 데몬이 임베딩까지 하면 MCP 주제에 상주 프로세스가 된다. 반대로 커밋만 보면 지금 고치는 파일이 인덱스에 없다.

그래서 갱신을 하나로 묶지 않고 비용으로 나눈다.

| | AST + FTS5 | 임베딩 |
| --- | --- | --- |
| 비용 | 로컬, 거의 없음 | API |
| v1 | `cidx index` 또는 MCP `reindex`가 즉시 반영 | `cidx embed --apply`로만 반영 |
| on-commit | 커밋 뒤 로컬 증분 | 실행하지 않음 |
| 나중 | dirty 파일을 검색 직전 파싱해도 됨 | 계속 명시적 실행 |

v1 기본은 수동이다. 감시하지 않는다. `status`가 로컬 인덱스 커밋/해시, 워킹트리에서 아직 반영되지 않은 파일, 임베딩 대기/실패 수를 따로 말한다. 에이전트나 사용자는 먼저 무료 로컬 인덱스를 갱신한다. 벡터가 아직 없어도 FTS 검색은 최신 상태로 동작한다.

모드 두 개.

- `manual` (기본): CLI/MCP로만 AST+FTS 증분
- `on-commit`: post-commit 훅이 같은 로컬 증분만 돌린다

저장할 때마다 도는 watch는 v1에 넣지 않는다. 안정 인덱스와 dirty 임시 인덱스를 나누는 이중 레이어도 v1에 넣지 않는다. 다만 `status`에 dirty/stale 목록을 지금부터 넣어, 나중에 레이어를 얹을 때 스키마를 다시 짜지 않는다.

로컬 재파싱 단위는 커밋이 아니라 파일 해시(SHA-256)다. 커밋은 언제 돌릴지이고, 파일 해시는 어떤 파일을 다시 파싱할지다. 임베딩 대상은 로컬 파싱 뒤 별도로 결정한다. 트리 전체 해시 비교는 싸므로 git diff에만 의존하지 않는다. on-commit 모드도 내부는 같은 로컬 해시 증분이다.

`cidx embed`는 기본적으로 대기 segment 수, 대략 입력 토큰과 USD만 보여 주고 끝난다. `--apply`를 명시했을 때만 과금한다. MCP `reindex`는 AST+FTS만 갱신하며 임베딩 API를 호출하지 않는다.

### 4.6 검색과 ANN

검색은 FTS5 + 256차원 int8 브루트포스 + RRF다. 브루트포스가 감당할 수 있는 청크 수는 실제 구현의 지연과 메모리를 측정해 고정한다. HNSW는 v1에 넣지 않는다. 검색이 닫힌 뒤 사이드카 가속기로 연다. 벡터를 청크에 붙여 두면 그때 구조를 다시 짤 필요가 없다.

벡터가 없는 청크와 임베딩 API를 사용할 수 없는 검색도 정상 상태다. 이때는 FTS-only 결과를 반환하고 응답에 vector coverage와 fallback 이유를 붙인다.

내부 후보 `k=20`, MCP 기본 반환 `k=5`. hit@1 / hit@5는 서빙 파라미터가 아니라 평가 지표다. 파일 hit와 심볼 hit를 나눈다.

### 4.7 MCP 표면

툴은 네 개다.

| 툴 | 하는 일 |
| --- | --- |
| `status` | 두 프로필, 인덱스 커밋, 파일/청크 수, stale/dirty, vector coverage/pending/failed |
| `search` | 하이브리드 검색, 함수 원문 상위 k |
| `read_span` | 경로와 줄범위 원문 |
| `reindex` | 로컬 파일 해시 증분과 AST+FTS 반영. 외부 API 호출 없음 |

이 네 개가 에이전트 행동을 고정한다. 질문이 오면 검색하고, 부족하면 줄을 더 읽고, 인덱스가 낡았으면 상태를 보고 증분을 결정한다.

---

## 5. v1에서 하지 않는 것

- HNSW, 파일 watch 임베딩, dirty 보조 인덱스
- Python 및 그 외 언어 공식 지원
- OpenAI 외 제공자와 custom OpenAI-compatible endpoint를 열기
- 검색 결과 LLM 재작성, 요약, 위키
- 호출 그래프를 주 검색 경로로 쓰기
- 커뮤니티 탐지, 리스크 점수, 허브/브리지, 멀티레포 데몬
- MCP 툴 확장
- 지식 베이스 패키지, 스키마, API 재사용

호출 반경이 나중에 필요하면 검색 다음의 얇은 레이어로 따로 둔다. 탐색의 주경로로 올리지 않는다.

---

## 6. 공개 저장소에서 구조를 나누는 이유

설치 한 줄, 호스트별 설정, 파서, 임베딩, 검색, MCP가 한 패키지에 섞이면 나중에 호스트가 늘 때마다 코어가 흔들린다. 요즘 MCP 도구는 서버와 설치를 나눈다. 서버는 stdio만 알고, `install --host …`가 Claude Code, Cursor, VS Code, Codex 쪽 JSON을 쓴다. CRG도 그 패턴이고, 호스트 목록만 잔뜩 늘린 점이 문제다.

이 도구도 같은 패턴을 쓰되, **호스트 지식은 어댑터에만** 둔다. 검색 코어는 Cursor가 있는지 모른다. 새 호스트를 추가하는 일은 어댑터 파일 하나와 등록 한 줄이다.

바이너리는 하나다. `cidx`가 CLI와 MCP 서버와 installer를 같이 제공한다. 배포 단위를 나누지 않는다. 나누는 것은 패키지 경계다.

가칭 모듈 경로: `github.com/<org>/cidx` (org와 이름은 저장소를 열 때 정한다).

---

## 7. 저장소 배치

```text
cidx/
  README.md
  LICENSE
  go.mod
  cmd/
    cidx/
      main.go                 # 서브커맨드만 연결
  internal/
    app/                      # CLI: init, status, index, embed, serve, install, uninstall
    config/                   # index/embedding 프로필 잠금과 무효화, 경로, ignore
    store/                    # SQLite 스키마, 마이그레이션, 쿼리
    chunk/                    # AST 청킹 인터페이스
      lang.go
      golang/
      typescript/
    symbol/                   # 함수/타입/메서드 FTS 심볼 추출
    embed/                    # 공식 OpenAI API 클라이언트, 계획, 배치, 재시도
      openai.go
    index/                    # 로컬 파일 나열, 해시 증분, 청크/FTS 반영, 삭제
    search/                   # FTS5, 벡터 스캔, RRF, 응답 조립
    mcp/                      # 툴 스키마와 핸들러. 호스트를 모른다
    hostinstall/              # 호스트 어댑터. 검색을 모른다
      host.go
      spec.go
      claude.go
      cursor.go
      vscode.go
      registry.go
    ignore/                   # git ls-files + .cidxignore
    estimate/                 # 유료 임베딩 토큰/비용 추정
  testdata/
    golang/
    typescript/
  docs/
    v1-design.md              # 이 문서를 옮긴다
    hosts.md                  # 호스트별 설정 파일 위치
  .github/
    workflows/
      ci.yml
```

규칙.

- `internal/mcp`는 `internal/search`, `internal/index`, `internal/store`만 본다. `hostinstall`을 import하지 않는다.
- `internal/index`는 `internal/embed`를 import하지 않는다. 로컬 인덱싱은 API 키와 네트워크 없이 끝나야 한다.
- `internal/embed`는 대기 segment와 벡터만 다룬다. 청크와 FTS 생성을 맡지 않는다.
- `internal/hostinstall`은 설정 파일만 다룬다. SQLite와 임베딩을 import하지 않는다.
- 언어 추가는 `internal/chunk` 아래 패키지와 테스트 코퍼스다. 검색 코드를 수정하지 않는다.
- 공개 API를 만들기 위해 `pkg/`를 미리 열지 않는다. 쓸 곳이 생기면 그때 연다.
- 지식 베이스 모듈을 `replace`하거나 import하지 않는다.

---

## 8. 데이터와 프로필

인덱스 디렉터리 기본값: 저장소 루트의 `.cidx/`. gitignore에 넣는다. `--data-dir`로 바깥에 둘 수 있다.

```text
.cidx/
  config.json          # 잠긴 두 프로필과 런타임 설정. 사람/도구가 읽는다
  index.db             # SQLite
```

`config.json`의 프로필과 런타임 설정 초안.

```json
{
  "version": 1,
  "index_profile": {
    "version": 1,
    "languages": ["go", "typescript"],
    "chunkers": {"go": 1, "typescript": 1},
    "symbols_version": 1,
    "fts_version": 1
  },
  "embedding_profile": {
    "version": 1,
    "model": "text-embedding-3-small",
    "dimensions": 256,
    "metric": "cosine",
    "text_format_version": 1,
    "quantization": "int8",
    "quantization_version": 1
  },
  "index_mode": "manual",
  "search": {
    "return_k": 5,
    "candidate_k": 20
  }
}
```

두 프로필은 canonical JSON의 fingerprint로 DB에도 저장한다. `index_profile`이 달라지면 파일 SHA가 같아도 AST+FTS를 전면 재생성한다. `embedding_profile`이 달라지면 AST+FTS는 유지하고 벡터만 전부 `pending`으로 바꾼다. `index_mode`, `return_k`, `candidate_k`는 무효화 대상이 아니다. 이전 config에 custom `base_url`이나 비공식 provider가 있으면 조용히 무시하지 않고 호환되지 않는 설정으로 거부한다.

API 키는 파일에 쓰지 않는다. 환경 변수 `OPENAI_API_KEY` 또는 `CIDX_OPENAI_API_KEY`. 프로젝트 MCP 설정에도 키를 넣지 않는다.

SQLite 초안.

- `meta`: 스키마 버전, 두 profile fingerprint, 인덱스가 가리키는 git HEAD(있으면), 마지막 로컬 인덱스/임베딩 시각
- `files`: path, sha256, mtime, language, status, index_generation
- `chunks`: file_id, symbol, kind, start_line, end_line, body
- `symbols`: chunk_id, name, folded_name
- `chunk_fts`: FTS5, 컬럼 `symbols`, `body`
- `embedding_segments`: chunk_id, segment_no, start_line, end_line, input_sha256, status, last_error
- `vectors`: segment_id, embedding_profile_fingerprint, int8 blob, codec metadata
- `index_runs`: 실행 시각, phase(local/embed), profile fingerprint, 파일/segment 수, 임베딩 토큰, 에러

마이그레이션은 번호 붙은 SQL로 둔다. 스키마를 코드에 문자열로 숨기지 않는다.

DB 스키마 마이그레이션과 프로필 무효화는 별개다. 테이블 모양이 같아도 청커나 심볼 규칙이 바뀌면 로컬 인덱스를 다시 만들고, 임베딩 입력이나 모델 규칙이 바뀌면 벡터만 다시 만든다.

ignore는 git 추적 파일만 기본으로 보고, `.cidxignore`로 vendor/generated를 더 뺀다. `node_modules`, `vendor`, `dist`, `*.min.js`, `go.sum`은 기본 제외 목록에 넣는다.

---

## 9. 인덱싱

로컬 인덱싱과 임베딩을 서로 다른 단계로 실행한다.

### 9.1 로컬 AST+FTS

1. 루트에서 대상 파일을 모은다.
2. 파일 sha256을 계산한다.
3. 새 파일·해시 변경 파일만 파싱한다. `index_profile`이 달라졌으면 파일 해시와 무관하게 전부 다시 파싱한다.
4. 변경 파일은 Tree-sitter로 함수/메서드/타입 청크를 만든다. 상수와 변수 선언은 독립 청크로 만들지 않는다.
5. 청크의 심볼과 FTS 본문을 만들고, 긴 청크에는 하나 이상의 embedding segment를 연결한다.
6. 각 segment가 실제 API에 보낼 canonical 입력과 `embedding_profile` fingerprint로 `input_sha256`을 계산한다. 같은 입력 해시의 기존 벡터가 있으면 재사용하고, 없으면 `pending`으로 둔다. 별도의 안정 chunk ID나 rename 추적은 만들지 않는다.
7. 새 청크, 심볼, FTS, segment 상태와 재사용 벡터를 파일 단위의 짧은 트랜잭션으로 교체한다. 사라진 파일의 데이터도 같은 로컬 단계에서 지운다.

파일이 바뀌면 파일 전체를 다시 파싱한다. 파싱은 로컬이고 싸기 때문에 청크 단위 증분 파서는 만들지 않는다. 반면 임베딩은 최종 API 입력이 완전히 같으면 다시 호출해도 얻는 정보가 없으므로 `input_sha256`이 바뀐 segment만 유료 대상이 된다. 원문용 `content_hash`는 v1 재사용 판단에 필요하지 않으며, 파일 stale 판정에는 `files.sha256`을 쓴다.

### 9.2 유료 임베딩

1. `pending` segment를 모아 예상 입력 토큰과 USD를 계산한다.
2. `cidx embed`는 기본적으로 예상치만 출력한다.
3. `cidx embed --apply`일 때만 공식 OpenAI Embeddings API를 호출한다. 배치는 모델 한도에 맞추고 실패는 segment 상태로 남긴다.
4. API의 float32 응답은 차원과 유효값을 검사한 뒤 int8로 양자화해 저장하고 폐기한다. 명시적인 평가 실행에서만 profile fingerprint를 붙인 별도 float32 산출물을 만들 수 있다.
5. 임베딩 실패는 이미 성공한 AST+FTS 갱신을 되돌리지 않는다.

`on-commit` 모드는 훅이 `cidx index --reason commit`을 호출하는 것뿐이다. 외부 API를 호출하지 않는다.

큰 함수는 하나의 source chunk를 유지하면서 AST statement 경계의 여러 embedding segment로 나눈다. 임의 바이트 기준으로 자르지 않는다. 검색은 맞은 segment와 부모 함수 범위를 함께 반환하고, 전체 함수가 필요하면 `read_span`으로 확장한다.

---

## 10. 검색

1. 쿼리에서 식별자형 토큰을 뽑는다. `GetUserByID` → 원문 + `get user by id`.
2. FTS5로 `symbols`와 `body`를 검색한다. `symbols` 가중을 높게 준다.
3. 준비된 벡터가 있으면 같은 쿼리를 임베딩한 뒤 int8 벡터와 cosine을 계산한다. v1은 전체 스캔이다. 키나 네트워크가 없거나 쿼리 임베딩이 실패하면 이 채널만 건너뛴다.
4. 벡터 리스트가 있으면 두 리스트를 RRF로 합치고, 없으면 FTS 순위를 그대로 쓴다. 식별자가 `qualified`/`symbol`에 정확히 있으면 작은 가산만 둔다. CRG처럼 평가셋에 맞춘 2.0배 부스트를 기본 정책으로 쓰지 않는다. 쓰려면 평가 코퍼스에서 증분을 보고 나중에 넣는다.
5. 상위 `return_k`개에 경로, 심볼, kind, 시작/끝 줄, 원문, 점수 출처(fts / vector / both), vector coverage와 fallback 이유를 붙인다.
6. 응답 전체 토큰 예상이 예산을 넘으면 본문을 자르고 `read_span`을 안내한다.

검색 경로에 LLM을 두지 않는다.

---

## 11. CLI와 MCP

```text
cidx init [--model small|large] [--mode manual|on-commit]
cidx status
cidx index [--dry-run]
cidx embed [--dry-run|--apply] [--eval-f32-out <path>]
cidx serve
cidx install [--host <id>|--detect] [--scope project|user] [--dry-run]
cidx uninstall [--host <id>] [--dry-run]
```

`serve`는 stdio MCP다. 호스트가 프로세스를 띄운다. v1에서 HTTP 서버를 열지 않는다.

MCP 도구 입력 초안.

`status` — 인자 없음. 두 profile fingerprint, 청크 수, HEAD, dirty_files, stale_files, vector_coverage, embeddings_pending/failed, last_local_index_at, last_embedding_at.

`search`

- `query` (필수)
- `k` (선택, 기본 5, 최대 20)

`read_span`

- `path` (저장소 상대 경로)
- `start_line`, `end_line`

`reindex`

- `dry_run` (bool, 기본 false)
- 응답: files_updated, chunks_updated, embeddings_pending. 외부 API 호출과 과금은 하지 않는다.

경로 탈출을 막는다. `read_span`은 인덱스 루트 밖을 읽지 않는다. 무시된 파일도 기본적으로 읽지 않는다.

---

## 12. 호스트 설치 어댑터

서버는 호스트를 모른다. 설치만 호스트를 안다. 이 분리가 공개 이후 플랫폼을 늘리는 방법이다.

```go
type Scope int // Project, User

type ServerSpec struct {
    Name    string // "cidx"
    Command string // 절대경로 또는 PATH의 cidx
    Args    []string
    Env     map[string]string // 키 값이 아니라 변수 이름만
    Cwd     string            // 저장소 루트
}

type Plan struct {
    Host        string
    Path        string
    Action      string // create, merge, skip, remove
    Before, After []byte
}

type Host interface {
    ID() string
    Detect() bool
    ConfigPath(root string, scope Scope) (string, error)
    PlanInstall(path string, spec ServerSpec) (Plan, error)
    PlanRemove(path string, name string) (Plan, error)
    Apply(plan Plan) error
}
```

`Apply`는 임시 파일에 쓴 뒤 rename한다. 원본 JSON/JSONC에서 이 서버 이름 항목만 바꾸거나 지운다. 다른 MCP 서버, 주석, 알 수 없는 필드는 유지한다. 파싱에 실패하면 쓰지 않는다.

v1에서 구현하는 호스트.

| id | 설정 위치 | v1 |
| --- | --- | --- |
| `claude-code` | 프로젝트 `.mcp.json`, 사용자 `~/.claude.json` | 구현 |
| `cursor` | 프로젝트 `.cursor/mcp.json`, 사용자 `~/.cursor/mcp.json` | 구현 |
| `vscode` | 프로젝트 `.vscode/mcp.json` | 구현 |
| 그 외 | 감지되면 스니펫만 출력 | 스텁 |

스텁도 `Host`를 구현한다. `Detect`와 `ConfigPath`는 두고, `PlanInstall`은 “이 JSON을 여기 붙여라”는 설명을 담은 Plan을 반환한다. 파일을 함부로 쓰지 않는다. 나중에 구현을 채우면 `cidx install --host windsurf`가 같은 커맨드로 동작한다.

`cidx install --detect`는 설치된 호스트만 보여 주고, 하나도 없으면 스니펫을 출력한다. 기본은 `--dry-run`이다. `--apply`가 있어야 파일을 쓴다.

프로젝트 스코프 설정에는 API 키를 넣지 않는다. 호스트가 부모 환경을 넘기므로, 사용자는 셸이나 호스트 시크릿에 키를 둔다. README에 그 순서를 적는다.

`uninstall`은 이 서버 이름 항목만 제거한다. 데이터 디렉터리 `.cidx/`는 `--purge`가 있을 때만 지운다.

호스트별 파일 위치와 JSON 모양은 자주 바뀐다. `docs/hosts.md`에 경로와 예시를 적고, 어댑터 테스트는 testdata 픽스처로 고정한다. 픽스처에는 이미 다른 MCP가 들어 있는 파일, JSONC 주석, 빈 파일을 넣는다.

---

## 13. 구현 순서

코어가 닫히기 전에 호스트를 늘리지 않는다.

1. `store` 스키마와 마이그레이션. 두 profile fingerprint, 파일·청크·segment·FTS·int8 벡터 왕복 테스트.
2. Go 청커와 심볼 추출. testdata로 함수/메서드/리시버가 잘리는지 고정.
3. TypeScript/TSX 청커. 같은 계약.
4. 로컬 `index` 해시 증분과 profile 무효화. 변경·삭제·재실행, 벡터 재사용과 pending 상태가 결정적인지 확인.
5. 공식 OpenAI API 클라이언트와 `embed`. 배치, 재시도, `dimensions=256`, float32 검증과 int8 양자화. 키 없이 테스트 가능하게 인터페이스를 둔다.
6. `search`: FTS5만, 그다음 벡터 스캔, 그다음 RRF. 벡터가 없거나 쿼리 임베딩이 실패한 FTS-only 경로도 고정한다.
7. 작은 Go 저장소와 작은 TypeScript 저장소에서 파일/심볼 hit@1 / hit@5를 평가한다. 이 단계에서 청킹·임베딩 텍스트·융합 방식이 닫히지 않으면 뒤의 어댑터로 넘어가지 않는다.
8. CLI: `init`, `status`, `index`, `embed`, `serve`.
9. MCP 네 툴. 인자와 경로 탈출, 응답 토큰 예산을 확인한다.
10. `hostinstall` 인터페이스와 Claude Code / Cursor / VS Code 어댑터. dry-run이 기본.
11. README: 설치, 키, `index`, 선택적 `embed --apply`, 호스트 연결, 하지 않는 것.

이 순서가 끝나기 전에 HNSW, watch, 언어 추가, 호스트 추가를 열지 않는다.

---

## 14. 평가

공개하려면 숫자가 CRG처럼 분모를 부풀리지 않아야 한다.

- 저장소 전체를 읽지 않은 척하지 않는다. 베이스라인은 식별자 grep 후 상위 파일, 또는 에이전트가 파일을 직접 연 토큰이다.
- 지표는 파일 hit@1/@5와 심볼 hit@1/@5다.
- 질문 세트, 인덱스가 가리키는 커밋, 두 profile fingerprint를 고정한다.
- small 256과 large 256을 같은 질문으로 비교할 수 있게 하되, v1 기본 프로필은 small이다.
- 같은 API float32 응답으로 평가 전용 float32 순위와 프로덕션 int8 순위를 비교한다. float32 산출물은 서버가 사용하지 않는다.
- 토큰 절감은 검색 응답 토큰과, 그 응답으로 답을 찾은 비율을 같이 적는다. 빗나간 검색은 절감이 아니다.
- 긴 청크를 `read_span`으로 확장하거나 검색을 다시 한 비용도 최종 컨텍스트에 포함한다.

---

## 15. 나중에 열 자리

자리를 만들어 두고 구현하지 않는 것.

- `vectors` 테이블 옆 ANN 사이드카. 프로덕션 int8 브루트포스 스캔은 남긴다.
- `status.dirty_files`를 받아 검색 직전에 AST+FTS만 치는 overlay.
- `hostinstall` 레지스트리에 Windsurf, Codex, Gemini CLI, Continue.
- `internal/chunk`에 Python.
- 호출 그래프는 청크 간 선택 테이블. 검색 RRF에 넣지 않은 채 `read_span` 보조로만.

이 다섯 자리가 비어 있어도 v1은 동작해야 한다.

---

## 16. 지식 베이스와의 관계

이 도구는 지식 베이스의 로컬 모드가 아니다. 문서를 지식 베이스 `docs/` 권위에 넣지 않고, 구현을 그 모듈에 붙이지 않는다. 검색 교훈만 가져온다. 설치 어댑터, MCP 툴, 코드 청킹, 로컬 인덱스 프로필과 공식 OpenAI API 임베딩 프로필은 이 저장소의 문제다.
