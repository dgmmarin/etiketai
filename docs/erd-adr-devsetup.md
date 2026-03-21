# EtiketAI — ERD · Architecture Decision Records · Dev Setup Guide

> Go Microservices · Flutter · Claude + Ollama · PostgreSQL
> Draft v1.0 · Martie 2026

---

## Cuprins

1. [Entity Relationship Diagram](#1-entity-relationship-diagram)
2. [Architecture Decision Records](#2-architecture-decision-records)
3. [Dev Setup Guide](#3-dev-setup-guide)
4. [Ollama Setup](#34-flutter-app-setup)
5. [Taskfile — comenzi utile](#35-taskfile--comenzi-utile-zilnic)
6. [Troubleshooting](#36-troubleshooting-frecvent)

---

## 1. Entity Relationship Diagram

Tabelele sunt grupate per serviciu. Relațiile cross-serviciu sunt **logice (denormalizate prin UUID)**, nu FK-uri fizice — conform pattern-ului Database-per-Service.

### 1.1 Relații principale

| Tabel sursă | Relație | Tabel țintă | Descriere |
|---|---|---|---|
| users | 1 → N | workspace_members | Un user poate fi în mai multe workspace-uri cu roluri diferite |
| workspaces | 1 → N | workspace_members | Un workspace are mai mulți membri |
| workspaces | 1 → 1 | agent_configs | Fiecare workspace are exact o configurație de agenți |
| workspaces | 1 → N | labels | Toate etichetele aparțin unui workspace |
| users | 1 → N | labels | Un user creează mai multe etichete |
| labels | 1 → N | label_audit_log | Fiecare etichetă are un trail complet de modificări |
| labels | 1 → N | print_jobs | O etichetă poate fi printată de multiple ori |
| labels | N → 1 | products | Opțional: eticheta e asociată unui produs salvat |
| agent_configs | 1 → N | agent_call_logs | Fiecare config generează log-uri de apeluri AI |
| refresh_tokens | N → 1 | users | Multiple tokeni activi per user (multi-device) |

### 1.2 ERD textual — schema completă

```
┌─────────────────────────────────────────────────────────────────────────┐
│  AUTH_DB                                                                 │
│                                                                          │
│  users                          refresh_tokens                           │
│  ─────────────────────          ──────────────────────────               │
│  id            UUID PK          id            UUID PK                   │
│  email         VARCHAR UNIQUE   user_id       UUID ──────────► users.id │
│  password_hash VARCHAR NULL     token_hash    VARCHAR UNIQUE             │
│  oauth_provider VARCHAR NULL    expires_at    TIMESTAMPTZ               │
│  oauth_sub     VARCHAR NULL     revoked_at    TIMESTAMPTZ NULL           │
│  is_verified   BOOLEAN          device_info   JSONB NULL                 │
│  created_at    TIMESTAMPTZ                                               │
│  last_login_at TIMESTAMPTZ NULL                                          │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│  WORKSPACE_DB                                                            │
│                                                                          │
│  workspaces                     workspace_members                        │
│  ─────────────────────          ──────────────────────────               │
│  id            UUID PK ◄────── workspace_id  UUID FK                   │
│  name          VARCHAR          user_id       UUID  (cross-svc ref)     │
│  cui           VARCHAR NULL     role          VARCHAR  admin|op|viewer   │
│  plan          VARCHAR          invited_by    UUID NULL                  │
│  label_quota_monthly INT        invited_at    TIMESTAMPTZ               │
│  label_quota_used    INT        accepted_at   TIMESTAMPTZ NULL           │
│  stripe_customer_id  VARCHAR    revoked_at    TIMESTAMPTZ NULL           │
│  subscription_expires TIMESTAMPTZ                                        │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│  AGENT_DB                                                                │
│                                                                          │
│  agent_configs                  agent_call_logs                          │
│  ─────────────────────          ──────────────────────────               │
│  id            UUID PK          id            UUID PK                   │
│  workspace_id  UUID UNIQUE ◄─── workspace_id  UUID  INDEX               │
│  vision_provider VARCHAR        label_id      UUID  INDEX               │
│  vision_model  VARCHAR          agent_type    VARCHAR  vision|transl|val │
│  transl_provider VARCHAR        provider      VARCHAR                    │
│  transl_model  VARCHAR          model         VARCHAR                    │
│  valid_provider VARCHAR         tokens_input  INT NULL                   │
│  fallback_provider VARCHAR      tokens_output INT NULL                   │
│  ollama_url    VARCHAR NULL      cost_usd      NUMERIC(10,6)             │
│  api_key_encrypted BYTEA NULL   latency_ms    INT                        │
│  updated_at    TIMESTAMPTZ      success       BOOLEAN                    │
│  updated_by    UUID             called_at     TIMESTAMPTZ  INDEX         │
└─────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────┐
│  LABEL_DB                                                                │
│                                                                          │
│  products              labels                    label_audit_log         │
│  ──────────────         ───────────────────       ─────────────────      │
│  id    UUID PK ◄─────── product_id UUID NULL     id       UUID PK       │
│  workspace_id UUID      id         UUID PK ◄──── label_id UUID FK       │
│  sku   VARCHAR NULL     workspace_id UUID         user_id  UUID          │
│  name  VARCHAR          user_id    UUID            action  VARCHAR       │
│  category VARCHAR       status     VARCHAR         changes JSONB NULL    │
│  default_fields JSONB   image_s3_key VARCHAR       metadata JSONB NULL   │
│  print_count INT        category   VARCHAR NULL    created_at TIMESTAMPTZ│
│  created_at TIMESTAMPTZ detected_language VARCHAR                        │
│                         ai_raw_json JSONB NULL                           │
│                         fields_translated JSONB                          │
│                         compliance_score SMALLINT                        │
│                         missing_fields JSONB                             │
│                         confirmed_at TIMESTAMPTZ                         │
│                         created_at TIMESTAMPTZ  INDEX                    │
│                         updated_at TIMESTAMPTZ                           │
│                                 │                                        │
│                                 ▼                                        │
│                         print_jobs                                       │
│                         ─────────────────────                            │
│                         id         UUID PK                               │
│                         label_id   UUID FK                               │
│                         workspace_id UUID                                │
│                         format     VARCHAR  pdf|zpl                      │
│                         label_size VARCHAR                               │
│                         status     VARCHAR                               │
│                         pdf_s3_key VARCHAR NULL                          │
│                         zpl_content TEXT NULL                            │
│                         created_at TIMESTAMPTZ                           │
└─────────────────────────────────────────────────────────────────────────┘
```

### 1.3 Indecși critici pentru performanță

| Tabel | Index | Justificare |
|---|---|---|
| labels | workspace_id, created_at | Listare etichete per workspace, sort by date |
| labels | workspace_id, status | Filtrare rapidă pe status |
| label_audit_log | label_id | History per etichetă — query ANPC audit |
| agent_call_logs | workspace_id, called_at | Rapoarte cost și utilizare per tenant |
| agent_call_logs | called_at | Cleanup job: ștergere log-uri > 90 zile |
| print_jobs | label_id | Status print per etichetă |
| refresh_tokens | user_id, revoked_at | Listare tokeni activi per user |
| workspace_members | workspace_id, role | RBAC check — cel mai frecvent query de autorizare |

---

## 2. Architecture Decision Records (ADR)

Fiecare ADR documentează o decizie arhitecturală semnificativă. ADR-urile sunt imutabile — deciziile superseded primesc un ADR nou.

---

### ADR-001 · Go pentru backend microservices

**Status:** Accepted

**Context:** Avem nevoie de un backend performant, cu latență mică pentru procesarea pipeline-ului AI (vision → traducere → validare în serie). Echipa are experiență cu Go și Node.js.

**Decizie:** Go (1.22+) pentru toate microserviciile backend. Framework HTTP: **Chi** pentru api-gateway și admin-svc. **gRPC** pentru comunicare inter-serviciu.

**Consecințe:**
- ✅ Timp de startup < 100ms per serviciu, Docker images < 20MB
- ✅ Goroutine pool ideal pentru procesare paralelă a batch-urilor
- ✅ Type safety la compile time reduce buguri în producție
- ❌ Ecosistem mai mic față de Node.js/Python pentru unele librării AI
- ❌ Generics relativ noi în Go — limitări de familiarizat

**Alternative considerate:** Node.js (Fastify), Python (FastAPI), Rust

---

### ADR-002 · Microservices cu Database-per-Service

**Status:** Accepted

**Context:** Sistemul trebuie să suporte mai mulți clienți enterprise cu cerințe diferite de scalare.

**Decizie:** Arhitectura microservices cu **Database-per-Service**: fiecare serviciu Go deține propria bază de date PostgreSQL. Comunicarea cross-serviciu exclusiv prin **gRPC** (sync) sau **Redis Streams** (async). **Monorepo** pentru management simplificat.

**Consecințe:**
- ✅ Fiecare serviciu scalează independent
- ✅ Izolare completă — un bug în print-svc nu afectează auth-svc
- ❌ Complexitate operațională ridicată — 7 baze de date
- ❌ Nu există JOIN-uri cross-serviciu
- ❌ Distributed transactions dificile — necesită saga pattern

**Mitigare:** Docker Compose pentru dev local; Kubernetes Helm charts pentru prod

---

### ADR-003 · Provider pattern pentru AI Agents — Claude + Ollama din start

**Status:** Accepted

**Context:** Clienți cu cerințe diferite: unii vor Claude pentru calitate maximă, alții cer on-premise din motive GDPR.

**Decizie:** **Provider Pattern** curat în Go: interfețe separate pentru `VisionAgent`, `TranslationAgent`, `ValidationAgent`. Implementări concrete pentru **Claude** (SDK oficial Anthropic Go) și **Ollama** (HTTP client). `AgentFactory` construiește implementarea corectă din config workspace-ului, stocată în **Redis cu TTL 5 minute**.

**Consecințe:**
- ✅ Adăugare provider nou = implementare interfață + înregistrare în factory (< 1 zi)
- ✅ Clienți enterprise pot rula 100% on-premise cu Ollama
- ✅ Cost zero pentru procesare AI on-premise
- ❌ Ollama (llava) are acuratețe mai scăzută față de Claude pe texte complexe
- ❌ Hardware requirements ridicate pentru Ollama cu modele mari

**Mitigare:** Fallback automat la Claude dacă Ollama returnează confidence score < 0.6

---

### ADR-004 · Flutter pentru mobile (iOS + Android)

**Status:** Accepted

**Context:** Aplicația mobilă trebuie să funcționeze nativ pe iOS și Android cu acces la camera hardware.

**Decizie:** **Flutter** (Dart) cu o singură codebase. State management: **Riverpod**. HTTP: **Dio** cu interceptor refresh JWT. Camera: `camera` package cu overlay custom Flutter Canvas. Secure storage: `flutter_secure_storage`. Notificări: `firebase_messaging`.

**Consecințe:**
- ✅ O singură codebase = jumătate din costul de development
- ✅ Dart compilat — performanță aproape de nativ
- ✅ Hot reload accelerează iterațiile UI
- ❌ Bundle size mai mare față de nativ (~+10MB)
- ❌ Un singur developer = risc de bus factor

**Alternative considerate:** React Native, Swift + Kotlin nativ, PWA

---

### ADR-005 · gRPC pentru comunicare inter-serviciu

**Status:** Accepted

**Context:** Microserviciile Go trebuie să comunice eficient cu contract strict.

**Decizie:** **gRPC cu Protocol Buffers** pentru toate apelurile sincrone inter-serviciu. Proto files în `/proto/` la nivel de monorepo — sursa de adevăr. API Gateway translatează REST → gRPC pentru clienții externi.

**Consecințe:**
- ✅ Serializare 5–10× mai rapidă față de JSON
- ✅ Contract strict — schimbările breaking detectate la compile time
- ✅ Streaming bidirectional nativ
- ❌ Nu poate fi testat direct din browser/curl (necesită grpcurl)
- ❌ Learning curve pentru developeri obișnuiți cu REST

---

### ADR-006 · Redis pentru caching, rate limiting și job queue

**Status:** Accepted

**Context:** 3 nevoi distincte: cache config agenți AI, rate limiting, job queue pentru procesare async.

**Decizie:** Un singur cluster **Redis (Valkey-compatible)** servește toate 3 cazurile:
- String keys cu TTL pentru cache agent configs (TTL 5 min, invalidare prin pub/sub)
- Sliding window rate limiter cu Lua script atomic (ZADD/ZREMRANGEBYSCORE)
- **Asynq** (Go library) ca job queue — retry automat, dead letter queue, UI monitoring
- Redis Streams pentru events pub/sub inter-serviciu

**Consecințe:**
- ✅ O singură componentă în loc de 3
- ✅ Asynq oferă UI vizual (Asynqmon)
- ❌ Single point of failure fără Sentinel/Cluster
- ❌ Job-uri pierdute fără AOF/RDB

**Mitigare:** Redis cu AOF persistence + Sentinel cu 3 noduri în producție

---

### ADR-007 · Monorepo cu Go workspaces

**Status:** Accepted

**Context:** 7 microservicii Go + 1 Flutter app + 1 Next.js web + proto files shared.

**Decizie:** **Monorepo cu Go workspaces** (`go.work`). Proto-generated code shared fără versionare. Toolchain: **Task** (taskfile.yml) în loc de Make pentru cross-platform compatibility. Fiecare serviciu are propriul Dockerfile multi-stage.

**Consecințe:**
- ✅ Proto files și shared utilities într-un singur loc
- ✅ Refactoring cross-serviciu vizibil în același PR
- ❌ Repo mare poate fi lent la clone inițial (mitigat cu sparse checkout)

**Alternative considerate:** Poly-repo, Git submodules (respins explicit)

---

### ADR-008 · Stripe pentru billing și abonamente

**Status:** Accepted

**Context:** Model de business abonament lunar/anual cu 3 planuri.

**Decizie:** **Stripe** ca provider de plăți. Stripe Checkout pentru flux de plată (PCI compliance inclusă). Stripe Customer Portal pentru auto-management. Webhooks pentru: `checkout.session.completed`, `customer.subscription.updated/deleted`, `invoice.payment_failed`.

**Consecințe:**
- ✅ PCI DSS compliance inclusă — nu stocăm detalii de card
- ✅ Stripe Portal = self-service pentru clienți
- ❌ Fee 1.4% + 0.25 EUR per tranzacție în Europa

---

## 3. Dev Setup Guide

> De la zero la toate serviciile rulând local în ~30 minute.

### 3.1 Prerequisites

| Tool | Versiune minimă | Instalare |
|---|---|---|
| Go | 1.22+ | https://go.dev/dl/ |
| Docker Desktop | 4.x | https://docker.com/products/docker-desktop |
| Flutter | 3.19+ | https://flutter.dev/docs/get-started/install |
| Node.js | 20 LTS | https://nodejs.org |
| protoc | 3.x | `brew install protobuf` / `apt install protobuf-compiler` |
| Task | 3.x | https://taskfile.dev |
| grpcurl | latest | `brew install grpcurl` |

### 3.2 First-time Setup

**Step 1 — Clone repo și configurare environment**
```bash
git clone https://github.com/etiketai/etiketai.git && cd etiketai
cp .env.example .env.local
# Editează .env.local — cel puțin ANTHROPIC_API_KEY
# Setează AES_KEY_HEX cu: openssl rand -hex 32
```

**Step 2 — Instalare tools Go și generare proto code**
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
task gen-proto
ls services/agent-svc/internal/proto/
```

**Step 3 — Start infrastructură cu Docker Compose**
```bash
docker compose up -d postgres-auth postgres-workspace postgres-label postgres-agent postgres-print redis minio
docker compose ps
docker compose run --rm minio-setup
```

**Step 4 — Rulare migrații baze de date**
```bash
task migrate-all
# Sau per serviciu:
task migrate SERVICE=auth-svc
task migrate SERVICE=workspace-svc
task migrate SERVICE=label-svc
task migrate SERVICE=agent-svc

# Seed date demo
task seed-dev
# user: admin@etiketai.dev / DevPass123!
# workspace: "EtiketAI Demo", plan Business
```

**Step 5 — Start microservicii Go (development mode)**
```bash
# Terminal 1 — Start toate serviciile cu hot-reload (air)
task dev-all

# SAU individual:
task dev SERVICE=auth-svc       # port 8081
task dev SERVICE=workspace-svc  # port 8082
task dev SERVICE=label-svc      # port 8083
task dev SERVICE=agent-svc      # port 8084
task dev SERVICE=print-svc      # port 8085
task dev SERVICE=api-gateway    # port 8080 (entry point)
```

**Step 6 — Verificare că totul funcționează**
```bash
curl http://localhost:8080/health
# Expected: {"status":"ok","services":{"auth":"ok","agent":"ok","label":"ok"}}

curl -X POST http://localhost:8080/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@etiketai.dev","password":"DevPass123!"}'

grpcurl -plaintext localhost:8084 agent.v1.AgentService/GetAgentConfig
```

### 3.3 Ollama Setup (provider on-premise)

**Step 7 — Instalare și configurare Ollama**
```bash
# Opțiunea A: Ollama în Docker Compose (recomandat pentru dev)
docker compose --profile ollama up -d ollama

# Opțiunea B: Ollama nativ
curl -fsSL https://ollama.ai/install.sh | sh
ollama serve &

# Download model vision (dev):
ollama pull llava:7b
# Model producție (mai precis, necesită ~8GB VRAM):
ollama pull llava:13b

ollama run llava:7b "Describe this image" --image /path/to/test-label.jpg
```

> **Mac Apple Silicon (M1/M2/M3):** llava:13b rulează decent pe CPU/Neural Engine fără GPU.
> **Linux cu NVIDIA GPU:** setează `OLLAMA_NUM_GPU=1`.
> **llava:7b** e suficient pentru dev. Producție recomandă llava:13b sau llava:34b.

**Step 8 — Configurare agent-svc să folosească Ollama**
```bash
# În .env.local:
OLLAMA_URL=http://localhost:11434
OLLAMA_DEFAULT_MODEL=llava:7b

# Test conexiune:
curl -X POST http://localhost:8080/v1/admin/workspaces/demo/agent-config/test \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"agent_type":"vision"}'
```

> Dacă `ANTHROPIC_API_KEY` lipsește, agent-svc folosește automat Ollama ca provider default.

### 3.4 Flutter App Setup

**Step 9 — Start aplicația Flutter**
```bash
cd mobile
flutter pub get
cp lib/core/config/config.dev.dart.example lib/core/config/config.dev.dart
# config.dev.dart: const apiBaseUrl = "http://localhost:8080/v1"

flutter run --dart-define=ENV=dev
flutter run -d "iPhone 15 Pro" --dart-define=ENV=dev
flutter run -d emulator-5554 --dart-define=ENV=dev
```

> Pentru device fizic: folosește IP-ul local al mașinii (`http://192.168.1.x:8080/v1`).

**Step 10 — Start Web Dashboard (Next.js)**
```bash
cd web
npm install
cp .env.local.example .env.local
# .env.local: NEXT_PUBLIC_API_URL=http://localhost:8080/v1
npm run dev
# Dashboard la http://localhost:3000
```

### 3.5 Taskfile — comenzi utile zilnic

```bash
task dev-all                          # Start toate serviciile Go cu hot-reload
task dev SERVICE=auth-svc             # Start un serviciu specific
task test                             # Run toate testele (go test ./...)
task test-svc SERVICE=agent-svc       # Teste un serviciu specific
task gen-proto                        # Regenerare cod din .proto files
task migrate-all                      # Rulare migrații toate serviciile
task migrate SERVICE=label-svc        # Migrații un serviciu
task migrate-down SERVICE=auth-svc    # Rollback ultima migrație
task seed-dev                         # Populare date demo în toate DB-urile
task lint                             # golangci-lint pe toate serviciile
task build SERVICE=agent-svc          # Build binar Go
task docker-build                     # Build toate Docker images
task logs SERVICE=agent-svc           # Tail logs serviciu în Docker
task db-shell DB=label                # psql shell în label_db
task redis-cli                        # redis-cli conectat la Redis local
task grpc-test SVC=agent-svc METHOD=ProcessVision
task asynqmon                         # Asynq UI la http://localhost:8090
task minio-ui                         # MinIO console la http://localhost:9001
```

### 3.6 Troubleshooting frecvent

| Problemă | Soluție |
|---|---|
| "connection refused" la gRPC intern | Infrastructura (PG, Redis) trebuie să fie healthy înainte de servicii. `task dev-all` pornește în ordinea corectă. |
| agent-svc returnează "provider not configured" | Verifică că `ANTHROPIC_API_KEY` sau `OLLAMA_URL` sunt setate în `.env.local` și că ai restartat serviciul. |
| Ollama timeout la prima imagine | Modelul llava se încarcă la primul apel (~30–60s). Crește `VISION_TIMEOUT_SECONDS=120` pentru dev. |
| Flutter "Connection refused" pe device fizic | Înlocuiește localhost cu IP-ul local (ifconfig). Asigură-te că firewall-ul permite portul 8080. |
| Migrație eșuată — "dirty state" | `task migrate-force SERVICE=xxx VERSION=N` — forțează versiunea. Verifică SQL-ul din migrație. |
| Redis "WRONGTYPE" error | Cache vechi dintr-o versiune anterioară. `task redis-flush` (doar în dev!). |
| Proto code out of sync | `task gen-proto` după orice modificare `.proto`. Fișierele `*.pb.go` sunt gitignored. |
