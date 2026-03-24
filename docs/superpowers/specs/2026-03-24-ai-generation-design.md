# AI-Powered Proposal & AC Generation for SaaS Users

## Problem

MCP 사용자는 `create_acceptance_criteria`, `write_spec` 등의 도구로 AI가 AC/proposal을 직접 생성할 수 있지만, SaaS 웹 사용자는 모든 내용을 수동으로 작성해야 한다. 특히 BDD 형식의 AC는 Given/When/Then 구조를 이해해야 해서 진입 장벽이 높다.

## Approach

하이브리드 패턴: Proposal 초안 생성(streaming) + Proposal 기반 AC 생성(일괄). Go 백엔드에서 eino 프레임워크로 LLM API를 직접 호출한다.

## Scope

- AI 설정 UI (provider, model, API key, context 옵션)
- Proposal AI 초안 생성 (streaming)
- AC AI 생성 (일괄)
- 미리보기 → 수락 플로우
- OpenAI + Anthropic provider 지원

## Out of Scope

- Inline suggestion (Copilot 스타일)
- Design/Spec 문서 AI 생성
- AI 기반 AC 자동 검증
- Colign managed API key (BYOK only)
- AI 피드백 루프 (👍👎) — 향후 별도 설계

---

## 1. AI 설정 (Project 레벨)

### 데이터 모델

새 테이블 `ai_configs`:

새 마이그레이션 파일 `migrations/000005_create_ai_configs.up.sql`:

```sql
CREATE TABLE ai_configs (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    project_id BIGINT NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT '',          -- "openai" | "anthropic"
    model TEXT NOT NULL DEFAULT '',             -- "gpt-4o", "claude-sonnet-4-20250514" 등
    api_key_encrypted BYTEA NOT NULL DEFAULT '',-- [key_version(1) || nonce(12) || ciphertext]
    key_version SMALLINT NOT NULL DEFAULT 1,   -- 암호화 키 버전 (rotation 지원)
    include_project_context BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### UI

프로젝트 설정 내 "AI" 탭:

- Provider 선택: dropdown (OpenAI / Anthropic)
- Model 선택: provider에 따라 동적 dropdown
  - OpenAI: gpt-4o, gpt-4o-mini
  - Anthropic: claude-sonnet-4-20250514, claude-haiku-4-5-20251001
- API Key 입력: password 필드 + "연결 테스트" 버튼
- "프로젝트 컨텍스트 포함" 토글: README, 기존 changes를 AI 프롬프트에 포함할지 여부

### API Key 보안

- DB 저장 시 AES-256-GCM 암호화
- Ciphertext 포맷: `[key_version(1byte) || nonce(12bytes) || ciphertext]`
- 환경변수 `AI_ENCRYPTION_KEY`에 encryption key 저장. `Config` struct에 `AIEncryptionKey` 필드 추가
- Key rotation: `key_version` 컬럼으로 버전 관리. 새 키 도입 시 기존 버전으로 복호화 → 새 버전으로 재암호화 가능
- API 응답 시 마스킹 (예: `sk-...xxxx`)
- Save 시 빈 문자열이면 기존 암호화 값 유지 (마스킹된 값을 다시 암호화하는 실수 방지)

### 연결 테스트

- `TestConnection` RPC: `{provider, model, api_key}` → 최소한의 API 호출 (예: 짧은 completion 요청)로 유효성 확인
- 테스트 시 key를 저장하지 않음 — 저장은 별도 Save 액션

---

## 2. Proposal AI 생성

### 트리거 UI

- **빈 상태**: StructuredProposal의 4개 섹션이 모두 비어있으면, 기존 폼 대신 empty state CTA 표시 — "이 change를 한 줄로 설명해주세요" 입력 + "AI로 초안 작성" 버튼
- **내용 있을 때**: 섹션 상단에 작은 "다시 생성" 버튼. 클릭 시 "기존 내용이 대체됩니다" 확인 dialog

### 생성 플로우

1. 사용자가 한 줄 설명 입력 (예: "결제 시스템에 정기구독 기능 추가")
2. "AI로 초안 작성" 클릭
3. 백엔드 `POST /api/ai/generate-proposal` 호출 — SSE로 streaming 응답
4. 미리보기 패널 등장 — 4개 섹션이 streaming으로 채워짐
5. streaming 완료 후 하단에 "적용" / "다시 생성" / "취소" 버튼
6. "적용" → `documentClient.saveDocument()`로 proposal 저장, 폼으로 전환

### SSE Streaming

- Go 백엔드: eino `ChatModel.Stream()` → SSE endpoint
- 프론트엔드: `fetch` + `ReadableStream` + `AbortController`로 수신 (unmount 시 정리)
- SSE response headers:
  ```
  Content-Type: text/event-stream
  Cache-Control: no-cache
  Connection: keep-alive
  X-Accel-Buffering: no   # nginx/k3s ingress 버퍼링 방지
  ```
- LLM에게 JSON이 아닌 **섹션 delimiter 방식**으로 출력 지시:
  ```
  ---SECTION:problem---
  이 기능은...
  ---SECTION:scope---
  포함 범위는...
  ---SECTION:outOfScope---
  ...
  ---SECTION:approach---
  ...
  ```
- 백엔드가 delimiter를 파싱하여 SSE event로 변환:
  ```
  data: {"section": "problem", "chunk": "이 기능은..."}
  data: {"section": "scope", "chunk": "포함 범위는..."}
  data: [DONE]
  ```
- 프론트엔드는 section별로 chunk를 누적하여 미리보기에 표시. "적용" 시 누적된 텍스트로 ProposalSections JSON을 구성하여 저장

### 프롬프트

```
System: You are a product specification writer for software projects.
Given a one-line description, generate a structured proposal.
Output each section with a delimiter line, then the content:

---SECTION:problem---
What problem does this solve? Why now?
---SECTION:scope---
What's included in this change?
---SECTION:outOfScope---
What's explicitly excluded?
---SECTION:approach---
High-level implementation approach

Write in the same language as the user's input.

{include_project_context가 true면}
Project context:
- README: {project README, max 2000자로 truncate}
- Recent changes: {최근 change 목록, 최대 10개}
{/if}

User: {사용자의 한 줄 설명}
```

---

## 3. AC AI 생성

### 트리거 UI

- **빈 상태**: AC 리스트가 비어있고 proposal이 존재하면, "Proposal 기반 AC 자동 생성" CTA 버튼 표시
- **내용 있을 때**: AC 섹션 상단에 "AI로 추가 생성" 버튼. 기존 AC를 삭제하지 않고 추가
- **Proposal 없으면**: CTA 비활성화 + "먼저 Proposal을 작성해주세요" 안내

### 생성 플로우

1. "AC 자동 생성" 클릭
2. 백엔드 `POST /api/ai/generate-ac` 호출 — 일반 JSON 응답
3. 로딩 중: skeleton 카드 3~4개 펄스 애니메이션
4. 미리보기 리스트 등장 — 각 AC 카드에 scenario 이름 + Given/When/Then steps 표시
5. 각 카드마다 체크박스 (기본 전체 선택), 개별 선택/해제 가능
6. 하단에 "선택항목 적용" / "다시 생성" / "취소" 버튼
7. "선택항목 적용" → 체크된 AC들만 `acceptanceClient.createAC()`로 저장

### 프롬프트

```
System: You are a QA engineer. Generate BDD acceptance criteria
for the following proposal. Each criterion must have:
- scenario: descriptive name
- steps: array of {keyword: "Given"|"When"|"Then"|"And"|"But", text: "..."}

Return as JSON array. Generate 3-8 scenarios covering:
- Happy path
- Edge cases
- Error cases

Write in the same language as the proposal.

{include_project_context가 true면}
Additional context:
- Existing AC: {기존 AC 목록}
- Design document: {design 문서}
- Spec document: {spec 문서}
{/if}

User: {proposal JSON - problem, scope, outOfScope, approach}
```

### 응답 포맷

- OpenAI: `response_format: { type: "json_object" }` + JSON schema로 구조 강제
- Anthropic: 프롬프트에 JSON 포맷 지시 + 응답 파싱
- 파싱 실패 시: 1회 재시도 (re-prompt). 2회 실패 시 사용자에게 에러 반환

응답 예시:

```json
[
  {
    "scenario": "정기구독 결제 성공",
    "steps": [
      {"keyword": "Given", "text": "사용자가 구독 플랜을 선택한 상태에서"},
      {"keyword": "When", "text": "결제 정보를 입력하고 구독 버튼을 클릭하면"},
      {"keyword": "Then", "text": "정기구독이 활성화되고 확인 이메일이 발송된다"}
    ]
  }
]
```

---

## 4. 백엔드 아키텍처

### 기존 LLM 코드와의 관계

현재 `internal/chat/claude.go`에 hand-rolled Anthropic HTTP 클라이언트가 있고, `internal/specgen/`에서 이를 사용한다. eino 도입 시 두 LLM 추상화가 공존하게 되는데:

- **1단계 (이 스펙)**: `ai/` 패키지에서 eino를 사용. 기존 `chat/`는 그대로 유지
- **2단계 (후속 작업)**: 기존 `chat.ClaudeClient`와 `specgen`을 eino 기반으로 마이그레이션하여 단일 LLM 레이어로 통합

이 스펙에서는 1단계만 구현하고, 2단계는 별도 change로 추적한다.

### 패키지 구조

```
internal/
├── ai/
│   ├── service.go      # 구체 타입 Service struct + GenerateProposal, GenerateAC 메서드
│   ├── providers.go    # OpenAI/Anthropic ChatModel 초기화
│   └── prompts.go      # 프롬프트 템플릿
├── aiconfig/
│   ├── service.go      # ai_configs 테이블 CRUD
│   └── connect_handler.go  # gRPC 핸들러
```

### AI Service (구체 타입)

```go
// internal/ai/service.go
type Service struct {
    configSvc  *aiconfig.Service
    docSvc     *document.Service
    projectSvc *project.Service
    changeSvc  *change.Service  // changeId → projectId 조회용
}

func (s *Service) GenerateProposal(ctx context.Context, input GenerateProposalInput) (*streams.Reader[string], error)
func (s *Service) GenerateAC(ctx context.Context, input GenerateACInput) ([]GeneratedAC, error)
```

Streaming 반환 타입은 eino의 `*streams.Reader[string]`을 사용. eino가 제거/변경될 경우를 대비하여 소비자 쪽 interface는 eino 타입에 직접 의존하지 않고, `io.Reader` 또는 `<-chan string` 등 표준 타입으로 래핑할 수 있도록 설계.

### Interface는 소비자 쪽에서 정의

```go
// internal/api/ai_handler.go (SSE 핸들러)
type proposalGenerator interface {
    GenerateProposal(ctx context.Context, input ai.GenerateProposalInput) (*streams.Reader[string], error)
}

type acGenerator interface {
    GenerateAC(ctx context.Context, input ai.GenerateACInput) ([]ai.GeneratedAC, error)
}
```

테스트 시 mockgen으로 소비자 쪽 interface 기반 mock 생성.

### API Endpoints

ConnectRPC와 별도로 `net/http` 핸들러:

- `POST /api/ai/generate-proposal` — SSE stream 응답
- `POST /api/ai/generate-ac` — JSON 응답
- 인증: 기존 JWT 미들웨어 재사용

### 인가 (Authorization)

모든 AI 엔드포인트에서 다음 체인을 검증:

1. JWT에서 `orgID` 추출 (`auth.ResolveFromHeader()`)
2. `changeId` → `changes` 테이블에서 `projectId` 조회
3. `projectId` → `projects` 테이블에서 `orgId` 검증 (JWT의 orgID와 일치 확인)
4. `projectId` → `ai_configs` 테이블에서 설정 조회
5. `ai_configs` 미존재 시: `CodeFailedPrecondition` 에러 + 메시지 "AI not configured for this project"

기존 `acceptance.ConnectHandler`의 인가 패턴을 따름.

### Rate Limiting

- 조직(org)별 in-memory rate limiter: 분당 최대 10회 요청
- LLM 요청 시 `max_tokens` 설정: Proposal 4096, AC 4096
- 프론트엔드: "다시 생성" 버튼에 3초 cooldown (연속 클릭 방지)

### eino Provider 초기화

```go
func NewChatModel(cfg *aiconfig.AIConfig) (model.ChatModel, error) {
    switch cfg.Provider {
    case "openai":
        return openai.NewChatModel(ctx, &openai.ChatModelConfig{
            Model: cfg.Model, APIKey: cfg.DecryptedAPIKey(),
        })
    case "anthropic":
        return anthropic.NewChatModel(ctx, &anthropic.ChatModelConfig{
            Model: cfg.Model, APIKey: cfg.DecryptedAPIKey(),
        })
    }
}
```

---

## 5. 프론트엔드 구조

### 새로운 컴포넌트

```
web/src/
├── components/
│   ├── ai/
│   │   ├── ai-proposal-generator.tsx   # Proposal 생성 트리거 + 미리보기
│   │   ├── ai-ac-generator.tsx         # AC 생성 트리거 + 미리보기
│   │   └── ai-streaming-preview.tsx    # SSE streaming 텍스트 렌더링
│   └── settings/
│       └── ai-config.tsx               # AI 설정 UI
├── lib/
│   └── ai.ts                           # SSE fetch 유틸 + AI API 클라이언트
```

### 기존 컴포넌트 수정

**`structured-proposal.tsx`**:
- 기존 "AI-assisted spec generation -- coming soon" placeholder를 `<AIProposalGenerator />`로 교체
- 빈 상태 → `<AIProposalGenerator />` 렌더링
- 내용 있으면 → 상단에 "다시 생성" 버튼
- 미리보기 수락 시 기존 state에 값 세팅 → debounce save

**`acceptance-criteria.tsx`**:
- AC 비어있고 proposal 존재 → `<AIACGenerator />` 렌더링
- 내용 있으면 → 상단에 "AI로 추가 생성" 버튼
- 미리보기에서 선택된 AC들 → 기존 `createAC` API로 저장

### 미리보기 상태 머신

```
idle → loading → previewing → idle
         ↑            ↓
         └── regenerate()
```

- `idle`: CTA 또는 "다시 생성" 버튼 표시
- `loading`: Proposal은 streaming, AC는 skeleton
- `previewing`: 결과 + "적용/다시 생성/취소". "적용" 또는 "취소" → idle, "다시 생성" → loading

### SSE 클라이언트

```typescript
// lib/ai.ts
async function* streamProposal(input: {
  changeId: number
  description: string
}): AsyncGenerator<{ section: string; chunk: string }> {
  const res = await fetch('/api/ai/generate-proposal', {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  })
  const reader = res.body!.getReader()
  const decoder = new TextDecoder()
  // SSE event 파싱, section별 chunk yield
}
```

---

## 6. UX 가이드라인

- **Loading**: Proposal은 streaming 타이핑 효과, AC는 skeleton 카드 펄스 애니메이션
- **Error feedback**: AI 설정 미완료 시 "먼저 AI 설정을 완료해주세요" 안내 + 설정 페이지 링크. `CodeFailedPrecondition` 에러 매칭
- **Regenerate**: "다시 생성" 시 이전 결과를 유지하면서 새 결과로 교체. 버튼에 3초 cooldown
- **Connection drop**: SSE 연결 끊김 시 "연결이 끊겼습니다. 다시 시도하시겠습니까?" 안내
- **접근성**: 모든 버튼에 aria-label, loading 상태에 aria-live="polite"
- **i18n**: 모든 UI 문자열은 i18n 키로 처리
- **아이콘**: lucide-react 사용 (Sparkles 아이콘 등)
