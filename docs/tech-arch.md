# EtiketAI — Arhitectură Tehnică & API Contract

> Go Microservices · Flutter · Claude + Ollama · PostgreSQL · Redis
> Draft v1.0 · Martie 2026 · Confidențial

---

## Cuprins

1. [Microservices Map](#1-microservices-map)
2. [Inter-Service Communication](#2-inter-service-communication)
3. [AI Agent Engine](#3-ai-agent-engine-agent-svc)
4. [Database Schema](#4-database-schema)
5. [API Contract](#5-api-contract-rest--api-gateway-8080)
6. [Monorepo Layout](#6-monorepo-layout)
7. [Infrastructure](#7-infrastructure)

---

## 1. Microservices Map

Sistemul este compus din **7 servicii Go independente + 1 gateway**, fiecare cu responsabilitate unică și bază de date proprie (Database-per-Service pattern).

| Serviciu | Port | Protocol | Responsabilitate | Tech stack |
|---|---|---|---|---|
| api-gateway | 8080 | HTTP/REST | Entry point, auth middleware, rate limit, routing | Go · Chi router · JWT · Redis rate limiter |
| auth-svc | 8081 | gRPC | Înregistrare, login, JWT emit/verify, OAuth2 | Go · PostgreSQL · bcrypt · Redis blacklist |
| workspace-svc | 8082 | gRPC | Tenants, membri, roluri, plan abonament, quota | Go · PostgreSQL · Stripe SDK |
| label-svc | 8083 | gRPC | CRUD etichete, upload imagini, status tracking | Go · PostgreSQL · S3/R2 |
| agent-svc | 8084 | gRPC | AI Agent engine: vision, traducere, validare | Go · Claude API · Ollama client · Redis cache |
| print-svc | 8085 | gRPC + WS | Generare PDF/ZPL, coadă print, gateway WS | Go · go-pdf · Asynq · WebSocket |
| notification-svc | 8086 | gRPC | Push notifications, email transacțional, webhooks | Go · FCM · APNs · Resend |
| admin-svc | 8087 | HTTP/REST | Super-admin: config agenți, metrics, logs | Go · Chi · PostgreSQL |

### 1.1 Shared Infrastructure

| Componentă | Port | Rol |
|---|---|---|
| PostgreSQL (per svc) | 5432–5439 | auth_db, workspace_db, label_db, agent_db, print_db |
| Redis | 6379 | Rate limiting, JWT blacklist, cache agent configs, Asynq queue, pub/sub |
| MinIO / R2 | 9000 | Object storage: imagini originale, PDF generate, logs arhivate |
| Traefik | 80/443 | Reverse proxy, TLS termination |
| Prometheus + Grafana | 9090/3000 | Metrics per serviciu: latență, throughput, erori, cost AI |
| Jaeger | 16686 | Distributed tracing cu OpenTelemetry |

---

## 2. Inter-Service Communication

### 2.1 gRPC — apeluri sincrone

```protobuf
// proto/agent/v1/agent.proto
syntax = "proto3";
package agent.v1;

service AgentService {
  rpc ProcessVision(VisionRequest)      returns (VisionResponse);
  rpc ProcessTranslation(TranslRequest) returns (TranslResponse);
  rpc ProcessValidation(ValidRequest)   returns (ValidResponse);
  rpc GetAgentConfig(ConfigRequest)     returns (AgentConfig);
  rpc UpdateAgentConfig(AgentConfig)    returns (UpdateResponse);
}

message VisionRequest {
  string label_id        = 1;
  string image_s3_key    = 2;
  string workspace_id    = 3;
  string target_language = 4; // default: "ro"
}

message VisionResponse {
  string label_id      = 1;
  string raw_json      = 2; // extracted fields JSON
  float  confidence    = 3;
  string detected_lang = 4;
  string provider_used = 5; // "claude" | "ollama"
  int32  tokens_used   = 6;
}
```

### 2.2 Event-driven — Redis Pub/Sub (Streams)

| Stream / Channel | Publisher | Subscribers |
|---|---|---|
| `label:processing:complete` | agent-svc | label-svc (update status), notification-svc (push) |
| `label:confirmed` | label-svc | print-svc (pregătire job), notification-svc |
| `print:job:status` | print-svc | label-svc (update status), notification-svc |
| `billing:subscription:updated` | workspace-svc | auth-svc (update quota cache) |
| `agent:config:updated` | admin-svc | agent-svc (invalidare config cache Redis) |

---

## 3. AI Agent Engine (agent-svc)

Serviciul `agent-svc` implementează un **provider pattern** curat în Go. Implementările concrete (Claude, Ollama) sunt injectate la runtime din configurația workspace-ului, stocată în Redis cu TTL de 5 minute.

### 3.1 Interfețe Go

```go
// internal/agent/interfaces.go

type VisionAgent interface {
    ExtractFields(ctx context.Context, req VisionRequest) (*VisionResult, error)
    Name() string
    EstimateCost(imageBytes int64) float64
}

type TranslationAgent interface {
    Translate(ctx context.Context, req TranslRequest) (*TranslResult, error)
    Name() string
    SupportedLanguages() []string
}

type ValidationAgent interface {
    Validate(ctx context.Context, req ValidRequest) (*ComplianceResult, error)
    Name() string
    RulesVersion() string
}

// VisionResult — output structurat din orice provider
type VisionResult struct {
    ProductName     string             `json:"product_name"`
    Ingredients     string             `json:"ingredients"`
    Manufacturer    string             `json:"manufacturer"`
    Address         string             `json:"address"`
    Quantity        string             `json:"quantity"`
    ExpiryDate      string             `json:"expiry_date"`
    Warnings        string             `json:"warnings"`
    CountryOfOrigin string             `json:"country_of_origin"`
    Category        ProductCategory    `json:"category"`
    DetectedLang    string             `json:"detected_lang"`
    Confidence      map[string]float32 `json:"confidence"` // per field
    ProviderUsed    string             `json:"provider_used"`
    TokensUsed      int                `json:"tokens_used"`
}
```

### 3.2 Implementare Claude Vision Agent

```go
// internal/agent/providers/claude/vision.go

type ClaudeVisionAgent struct {
    client    *anthropic.Client
    model     string // "claude-sonnet-4-6" default
    maxTokens int
}

func (a *ClaudeVisionAgent) ExtractFields(ctx context.Context, req VisionRequest) (*VisionResult, error) {
    imageB64, err := loadImageFromS3(ctx, req.ImageS3Key)
    if err != nil {
        return nil, fmt.Errorf("s3 fetch: %w", err)
    }

    prompt := buildVisionPrompt(req.TargetLanguage)

    msg, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
        Model:     anthropic.F(a.model),
        MaxTokens: anthropic.F(int64(a.maxTokens)),
        Messages: anthropic.F([]anthropic.MessageParam{{
            Role: anthropic.F(anthropic.MessageParamRoleUser),
            Content: anthropic.F([]anthropic.ContentBlockParamUnion{
                anthropic.ImageBlockParam{
                    Type: anthropic.F(anthropic.ImageBlockParamTypeImage),
                    Source: anthropic.F(anthropic.ImageBlockParamSource{
                        Type:      anthropic.F(anthropic.Base64ImageSourceTypeBase64),
                        MediaType: anthropic.F(anthropic.Base64ImageSourceMediaTypeImageJpeg),
                        Data:      anthropic.F(imageB64),
                    }),
                },
                anthropic.TextBlockParam{
                    Type: anthropic.F(anthropic.TextBlockParamTypeText),
                    Text: anthropic.F(prompt),
                },
            }),
        }}),
    })
    if err != nil {
        return nil, fmt.Errorf("claude api: %w", err)
    }

    return parseVisionJSON(msg.Content[0].Text, "claude/"+a.model)
}
```

### 3.3 Implementare Ollama Vision Agent

```go
// internal/agent/providers/ollama/vision.go

type OllamaVisionAgent struct {
    baseURL string // ex: "http://ollama-server:11434"
    model   string // ex: "llava:13b"
    timeout time.Duration
}

func (a *OllamaVisionAgent) ExtractFields(ctx context.Context, req VisionRequest) (*VisionResult, error) {
    payload := OllamaRequest{
        Model:  a.model,
        Prompt: buildVisionPrompt(req.TargetLanguage),
        Images: []string{base64.StdEncoding.EncodeToString(imageBytes)},
        Stream: false,
        Options: map[string]any{"temperature": 0.1, "num_ctx": 4096},
    }
    // ...
    return parseVisionJSON(resp.Response, "ollama/"+a.model)
}

func (a *OllamaVisionAgent) EstimateCost(_ int64) float64 { return 0.0 } // on-premise = cost 0
```

### 3.4 AgentFactory

```go
// internal/agent/factory.go

func (f *AgentFactory) GetVisionAgent(ctx context.Context, workspaceID string) (VisionAgent, error) {
    // 1. Check Redis cache (TTL 5min)
    cfg, err := f.configFromCache(ctx, workspaceID)
    if err != nil {
        cfg, err = f.configFromDB(ctx, workspaceID)
        if err != nil { return nil, err }
        f.cacheConfig(ctx, workspaceID, cfg) // async
    }

    switch cfg.VisionProvider {
    case "claude":
        apiKey, _ := f.kms.Decrypt(cfg.APIKeyEncrypted)
        return claude.NewVisionAgent(apiKey, cfg.VisionModel), nil
    case "ollama":
        return ollama.NewVisionAgent(cfg.OllamaURL, cfg.VisionModel), nil
    default:
        return nil, fmt.Errorf("unknown vision provider: %s", cfg.VisionProvider)
    }
}
```

### 3.5 Fallback chain

```go
func (s *AgentService) ProcessVisionWithFallback(ctx context.Context, req VisionRequest) (*VisionResult, error) {
    primary, _ := s.factory.GetVisionAgent(ctx, req.WorkspaceID)
    result, err := primary.ExtractFields(ctx, req)
    if err == nil {
        return result, nil
    }

    s.logger.Warn("primary vision agent failed, trying fallback", "provider", primary.Name(), "error", err)

    fallback, err := s.factory.GetFallbackVisionAgent(ctx, req.WorkspaceID)
    if err != nil || fallback == nil {
        return nil, fmt.Errorf("primary failed (%w), no fallback configured", err)
    }

    return fallback.ExtractFields(ctx, req)
}
```

### 3.6 Vision Prompt Template

```go
const VisionPromptRO = `Ești un expert în analiza etichetelor de produse și legislația română.
Analizează imaginea și extrage TOATE informațiile textuale vizibile.

Returnează EXCLUSIV un JSON valid cu structura exactă de mai jos (fără text în afara JSON):

{
  "product_name": "denumirea produsului",
  "ingredients": "lista completă ingrediente în ordinea originală",
  "manufacturer": "producător / fabricant",
  "address": "adresa completă producător",
  "quantity": "cantitate netă cu unitate (ex: 500g, 250ml)",
  "expiry_date": "termen valabilitate sau data durabilității minimale",
  "warnings": "toate avertismentele și precauțiile",
  "country_of_origin": "țara de origine",
  "storage_conditions": "condiții de păstrare",
  "lot_number": "număr lot / serie dacă există",
  "category": "food|cosmetic|electronics|toy|other",
  "detected_language": "codul ISO al limbii detectate (ex: zh, ar, ko, de)",
  "confidence": {
    "product_name": 0.95,
    "ingredients": 0.87,
    "manufacturer": 0.92
  }
}

Reguli:
- Dacă un câmp nu este vizibil, returnează null (nu string gol)
- Confidence score: 0.0–1.0 per câmp (1.0 = certitudine completă)
- Pentru ingrediente cosmetice: păstrează nomenclatura INCI originală
- Nu traduce încă — returnează textul ORIGINAL din imagine
`
```

---

## 4. Database Schema

### 4.1 auth_db

**users**

| Coloană | Tip | Constrângeri | Descriere |
|---|---|---|---|
| id | UUID | PK | Identificator unic |
| email | VARCHAR(255) | UNIQUE, NOT NULL | Email — username |
| password_hash | VARCHAR(255) | NULLABLE | Null pentru OAuth |
| oauth_provider | VARCHAR(50) | NULLABLE | google \| apple \| null |
| oauth_sub | VARCHAR(255) | NULLABLE | Subject ID de la provider OAuth |
| is_verified | BOOLEAN | DEFAULT false | True după confirmare email |
| created_at | TIMESTAMPTZ | NOT NULL | |
| last_login_at | TIMESTAMPTZ | NULLABLE | |

**refresh_tokens**

| Coloană | Tip | Constrângeri | Descriere |
|---|---|---|---|
| id | UUID | PK | |
| user_id | UUID | FK users | |
| token_hash | VARCHAR(255) | UNIQUE, NOT NULL | Hash SHA-256 al tokenului |
| expires_at | TIMESTAMPTZ | NOT NULL | |
| revoked_at | TIMESTAMPTZ | NULLABLE | Null dacă activ |
| device_info | JSONB | NULLABLE | User-agent, IP |

### 4.2 workspace_db

**workspaces**

| Coloană | Tip | Constrângeri | Descriere |
|---|---|---|---|
| id | UUID | PK | Echivalent "tenant" |
| name | VARCHAR(255) | NOT NULL | Denumire firmă |
| cui | VARCHAR(20) | NULLABLE | Cod Unic de Identificare fiscală |
| plan | VARCHAR(50) | NOT NULL | starter \| business \| enterprise |
| label_quota_monthly | INTEGER | NOT NULL | Limită etichete/lună (100/500/-1) |
| label_quota_used | INTEGER | DEFAULT 0 | Contor reset la 1 ale lunii |
| stripe_customer_id | VARCHAR(100) | NULLABLE | |
| stripe_subscription_id | VARCHAR(100) | NULLABLE | |
| subscription_expires_at | TIMESTAMPTZ | NULLABLE | |
| logo_s3_key | VARCHAR(500) | NULLABLE | |
| created_at | TIMESTAMPTZ | NOT NULL | |

**workspace_members**

| Coloană | Tip | Constrângeri | Descriere |
|---|---|---|---|
| id | UUID | PK | |
| workspace_id | UUID | FK workspaces | |
| user_id | UUID | FK (auth-svc) | Cross-serviciu (denormalizat) |
| role | VARCHAR(50) | NOT NULL | admin \| operator \| viewer |
| invited_by_user_id | UUID | NULLABLE | |
| invited_at | TIMESTAMPTZ | NOT NULL | |
| accepted_at | TIMESTAMPTZ | NULLABLE | Null = invitație în așteptare |
| revoked_at | TIMESTAMPTZ | NULLABLE | Null = activ |

### 4.3 agent_db

**agent_configs**

| Coloană | Tip | Constrângeri | Descriere |
|---|---|---|---|
| id | UUID | PK | |
| workspace_id | UUID | UNIQUE NOT NULL | Un set de config per workspace |
| vision_provider | VARCHAR(50) | NOT NULL | claude \| ollama |
| vision_model | VARCHAR(100) | NOT NULL | ex: claude-sonnet-4-6, llava:13b |
| transl_provider | VARCHAR(50) | NOT NULL | |
| transl_model | VARCHAR(100) | NOT NULL | |
| valid_provider | VARCHAR(50) | NOT NULL | claude \| ollama \| rules_engine |
| fallback_provider | VARCHAR(50) | NULLABLE | Provider backup la eroare primar |
| ollama_url | VARCHAR(500) | NULLABLE | URL server Ollama on-premise |
| api_key_encrypted | BYTEA | NULLABLE | AES-256-GCM |
| updated_at | TIMESTAMPTZ | NOT NULL | |
| updated_by_user_id | UUID | NOT NULL | |

**agent_call_logs**

| Coloană | Tip | Constrângeri | Descriere |
|---|---|---|---|
| id | UUID | PK | |
| workspace_id | UUID | NOT NULL, INDEX | |
| label_id | UUID | NOT NULL | |
| agent_type | VARCHAR(50) | NOT NULL | vision \| translation \| validation |
| provider | VARCHAR(50) | NOT NULL | |
| model | VARCHAR(100) | NOT NULL | |
| tokens_input | INTEGER | NULLABLE | |
| tokens_output | INTEGER | NULLABLE | |
| cost_usd | NUMERIC(10,6) | DEFAULT 0 | 0 pentru Ollama on-prem |
| latency_ms | INTEGER | NOT NULL | |
| success | BOOLEAN | NOT NULL | |
| error_message | TEXT | NULLABLE | |
| called_at | TIMESTAMPTZ | NOT NULL, INDEX | |

### 4.4 label_db

**labels**

| Coloană | Tip | Constrângeri | Descriere |
|---|---|---|---|
| id | UUID | PK | |
| workspace_id | UUID | NOT NULL, INDEX | |
| created_by_user_id | UUID | NOT NULL | |
| status | VARCHAR(50) | NOT NULL | draft \| processing \| review \| confirmed \| archived |
| image_s3_key | VARCHAR(500) | NOT NULL | Imaginea originală |
| product_id | UUID | NULLABLE, FK products | |
| category | VARCHAR(50) | NULLABLE | food\|cosmetic\|electronics\|toy\|other |
| detected_language | VARCHAR(10) | NULLABLE | ISO 639-1 |
| ai_raw_json | JSONB | NULLABLE | Output brut Vision Agent |
| fields_translated | JSONB | NULLABLE | Câmpuri traduse (editate de user) |
| compliance_score | SMALLINT | NULLABLE | 0–100 |
| missing_fields | JSONB | NULLABLE | Lista câmpuri obligatorii lipsă |
| confirmed_at | TIMESTAMPTZ | NULLABLE | |
| created_at | TIMESTAMPTZ | NOT NULL, INDEX | |
| updated_at | TIMESTAMPTZ | NOT NULL | |

**print_jobs**

| Coloană | Tip | Constrângeri | Descriere |
|---|---|---|---|
| id | UUID | PK | |
| label_id | UUID | NOT NULL, FK labels | |
| workspace_id | UUID | NOT NULL | |
| format | VARCHAR(20) | NOT NULL | pdf \| zpl |
| label_size | VARCHAR(50) | NOT NULL | 50x30 \| 62x29 \| 100x50 \| custom |
| status | VARCHAR(50) | NOT NULL | pending\|generating\|ready\|sent\|printed\|failed |
| pdf_s3_key | VARCHAR(500) | NULLABLE | URL pre-signed generat la ready |
| zpl_content | TEXT | NULLABLE | |
| printer_id | VARCHAR(255) | NULLABLE | |
| print_gateway_id | VARCHAR(255) | NULLABLE | |
| error_message | TEXT | NULLABLE | |
| created_at | TIMESTAMPTZ | NOT NULL | |
| completed_at | TIMESTAMPTZ | NULLABLE | |

---

## 5. API Contract (REST — api-gateway :8080)

**Base URL:** `https://api.etiketai.ro/v1`
**Auth:** `Bearer <JWT>` în header `Authorization`
**Error format:** `{ "error": "mesaj", "code": "ERROR_CODE", "details": {} }`

### 5.1 Auth

**POST `/auth/register`**
```json
// Request
{ "email": "user@firma.ro", "password": "SecurePass1!", "workspace_name": "Import SRL", "cui": "RO12345678" }
// Response
{ "user_id": "uuid", "workspace_id": "uuid", "message": "Verification email sent" }
```

**POST `/auth/login`**
```json
// Request
{ "email": "user@firma.ro", "password": "SecurePass1!" }
// Response
{ "access_token": "eyJ...", "refresh_token": "eyJ...", "expires_in": 900,
  "user": { "id":"uuid", "email":"...", "workspace_id":"uuid", "role":"admin" } }
```

**POST `/auth/refresh`**
```json
// Request
{ "refresh_token": "eyJ..." }
// Response
{ "access_token": "eyJ...", "expires_in": 900 }
```

### 5.2 Labels

**POST `/labels/upload`** — multipart/form-data, field: `image` (JPEG/PNG/PDF, max 10MB)
```json
// Response
{ "label_id": "uuid", "status": "processing", "job_id": "asynq-job-id", "image_url": "https://cdn.../thumb.jpg" }
```

**GET `/labels/:id/status`**
```json
{
  "label_id": "uuid",
  "status": "done",
  "progress": 100,
  "fields": {
    "product_name": { "value": "Cremă hidratantă", "confidence": 0.95, "original": "保湿霜" },
    "ingredients":  { "value": "Aqua, Glycerin...", "confidence": 0.88, "original": "水,甘油..." }
  },
  "compliance": {
    "score": 75,
    "missing": [
      { "field": "importer_address", "severity": "blocker" },
      { "field": "lot_number", "severity": "warning" }
    ]
  },
  "provider_used": "claude",
  "processing_ms": 2340
}
```

**PATCH `/labels/:id/fields`**
```json
// Request
{ "product_name": "Cremă hidratantă față", "importer_address": "SC Import SRL, Str. Exemplu 1, București" }
// Response
{ "label_id": "uuid", "updated_fields": ["product_name","importer_address"], "compliance": { "score": 95, "missing": [] } }
```

**POST `/labels/:id/confirm`** — Decrement automat din quota lunară. 402 dacă quota = 0.
```json
// Response
{ "label_id": "uuid", "status": "confirmed", "quota_remaining": 87, "confirmed_at": "2026-03-20T14:32:00Z" }
```

**GET `/labels`** — `?status=confirmed&category=food&q=crema&page=2&from=2026-01-01&to=2026-03-31`
```json
{
  "data": [{ "id":"uuid","status":"confirmed","product_name":"...","category":"food","compliance_score":95,"created_at":"...","operator":"Ion P." }],
  "pagination": { "total": 342, "page": 1, "per_page": 50 }
}
```

### 5.3 Print

**POST `/labels/:id/print/pdf`**
```json
// Request
{ "size": "50x30", "copies": 1, "labels_per_page": 4 }
// Response
{ "job_id": "uuid", "status": "generating", "estimated_seconds": 5 }
```

**GET `/labels/:id/print/pdf/:job_id`**
```json
{ "status": "ready", "download_url": "https://r2.etiketai.ro/signed/...", "expires_at": "...", "pages": 1, "labels_count": 4 }
```

### 5.4 Agent Config (Admin)

**GET `/admin/workspaces/:id/agent-config`**
```json
{
  "workspace_id": "uuid",
  "vision":      { "provider":"claude",  "model":"claude-sonnet-4-6" },
  "translation": { "provider":"claude",  "model":"claude-sonnet-4-6" },
  "validation":  { "provider":"ollama",  "model":"llava:13b" },
  "fallback":    { "provider":"ollama",  "model":"llava:13b" },
  "ollama_url":  "http://192.168.1.100:11434",
  "updated_at":  "2026-03-15T10:00:00Z"
}
```

**PUT `/admin/workspaces/:id/agent-config`** — Efectiv imediat, invalidează Redis cache.
```json
// Request
{
  "vision":      { "provider":"ollama", "model":"llava:34b" },
  "translation": { "provider":"claude", "model":"claude-sonnet-4-6" },
  "fallback":    { "provider":"claude", "model":"claude-haiku-4-5-20251001" },
  "ollama_url":  "http://10.0.0.5:11434",
  "api_key":     "sk-ant-..."
}
// Response
{ "success": true, "cache_invalidated": true }
```

**POST `/admin/workspaces/:id/agent-config/test`**
```json
// Request
{ "agent_type": "vision" }
// Response
{ "success": true, "provider": "ollama", "latency_ms": 1240, "estimated_cost_per_1000": 0.00 }
```

---

## 6. Monorepo Layout

```
etiketai/
├── services/
│   ├── api-gateway/          # Go Chi — routing, auth middleware, rate limit
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── middleware/   # jwt.go, ratelimit.go, cors.go
│   │   │   ├── handlers/     # auth.go, labels.go, print.go, admin.go
│   │   │   └── proxy/        # grpc clients pentru fiecare svc
│   │   └── Dockerfile
│   │
│   ├── auth-svc/             # Go gRPC
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── service/      # auth.go — login, register, refresh
│   │   │   ├── repo/         # postgres.go
│   │   │   └── oauth/        # google.go
│   │   └── Dockerfile
│   │
│   ├── agent-svc/            # Go gRPC — AI Agent Engine
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── agent/
│   │   │   │   ├── interfaces.go
│   │   │   │   ├── factory.go
│   │   │   │   ├── providers/
│   │   │   │   │   ├── claude/   # vision.go, translation.go, validation.go
│   │   │   │   │   └── ollama/   # vision.go, translation.go
│   │   │   │   └── prompts/      # vision_ro.go, transl_*.go (per categorie)
│   │   │   ├── service/          # ProcessVision, fallback chain
│   │   │   ├── repo/             # config.go, logs.go
│   │   │   └── rules/            # validation rules engine
│   │   │       ├── engine.go
│   │   │       └── rules/        # food.yaml, cosmetic.yaml, electronics.yaml, toy.yaml
│   │   └── Dockerfile
│   │
│   ├── label-svc/
│   ├── workspace-svc/
│   ├── print-svc/
│   ├── notification-svc/
│   └── admin-svc/
│
├── mobile/                   # Flutter iOS + Android
│   └── lib/
│       ├── features/
│       │   ├── auth/
│       │   ├── camera/
│       │   ├── editor/
│       │   ├── library/
│       │   └── print/
│       └── core/
│           ├── api/
│           ├── storage/
│           └── theme/
│
├── web/                      # Next.js Dashboard
│   └── app/
│       ├── dashboard/
│       ├── products/
│       ├── settings/
│       └── admin/
│
├── proto/                    # Protocol Buffers (shared)
│   ├── agent/v1/agent.proto
│   ├── auth/v1/auth.proto
│   ├── label/v1/label.proto
│   └── workspace/v1/workspace.proto
│
├── infra/
│   ├── docker-compose.yml
│   ├── k8s/
│   └── terraform/
│
├── scripts/
│   ├── migrate.sh
│   ├── gen-proto.sh
│   └── seed-dev.sh
│
├── go.work
└── Taskfile.yml
```

---

## 7. Infrastructure

### 7.1 Docker Compose — development local

```yaml
# infra/docker-compose.yml
version: "3.9"
services:

  api-gateway:
    build: ./services/api-gateway
    ports: ["8080:8080"]
    environment:
      AUTH_SVC_ADDR:  auth-svc:8081
      AGENT_SVC_ADDR: agent-svc:8084
      LABEL_SVC_ADDR: label-svc:8083
    depends_on: [auth-svc, agent-svc, label-svc]

  agent-svc:
    build: ./services/agent-svc
    ports: ["8084:8084"]
    environment:
      ANTHROPIC_API_KEY: ${ANTHROPIC_API_KEY}
      OLLAMA_URL:        http://ollama:11434
      AGENT_DB_DSN:      postgres://agent:pass@agent-db:5432/agent_db
      REDIS_URL:         redis://redis:6379
      ENCRYPTION_KEY:    ${AES_KEY_HEX}

  ollama:
    image: ollama/ollama:latest
    ports: ["11434:11434"]
    volumes: ["ollama_models:/root/.ollama"]

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]

  minio:
    image: minio/minio
    ports: ["9000:9000", "9001:9001"]
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER:     minioadmin
      MINIO_ROOT_PASSWORD: minioadmin123

  auth-db:      { image: postgres:16, environment: { POSTGRES_DB: auth_db } }
  label-db:     { image: postgres:16, environment: { POSTGRES_DB: label_db } }
  agent-db:     { image: postgres:16, environment: { POSTGRES_DB: agent_db } }
  workspace-db: { image: postgres:16, environment: { POSTGRES_DB: workspace_db } }

volumes:
  ollama_models:
```

### 7.2 Environment Variables — agent-svc

| Variabilă | Exemplu | Descriere |
|---|---|---|
| ANTHROPIC_API_KEY | sk-ant-api03-... | API key Anthropic; null = Ollama only |
| OLLAMA_URL | http://10.0.0.5:11434 | URL server Ollama on-premise |
| OLLAMA_DEFAULT_MODEL | llava:13b | Model default |
| ENCRYPTION_KEY | 64-char hex | AES-256-GCM pentru API keys din DB |
| AGENT_CACHE_TTL_SECONDS | 300 | TTL cache config agenți în Redis |
| VISION_TIMEOUT_SECONDS | 60 | Timeout apel Vision Agent |
| TRANSL_TIMEOUT_SECONDS | 30 | Timeout apel Translation Agent |
| MAX_RETRIES | 3 | Retry la eroare provider (exp. backoff) |
| COST_TRACKING_ENABLED | true | Activare tracking cost per workspace |

### 7.3 Kubernetes HPA — agent-svc

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: agent-svc-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: agent-svc
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target: { type: Utilization, averageUtilization: 70 }
```
