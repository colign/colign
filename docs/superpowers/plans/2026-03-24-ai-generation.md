# AI-Powered Proposal & AC Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** SaaS 웹 사용자가 AI를 통해 Proposal 초안(streaming)과 AC(일괄)를 생성할 수 있도록 한다.

**Architecture:** Go 백엔드에서 eino 프레임워크로 OpenAI/Anthropic LLM API를 호출. AI 설정은 프로젝트 레벨로 관리하고, API key는 BYOK 방식으로 AES-256-GCM 암호화 저장. Proposal 생성은 SSE streaming, AC 생성은 JSON 일괄 응답.

**Tech Stack:** Go (eino, net/http SSE), Next.js (React, fetch ReadableStream), ConnectRPC (aiconfig gRPC), PostgreSQL (ai_configs 테이블)

**Spec:** `docs/superpowers/specs/2026-03-24-ai-generation-design.md`

**Branch:** main (직접 작업)

**Spec 대비 변경점:**
- `GenerateProposal` 반환 타입: spec의 `*streams.Reader[string]` → `<-chan SectionChunk` (eino 타입 누출 방지, spec 권고 사항 반영)
- 마이그레이션 번호: spec의 `000005` → `000019` (실제 최신이 `000018`)
- AI Service의 의존성: spec의 `docSvc/projectSvc/changeSvc` → `*bun.DB` 직접 사용 (change 패키지 미존재, 기존 패턴과 일관성)

---

### Task 1: DB 마이그레이션 — ai_configs 테이블

**Files:**
- Create: `migrations/000019_create_ai_configs.up.sql`
- Create: `migrations/000019_create_ai_configs.down.sql`

- [ ] **Step 1: up 마이그레이션 작성**

```sql
-- migrations/000019_create_ai_configs.up.sql
CREATE TABLE ai_configs (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    project_id BIGINT NOT NULL UNIQUE REFERENCES projects(id) ON DELETE CASCADE,
    provider TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    api_key_encrypted BYTEA NOT NULL DEFAULT '',
    key_version SMALLINT NOT NULL DEFAULT 1,
    include_project_context BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

- [ ] **Step 2: down 마이그레이션 작성**

```sql
-- migrations/000019_create_ai_configs.down.sql
DROP TABLE IF EXISTS ai_configs;
```

- [ ] **Step 3: 마이그레이션 적용**

Run: `migrate -path migrations -database "$DATABASE_URL" up`

- [ ] **Step 4: Commit**

```bash
git add migrations/000019_create_ai_configs.up.sql migrations/000019_create_ai_configs.down.sql
git commit -m "feat(db): add ai_configs table for project-level AI settings"
```

---

### Task 2: Config struct에 AIEncryptionKey 추가

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Config struct에 필드 추가**

`internal/config/config.go`의 `Config` struct에 `AIEncryptionKey string` 필드 추가. `Load()` 함수에서 `getEnv("AI_ENCRYPTION_KEY", "")` 로 로드.

- [ ] **Step 2: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add AIEncryptionKey for AI API key encryption"
```

---

### Task 3: aiconfig 패키지 — 모델, 암호화, CRUD 서비스

**Files:**
- Create: `internal/aiconfig/model.go`
- Create: `internal/aiconfig/crypto.go`
- Create: `internal/aiconfig/crypto_test.go`
- Create: `internal/aiconfig/service.go`
- Create: `internal/aiconfig/service_test.go`

- [ ] **Step 1: crypto 테스트 작성**

`internal/aiconfig/crypto_test.go`:
- `TestEncryptDecrypt` — 평문 → 암호화 → 복호화 → 원본 일치 확인
- `TestEncryptDifferentNonces` — 같은 평문이라도 다른 ciphertext 생성
- `TestDecryptWrongKey` — 잘못된 키로 복호화 시 에러
- `TestDecryptCorrupted` — 손상된 ciphertext로 복호화 시 에러

Run: `go test ./internal/aiconfig/ -run TestEncrypt -v`
Expected: FAIL (crypto.go 미존재)

- [ ] **Step 2: crypto 구현**

`internal/aiconfig/crypto.go`:
- `Encrypt(plaintext string, key []byte, keyVersion byte) ([]byte, error)` — AES-256-GCM. 반환: `[keyVersion(1) || nonce(12) || ciphertext]`
- `Decrypt(data []byte, key []byte) (string, error)` — data[0]은 keyVersion, data[1:13]은 nonce, data[13:]은 ciphertext
- `MaskAPIKey(key string) string` — `sk-...xxxx` 형태로 마스킹. 4자 이하면 `****`

Run: `go test ./internal/aiconfig/ -run TestEncrypt -v`
Expected: PASS

- [ ] **Step 3: model 작성**

`internal/aiconfig/model.go`:
```go
package aiconfig

import "time"

type AIConfig struct {
    bun.BaseModel         `bun:"table:ai_configs,alias:aic"`
    ID                    int64     `bun:"id,pk,autoincrement"`
    ProjectID             int64     `bun:"project_id,notnull,unique"`
    Provider              string    `bun:"provider,notnull"`
    Model                 string    `bun:"model,notnull"`
    APIKeyEncrypted       []byte    `bun:"api_key_encrypted,notnull"`
    KeyVersion            int16     `bun:"key_version,notnull,default:1"`
    IncludeProjectContext bool      `bun:"include_project_context,notnull"`
    CreatedAt             time.Time `bun:"created_at,notnull,default:current_timestamp"`
    UpdatedAt             time.Time `bun:"updated_at,notnull,default:current_timestamp"`
}
```

- [ ] **Step 4: service 테스트 작성**

`internal/aiconfig/service_test.go`:
- `TestUpsert_Create` — 새 config 생성 확인
- `TestUpsert_Update` — 기존 config 업데이트 확인 (빈 apiKey면 기존 유지)
- `TestGetByProjectID` — 조회 성공 + 미존재 시 nil 반환
- `TestDelete` — 삭제 확인

sqlmock 사용. 기존 서비스 테스트 패턴(`internal/acceptance/service_test.go`) 참고.

Run: `go test ./internal/aiconfig/ -run TestUpsert -v`
Expected: FAIL

- [ ] **Step 5: service 구현**

`internal/aiconfig/service.go`:
- `NewService(db *bun.DB, encryptionKey []byte) *Service`
- `Upsert(ctx, projectID int64, input UpsertInput) (*AIConfig, error)` — INSERT ON CONFLICT UPDATE. apiKey가 빈 문자열이면 기존 암호화 값 유지
- `GetByProjectID(ctx, projectID int64) (*AIConfig, error)` — 미존재 시 nil, nil 반환
- `Delete(ctx, projectID int64) error`
- `DecryptAPIKey(cfg *AIConfig) (string, error)` — 복호화 헬퍼

`UpsertInput` struct: `Provider, Model, APIKey string, IncludeProjectContext bool`

Run: `go test ./internal/aiconfig/ -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/aiconfig/
git commit -m "feat(aiconfig): add model, crypto, and CRUD service for AI configuration"
```

---

### Task 4: aiconfig proto + ConnectRPC 핸들러

> **Note:** `TestConnection`은 eino provider가 필요하므로 Task 5 이후에 구현. 이 Task에서는 CRUD RPC만 구현하고, `TestConnection`은 Task 5에서 `ai.NewChatModel`이 생기면 추가.

**Files:**
- Create: `proto/aiconfig/v1/aiconfig.proto`
- Modify: `proto/buf.gen.yaml` (필요 시)
- Create: `internal/aiconfig/connect_handler.go`
- Create: `internal/aiconfig/connect_handler_test.go`

- [ ] **Step 1: proto 정의**

`proto/aiconfig/v1/aiconfig.proto`:
```protobuf
syntax = "proto3";
package aiconfig.v1;
option go_package = "github.com/gobenpark/colign/gen/proto/aiconfig/v1;aiconfigv1";

import "google/protobuf/timestamp.proto";

message AIConfig {
  int64 id = 1;
  int64 project_id = 2;
  string provider = 3;
  string model = 4;
  string api_key_masked = 5;  // 마스킹된 키만 반환
  bool include_project_context = 6;
  google.protobuf.Timestamp created_at = 7;
  google.protobuf.Timestamp updated_at = 8;
}

message GetAIConfigRequest { int64 project_id = 1; }
message GetAIConfigResponse { optional AIConfig config = 1; }

message SaveAIConfigRequest {
  int64 project_id = 1;
  string provider = 2;
  string model = 3;
  string api_key = 4;  // 빈 문자열이면 기존 유지
  bool include_project_context = 5;
}
message SaveAIConfigResponse { AIConfig config = 1; }

message TestConnectionRequest {
  string provider = 1;
  string model = 2;
  string api_key = 3;
}
message TestConnectionResponse { bool success = 1; string error = 2; }

message DeleteAIConfigRequest { int64 project_id = 1; }
message DeleteAIConfigResponse {}

service AIConfigService {
  rpc GetAIConfig(GetAIConfigRequest) returns (GetAIConfigResponse) {}
  rpc SaveAIConfig(SaveAIConfigRequest) returns (SaveAIConfigResponse) {}
  rpc TestConnection(TestConnectionRequest) returns (TestConnectionResponse) {}
  rpc DeleteAIConfig(DeleteAIConfigRequest) returns (DeleteAIConfigResponse) {}
}
```

- [ ] **Step 2: proto 생성**

Run: `cd proto && buf generate`
확인: `gen/proto/aiconfig/v1/` 및 `web/src/gen/proto/aiconfig/v1/` 생성 확인

- [ ] **Step 3: connect_handler 테스트 작성**

`internal/aiconfig/connect_handler_test.go`:
- `TestGetAIConfig_Success` — 정상 조회
- `TestGetAIConfig_NotFound` — 미존재 시 nil config 반환
- `TestSaveAIConfig_Create` — 새 생성
- `TestSaveAIConfig_UpdateKeepKey` — apiKey 빈 문자열 시 기존 유지
- `TestSaveAIConfig_Unauthenticated` — 인증 없이 호출 시 에러

mockgen으로 mock 생성. 기존 `internal/acceptance/connect_handler.go` 패턴 참조.

Run: `go test ./internal/aiconfig/ -run TestGetAIConfig -v`
Expected: FAIL

- [ ] **Step 4: connect_handler 구현**

`internal/aiconfig/connect_handler.go`:
- `ConnectHandler` struct: `service *Service`, `jwtManager *auth.JWTManager`, `apiTokenValidator auth.APITokenValidator`
- `NewConnectHandler(...)` 생성자
- 각 RPC 메서드에서 `auth.ResolveFromHeader()`로 인증, `claims.OrgID`로 project 소유권 검증
- `GetAIConfig`: 조회 → 마스킹된 key 반환
- `SaveAIConfig`: upsert → 마스킹된 key 반환
- `TestConnection`: 이 Task에서는 stub으로 구현 (`return connect.NewError(connect.CodeUnimplemented, ...)`). Task 5 완료 후 실제 구현 추가
- `DeleteAIConfig`: 삭제

Run: `go test ./internal/aiconfig/ -v`
Expected: PASS

- [ ] **Step 5: server.go에 라우트 등록**

`internal/server/server.go`에 aiconfig 서비스 등록:
```go
aiConfigService := aiconfig.NewService(s.db, []byte(cfg.AIEncryptionKey))
aiConfigConnectHandler := aiconfig.NewConnectHandler(aiConfigService, s.jwtManager, apiTokenService)
aiConfigPath, aiConfigHandler := aiconfigv1connect.NewAIConfigServiceHandler(aiConfigConnectHandler)
s.mux.Handle(aiConfigPath, aiConfigHandler)
```

- [ ] **Step 6: Commit**

```bash
git add proto/aiconfig/ gen/proto/aiconfig/ web/src/gen/proto/aiconfig/ internal/aiconfig/connect_handler.go internal/aiconfig/connect_handler_test.go internal/server/server.go
git commit -m "feat(aiconfig): add proto definition, ConnectRPC handler, and route registration"
```

---

### Task 5: eino 의존성 추가 + ai 패키지 — providers

**Files:**
- Modify: `go.mod`, `go.sum`
- Create: `internal/ai/providers.go`
- Create: `internal/ai/providers_test.go`

- [ ] **Step 1: eino 의존성 추가**

Run: `go get github.com/cloudwego/eino github.com/cloudwego/eino-ext/components/model/openai github.com/cloudwego/eino-ext/components/model/claude`

eino 패키지 이름이 다를 수 있으므로 공식 GitHub 확인 후 정확한 import path 사용.

- [ ] **Step 2: providers 테스트 작성**

`internal/ai/providers_test.go`:
- `TestNewChatModel_OpenAI` — provider "openai"일 때 에러 없이 모델 생성
- `TestNewChatModel_Anthropic` — provider "anthropic"일 때 에러 없이 모델 생성
- `TestNewChatModel_Unknown` — 알 수 없는 provider 시 에러 반환

Run: `go test ./internal/ai/ -run TestNewChatModel -v`
Expected: FAIL

- [ ] **Step 3: providers 구현**

`internal/ai/providers.go`:
```go
package ai

import (
    "context"
    "fmt"
    // eino imports
)

func NewChatModel(ctx context.Context, provider, model, apiKey string) (model.ChatModel, error) {
    switch provider {
    case "openai":
        return openai.NewChatModel(ctx, &openai.ChatModelConfig{
            Model: model, APIKey: apiKey,
        })
    case "anthropic":
        return claude.NewChatModel(ctx, &claude.ChatModelConfig{
            Model: model, APIKey: apiKey,
        })
    default:
        return nil, fmt.Errorf("unsupported provider: %s", provider)
    }
}
```

Run: `go test ./internal/ai/ -run TestNewChatModel -v`
Expected: PASS

- [ ] **Step 4: TestConnection 구현 연결**

Task 4에서 stub으로 남겨둔 `aiconfig.ConnectHandler.TestConnection`을 실제 구현으로 교체.
`ConnectHandler`에 `testConnFunc func(ctx, provider, model, apiKey string) error` 콜백 필드 추가.
`NewConnectHandler`에서 `ai.TestConnection` 함수를 전달받아 주입.

```go
func (h *ConnectHandler) TestConnection(ctx context.Context, req *connect.Request[...]) (*connect.Response[...], error) {
    // auth check
    err := h.testConnFunc(ctx, req.Msg.Provider, req.Msg.Model, req.Msg.ApiKey)
    if err != nil {
        return connect.NewResponse(&TestConnectionResponse{Success: false, Error: err.Error()}), nil
    }
    return connect.NewResponse(&TestConnectionResponse{Success: true}), nil
}
```

server.go에서 wiring:
```go
aiConfigConnectHandler := aiconfig.NewConnectHandler(aiConfigService, s.jwtManager, apiTokenService, ai.TestConnection)
```

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/ai/ internal/aiconfig/connect_handler.go internal/server/server.go
git commit -m "feat(ai): add eino dependency, multi-provider ChatModel factory, and TestConnection"
```

---

### Task 6: ai 패키지 — prompts + section parser

**Files:**
- Create: `internal/ai/prompts.go`
- Create: `internal/ai/parser.go`
- Create: `internal/ai/parser_test.go`

- [ ] **Step 1: parser 테스트 작성**

`internal/ai/parser_test.go`:
- `TestParseSections_AllSections` — `---SECTION:problem---\ncontent\n---SECTION:scope---\ncontent2` → map 반환
- `TestParseSections_Partial` — 일부 섹션만 있을 때
- `TestParseSections_Empty` — 빈 입력
- `TestParseSections_StreamChunks` — chunk 단위로 들어오는 텍스트에서 현재 section 추적

Run: `go test ./internal/ai/ -run TestParseSections -v`
Expected: FAIL

- [ ] **Step 2: parser 구현**

`internal/ai/parser.go`:
- `SectionParser` struct — streaming chunk를 받아 현재 section과 텍스트를 추적
- `Feed(chunk string) []SectionChunk` — chunk를 파싱하여 `{Section, Text}` 슬라이스 반환
- delimiter `---SECTION:xxx---` 감지 시 section 전환

Run: `go test ./internal/ai/ -run TestParseSections -v`
Expected: PASS

- [ ] **Step 3: prompts 테스트 작성**

`internal/ai/prompts_test.go`:
- `TestProposalSystemPrompt_WithoutContext` — context 미포함 시 기본 프롬프트 확인
- `TestProposalSystemPrompt_WithContext` — README + changes 포함 확인
- `TestProposalSystemPrompt_TruncatesReadme` — 2000자 초과 README truncation 확인
- `TestProposalSystemPrompt_MaxChanges` — 10개 초과 changes 잘림 확인
- `TestACSystemPrompt_WithoutContext` — 기본 AC 프롬프트 확인
- `TestACSystemPrompt_WithContext` — 기존 AC + design/spec 문서 포함 확인

Run: `go test ./internal/ai/ -run TestProposal -v`
Expected: FAIL

- [ ] **Step 4: prompts 구현**

`internal/ai/prompts.go`:
- `ProposalSystemPrompt(includeContext bool, readme string, recentChanges []string) string`
- `ACSystemPrompt(includeContext bool, existingAC []string, designDoc, specDoc string) string`
- 스펙에 정의된 프롬프트 템플릿 그대로 구현
- context 포함 시 README는 2000자로 truncate, changes는 최대 10개

Run: `go test ./internal/ai/ -run TestProposal -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/ai/prompts.go internal/ai/parser.go internal/ai/parser_test.go
git commit -m "feat(ai): add prompt templates and streaming section parser"
```

---

### Task 7: ai 패키지 — Service (GenerateProposal, GenerateAC)

**Files:**
- Create: `internal/ai/service.go`
- Create: `internal/ai/service_test.go`

- [ ] **Step 1: service 테스트 작성**

`internal/ai/service_test.go`:
- `TestGenerateProposal_Success` — mock ChatModel로 streaming 응답 생성, SectionParser로 파싱 확인
- `TestGenerateProposal_NoConfig` — ai_config 미설정 시 에러
- `TestGenerateAC_Success` — mock ChatModel로 JSON 배열 응답, []GeneratedAC로 파싱 확인
- `TestGenerateAC_ParseFailRetry` — 첫 번째 파싱 실패 → 재시도 → 성공
- `TestGenerateAC_ParseFailTwice` — 2회 실패 시 에러 반환

Run: `go test ./internal/ai/ -run TestGenerate -v`
Expected: FAIL

- [ ] **Step 2: service 구현**

`internal/ai/service.go`:
```go
type Service struct {
    configSvc  *aiconfig.Service
    db         *bun.DB  // change → project 조회용
}

type GenerateProposalInput struct {
    ChangeID    int64
    OrgID       int64
    Description string
}

type GenerateACInput struct {
    ChangeID int64
    OrgID    int64
    Proposal string // JSON
}

type GeneratedAC struct {
    Scenario string
    Steps    []ACStep
}

type ACStep struct {
    Keyword string
    Text    string
}
```

- `GenerateProposal`: ai_config(이미 조회됨) → 복호화 → ChatModel 생성 → Stream() 호출 → SectionParser로 변환하여 `<-chan SectionChunk` 반환. `max_tokens: 4096` 설정
- `GenerateAC`: ai_config(이미 조회됨) → ChatModel.Generate() → JSON 파싱 → 실패 시 1회 재시도. `max_tokens: 4096` 설정
- **인가는 handler에서 수행** (기존 코드베이스 패턴 준수). Service는 이미 검증된 `*aiconfig.AIConfig`를 받음

Run: `go test ./internal/ai/ -run TestGenerate -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/ai/service.go internal/ai/service_test.go
git commit -m "feat(ai): add Service with GenerateProposal (streaming) and GenerateAC (batch)"
```

---

### Task 8: HTTP 핸들러 — SSE + JSON 엔드포인트

**Files:**
- Create: `internal/ai/handler.go`
- Create: `internal/ai/handler_test.go`
- Create: `internal/ai/ratelimit.go`
- Create: `internal/ai/ratelimit_test.go`
- Modify: `internal/server/server.go`

- [ ] **Step 1: rate limiter 테스트 작성**

`internal/ai/ratelimit_test.go`:
- `TestRateLimiter_Allow` — 10회까지 허용
- `TestRateLimiter_Deny` — 11번째 거부
- `TestRateLimiter_Reset` — 1분 후 리셋

Run: `go test ./internal/ai/ -run TestRateLimiter -v`
Expected: FAIL

- [ ] **Step 2: rate limiter 구현**

`internal/ai/ratelimit.go`:
- `OrgRateLimiter` struct — `sync.Map`으로 org별 토큰 버킷 관리
- `Allow(orgID int64) bool` — 분당 10회 제한
- 만료된 엔트리 자동 정리

Run: `go test ./internal/ai/ -run TestRateLimiter -v`
Expected: PASS

- [ ] **Step 3: handler 테스트 작성**

`internal/ai/handler_test.go`:
- `TestHandleGenerateProposal_SSE` — SSE 응답 형식 검증 (Content-Type, X-Accel-Buffering, event format)
- `TestHandleGenerateProposal_Unauthorized` — 인증 없이 401
- `TestHandleGenerateProposal_NoAIConfig` — AI 설정 미존재 시 412 (FailedPrecondition)
- `TestHandleGenerateAC_JSON` — JSON 응답 형식 검증
- `TestHandleGenerateAC_RateLimited` — rate limit 초과 시 429

소비자 쪽 interface 정의 + `//go:generate mockgen` 디렉티브:
```go
//go:generate mockgen -source=handler.go -destination=mock_handler_test.go -package=ai

type proposalGenerator interface {
    GenerateProposal(ctx context.Context, cfg *aiconfig.AIConfig, input GenerateProposalInput) (<-chan SectionChunk, error)
}
type acGenerator interface {
    GenerateAC(ctx context.Context, cfg *aiconfig.AIConfig, input GenerateACInput) ([]GeneratedAC, error)
}
```

Run: `go test ./internal/ai/ -run TestHandle -v`
Expected: FAIL

- [ ] **Step 4: handler 구현**

`internal/ai/handler.go`:
- `Handler` struct: `proposalGen proposalGenerator`, `acGen acGenerator`, `jwtManager`, `configSvc *aiconfig.Service`, `db *bun.DB`, `limiter *OrgRateLimiter`
- `NewHandler(...)` 생성자
- `resolveAIConfig(ctx, r) (*aiconfig.AIConfig, *auth.Claims, error)` — 공통 인가 헬퍼:
  1. `auth.ResolveFromHeader()` → JWT claims 추출
  2. rate limit 체크 (claims.OrgID)
  3. request body에서 `changeId` 추출
  4. `changeId` → DB에서 `projectId` 조회 (JOIN changes c, projects p WHERE p.organization_id = claims.OrgID)
  5. `configSvc.GetByProjectID(projectId)` → 미존재 시 412 에러
  6. `*AIConfig` + `*Claims` 반환
- `HandleGenerateProposal(w, r)`:
  1. `resolveAIConfig()` 호출
  2. SSE headers 설정 (`Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`, `X-Accel-Buffering: no`)
  3. `GenerateProposal(ctx, cfg, input)` 호출 → channel에서 읽어 SSE event로 write
  4. flush 후 `data: [DONE]` 전송
- `HandleGenerateAC(w, r)`:
  1. `resolveAIConfig()` 호출
  2. DB에서 proposal document 조회 (`documents` 테이블, type="proposal", change_id 기준)
  3. `GenerateAC(ctx, cfg, input)` 호출 → JSON 응답

Run: `go test ./internal/ai/ -run TestHandle -v`
Expected: PASS

- [ ] **Step 5: server.go에 라우트 등록**

`internal/server/server.go`에 추가:
```go
aiService := ai.NewService(aiConfigService, s.db)
aiHandler := ai.NewHandler(aiService, aiService, s.jwtManager)
s.mux.HandleFunc("POST /api/ai/generate-proposal", aiHandler.HandleGenerateProposal)
s.mux.HandleFunc("POST /api/ai/generate-ac", aiHandler.HandleGenerateAC)
```

- [ ] **Step 6: Commit**

```bash
git add internal/ai/handler.go internal/ai/handler_test.go internal/ai/ratelimit.go internal/ai/ratelimit_test.go internal/server/server.go
git commit -m "feat(ai): add SSE and JSON HTTP handlers with org-level rate limiting"
```

---

### Task 9: 프론트엔드 — i18n 키 추가

**Files:**
- Modify: `web/src/lib/i18n/locales/en.json`
- Modify: `web/src/lib/i18n/locales/ko.json`

- [ ] **Step 1: 영문/한국어 번역 키 추가**

`ai` 섹션 추가:
```json
"ai": {
  "generateProposal": "Generate with AI",
  "regenerateProposal": "Regenerate",
  "generateAC": "Generate AC from Proposal",
  "addMoreAC": "Generate more with AI",
  "apply": "Apply",
  "cancel": "Cancel",
  "descriptionPlaceholder": "Describe this change in one line...",
  "confirmRegenerate": "This will replace existing content. Continue?",
  "writeProposalFirst": "Write a proposal first to generate acceptance criteria.",
  "loading": "Generating...",
  "connectionError": "Connection lost. Retry?",
  "selectAll": "Select all",
  "deselectAll": "Deselect all",
  "applySelected": "Apply selected",
  "emptyStateTitle": "Start with AI",
  "emptyStateDescription": "Describe what you want to build and AI will draft the proposal for you."
},
"aiConfig": {
  "title": "AI Configuration",
  "provider": "Provider",
  "model": "Model",
  "apiKey": "API Key",
  "apiKeyPlaceholder": "Enter your API key",
  "testConnection": "Test Connection",
  "testSuccess": "Connection successful",
  "testFailed": "Connection failed",
  "includeContext": "Include project context",
  "includeContextHelp": "Include project README and recent changes in AI prompts",
  "save": "Save",
  "notConfigured": "Configure AI settings to use AI features"
}
```

한국어(`ko.json`)도 동일 구조로 번역 추가.

- [ ] **Step 2: Commit**

```bash
git add web/src/lib/i18n/locales/
git commit -m "feat(i18n): add AI generation and AI config translation keys"
```

---

### Task 10: 프론트엔드 — AI Config 설정 UI

**Files:**
- Create: `web/src/lib/aiconfig.ts`
- Create: `web/src/components/settings/ai-config.tsx`
- Modify: `web/src/app/projects/[slug]/settings/page.tsx`

- [ ] **Step 1: aiconfig 클라이언트 생성**

`web/src/lib/aiconfig.ts`:
```typescript
import { createClient } from "@connectrpc/connect";
import { AIConfigService } from "@/gen/proto/aiconfig/v1/aiconfig_pb";
import { transport } from "./connect";

export const aiConfigClient = createClient(AIConfigService, transport);
```

- [ ] **Step 2: AI Config 컴포넌트 구현**

`web/src/components/settings/ai-config.tsx`:
- Provider dropdown: OpenAI / Anthropic
- Model dropdown: provider 변경 시 동적 옵션 (하드코딩된 모델 리스트)
- API Key: password input + "연결 테스트" 버튼
- Include context: checkbox/toggle
- Save 버튼
- 상태: loading, saving, testing

기존 `settings/page.tsx`의 tab 패턴 참조. lucide-react 아이콘 사용. 모든 문자열 i18n 키.

- [ ] **Step 3: settings page에 AI 탭 추가**

`web/src/app/projects/[slug]/settings/page.tsx`:
- `SettingsTab` type에 `"ai"` 추가: `type SettingsTab = "general" | "members" | "approval" | "archive" | "ai" | "danger";`
- tabs 배열에 `"ai"` 추가
- `{activeTab === "ai" && <AIConfigSettings projectId={project.id} />}` 렌더링

- [ ] **Step 4: Commit**

```bash
git add web/src/lib/aiconfig.ts web/src/components/settings/ai-config.tsx web/src/app/projects/[slug]/settings/page.tsx
git commit -m "feat(web): add AI configuration tab in project settings"
```

---

### Task 11: 프론트엔드 — SSE 클라이언트 유틸

**Files:**
- Create: `web/src/lib/ai.ts`

- [ ] **Step 1: SSE 클라이언트 구현**

`web/src/lib/ai.ts`:

> **Note:** `getAccessToken`은 `./auth`에서 import (connect.ts는 re-export하지 않음). `credentials: "include"`로 cookie auth도 지원. `changeId`는 proto에서 `bigint`로 올 수 있으므로 `Number()`로 변환.

```typescript
import { getAccessToken, refreshAccessToken } from "./auth";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export interface SectionChunk {
  section: string;
  chunk: string;
}

// 401 시 토큰 리프레시 후 재시도하는 헬퍼
async function fetchWithRefresh(url: string, init: RequestInit): Promise<Response> {
  const res = await fetch(url, init);
  if (res.status === 401) {
    const newToken = await refreshAccessToken();
    if (!newToken) throw new Error("Authentication expired");
    const retryInit = {
      ...init,
      headers: { ...init.headers as Record<string, string>, Authorization: `Bearer ${newToken}` },
    };
    return fetch(url, retryInit);
  }
  return res;
}

export async function* streamProposal(
  changeId: number | bigint,
  description: string,
  signal?: AbortSignal
): AsyncGenerator<SectionChunk> {
  const token = getAccessToken();
  const res = await fetchWithRefresh(`${API_BASE}/api/ai/generate-proposal`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ changeId: Number(changeId), description }),
    credentials: "include",
    signal,
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error(err.message || "Failed to generate proposal");
  }

  const reader = res.body!.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;

    buffer += decoder.decode(value, { stream: true });
    const lines = buffer.split("\n");
    buffer = lines.pop() || "";

    for (const line of lines) {
      if (line.startsWith("data: ")) {
        const data = line.slice(6);
        if (data === "[DONE]") return;
        try {
          yield JSON.parse(data) as SectionChunk;
        } catch {
          // skip malformed lines
        }
      }
    }
  }
}

export interface GeneratedAC {
  scenario: string;
  steps: { keyword: string; text: string }[];
}

export async function generateAC(changeId: number | bigint): Promise<GeneratedAC[]> {
  const token = getAccessToken();
  const res = await fetchWithRefresh(`${API_BASE}/api/ai/generate-ac`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ changeId: Number(changeId) }),
    credentials: "include",
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: res.statusText }));
    throw new Error(err.message || "Failed to generate AC");
  }

  return res.json();
}
```

- [ ] **Step 2: Commit**

```bash
git add web/src/lib/ai.ts
git commit -m "feat(web): add SSE streaming client and AC generation fetch utility"
```

---

### Task 12: 프론트엔드 — AI Proposal Generator 컴포넌트

**Files:**
- Create: `web/src/components/ai/ai-proposal-generator.tsx`
- Create: `web/src/components/ai/ai-streaming-preview.tsx`
- Modify: `web/src/components/change/structured-proposal.tsx`

- [ ] **Step 1: streaming preview 컴포넌트 구현**

`web/src/components/ai/ai-streaming-preview.tsx`:
- Props: `sections: Record<string, string>`, `isStreaming: boolean`
- 4개 섹션(problem, scope, outOfScope, approach)을 카드 형태로 렌더링
- streaming 중이면 마지막 섹션에 커서 깜빡임 효과

- [ ] **Step 2: AI Proposal Generator 컴포넌트 구현**

`web/src/components/ai/ai-proposal-generator.tsx`:
- Props: `changeId: number`, `onApply: (sections: ProposalSections) => void`, `hasExistingContent: boolean`
- 상태 머신: idle → loading → previewing → idle
- **idle (빈 상태)**: 한 줄 입력 + "AI로 초안 작성" 버튼 (Sparkles 아이콘)
- **idle (내용 있을 때)**: 작은 "다시 생성" 버튼, 클릭 시 확인 dialog
- **loading**: `streamProposal()` 호출, `AbortController` 관리, `AIStreamingPreview` 렌더링. 연결 끊김 시 에러 UI + 재시도 버튼
- **previewing**: 완성된 미리보기 + "적용/다시 생성/취소" 버튼 (다시 생성에 3초 cooldown)
- "적용" → `onApply(sections)` 콜백 호출
- **접근성**: 모든 버튼에 `aria-label`, loading 영역에 `aria-live="polite"`, Sparkles 아이콘에 `aria-hidden="true"`

- [ ] **Step 3: structured-proposal.tsx 수정**

기존 "AI-assisted spec generation -- coming soon" placeholder 제거.
`<AIProposalGenerator>` 통합:
- 빈 상태: `<AIProposalGenerator>` 표시
- 내용 있을 때: 폼 상단에 `<AIProposalGenerator hasExistingContent />` 표시
- `onApply`에서 각 섹션 state 업데이트 → 기존 debounce save 로직 작동

- [ ] **Step 4: Commit**

```bash
git add web/src/components/ai/ web/src/components/change/structured-proposal.tsx
git commit -m "feat(web): add AI proposal generator with streaming preview"
```

---

### Task 13: 프론트엔드 — AI AC Generator 컴포넌트

**Files:**
- Create: `web/src/components/ai/ai-ac-generator.tsx`
- Modify: `web/src/components/change/acceptance-criteria.tsx`

- [ ] **Step 1: AI AC Generator 컴포넌트 구현**

`web/src/components/ai/ai-ac-generator.tsx`:
- Props: `changeId: number`, `hasProposal: boolean`, `hasExistingAC: boolean`, `onApply: (acs: GeneratedAC[]) => void`
- 상태: idle → loading → previewing → idle
- **idle (빈 상태 + proposal 있음)**: "Proposal 기반 AC 자동 생성" CTA
- **idle (내용 있을 때)**: "AI로 추가 생성" 버튼
- **idle (proposal 없음)**: 비활성화 + "먼저 Proposal을 작성해주세요"
- **loading**: `generateAC()` 호출, skeleton 카드 3~4개 표시
- **previewing**: AC 카드 리스트, 각 카드에 체크박스 (기본 전체 선택), Given/When/Then steps 표시, 하단에 "선택항목 적용/다시 생성/취소"

- [ ] **Step 2: structured-proposal.tsx에 hasProposal prop 전달 추가**

`AcceptanceCriteria` 렌더링 시 `hasProposal` prop 추가:
```tsx
<AcceptanceCriteria changeId={changeId} reviewMode={isReviewMode} hasProposal={!!sections.problem || !!sections.scope} />
```

- [ ] **Step 3: acceptance-criteria.tsx 수정**

- `hasProposal` prop 추가
- `<AIACGenerator>` 통합: AC 리스트 상단에 조건부 렌더링
- `onApply`에서 선택된 AC들을 `acceptanceClient.createAC()` 순차 호출 → 리스트 리프레시

- [ ] **Step 3: Commit**

```bash
git add web/src/components/ai/ai-ac-generator.tsx web/src/components/change/acceptance-criteria.tsx web/src/components/change/structured-proposal.tsx
git commit -m "feat(web): add AI acceptance criteria generator with batch preview"
```

---

### Task 14: 통합 테스트 및 정리

**Files:**
- 전체 코드베이스

- [ ] **Step 1: Go 테스트 전체 실행**

Run: `go test ./internal/aiconfig/... ./internal/ai/... -v -count=1`
Expected: ALL PASS

- [ ] **Step 2: lint 확인**

Run: `golangci-lint run ./internal/aiconfig/... ./internal/ai/...`
Expected: 에러 없음

- [ ] **Step 3: 프론트엔드 빌드 확인**

Run: `cd web && npm run build`
Expected: 빌드 성공

- [ ] **Step 4: 최종 Commit (필요 시)**

남은 수정사항이 있으면 커밋.

```bash
git commit -m "chore: fix lint and build issues for AI generation feature"
```
