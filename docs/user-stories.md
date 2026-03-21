# EtiketAI — User Stories & Development Tasks

> Flutter · Go · Configurable AI Agents · REST API
> Draft v1.0 · Martie 2026

---

## Epics & Structură

| Epic | Nume | Scop | Stories |
|---|---|---|---|
| EP-01 | Auth & Onboarding | Înregistrare, login, profil firmă, management abonament | US-01, US-02, US-03 |
| EP-02 | Fotografiere & Capture | Capturarea etichetei din app Flutter, ghidare cameră, upload | US-04, US-05 |
| EP-03 | AI Agent Pipeline | Vision, traducere, validare — agenți configurabili per tenant | US-06, US-07, US-08, US-09 |
| EP-04 | Editor & Validare | Editare câmpuri traduse, verificare conformitate OG 21/1992 | US-10, US-11 |
| EP-05 | Print & Export | Generare PDF/ZPL, trimitere la imprimantă, download | US-12, US-13 |
| EP-06 | Web Dashboard & Admin | Dashboard Next.js, istoric, multi-user, admin panel agenți | US-14, US-15, US-16 |

### Legendă implementare

| Simbol | Semnificație |
|---|---|
| ✅ Done | Implementat și în build |
| 🔄 Partial | Schelet / stub / implementare parțială |
| ⬜ | Neînceput |

### Progres curent (2026-03-21)

**BE implementat:** auth-svc (register/login/refresh/logout/verify-email/verify-token + rate limiter), workspace-svc (create/invite/accept/revoke/quota), label-svc (upload/status/fields/confirm/delete/list + Asynq worker), agent-svc (VisionAgent Claude+Ollama, TranslationAgent Claude, ValidationAgent rules engine, AgentFactory, gRPC handler), api-gateway (proxy-uri gRPC + HTTP handlers + S3 upload real + RBAC + rate limiter), print-svc (PDF + ZPL generator, Asynq worker, reprint URL), billing (Stripe checkout + webhook). **FE Web (Next.js) implementat:** login (email+password + Google OAuth), register, verify-email, dashboard, labels list (filtru status + paginare + export CSV), label detail (editor câmpuri + compliance panel + print SSE streaming), products (CRUD + search), workspace (profil + membri + invite + revocare), billing (plan curent + checkout), admin (agent config + test connection + logs + metrics + rate limits), invite acceptance page. **FE Flutter:** 0% (neînceput).

---

### Legendă estimări

| Simbol | Estimare | Rol | Semnificație |
|---|---|---|---|
| XS | 1–2h | BE | Go Backend |
| S | 3–4h | FE | Flutter Frontend |
| M | 5–8h | AI | AI / Prompts |
| L | 1–2 zile | Full | Full-stack |
| XL | 3–5 zile | QA | QA / Testing |

---

## EP-01 · Auth & Onboarding

### US-01 — Înregistrare cont nou

> Ca importator / retailer / revânzător, vreau să mă pot înregistra rapid cu email și parolă sau Google OAuth.

**Criterii de acceptare:**
- ✓ Formular Flutter cu câmpuri: email, parolă, nume firmă, CUI (opțional)
- ✓ Validare email în timp real (format + disponibilitate via API)
- ✓ Parolă cu cerințe minime afișate inline (min 8 caractere, 1 număr)
- ✓ Go backend creează user + workspace + plan Free în DB
- ✓ Email de confirmare trimis automat (Resend / SendGrid)
- ✓ Redirect la onboarding după confirmare

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-0101 | [FE] Ecran înregistrare Flutter — formular, validare inline, state management | M | FE | ⬜ |
| T-0101-W ✅ | [FE Web] Pagină /register Next.js — formular react-hook-form + Zod (email, parolă, nume firmă, CUI), validare inline | S | FE | ✅ Done |
| T-0102 ✅ | [BE] POST /auth/register — validare, hash parolă (bcrypt), creare user + workspace în PG | M | BE | ✅ Done |
| T-0103 ✅ | [BE] Sistem email transacțional — Resend client + Asynq worker + 4 template-uri | S | BE | ✅ Done (notification-svc/internal/email + worker) |
| T-0104 ✅ | [BE] GET /auth/verify-email?token= — validare token, activare cont | S | BE | ✅ Done |
| T-0104-W ✅ | [FE Web] Pagină /verify-email?token= — 3 stări: loading / success cu link login / error; rută publică | XS | FE | ✅ Done |
| T-0105 | [FE] Google OAuth2 flow în Flutter (google_sign_in package) | M | FE | ⬜ |
| T-0105-W ✅ | [FE Web] Google OAuth pe pagina /login — buton GSI oficial, callback → POST /auth/oauth/google, sesiune identică cu login clasic; activat prin NEXT_PUBLIC_GOOGLE_CLIENT_ID | S | FE | ✅ Done |
| T-0106 | [BE] POST /auth/oauth/google — exchange token, creare user dacă nou | S | BE | ✅ |
| T-0107 | [QA] Teste: email invalid, parolă slabă, email duplicat, OAuth flow | S | QA | ⬜ |

---

### US-02 — Autentificare & sesiune

> Ca utilizator înregistrat, vreau să mă pot autentifica și rămân logat în mod sigur.

**Criterii de acceptare:**
- ✓ Login cu email/parolă și cu Google OAuth
- ✓ JWT access token (15 min) + refresh token (30 zile) stored secure
- ✓ Flutter: token stocat în `flutter_secure_storage` (nu SharedPreferences)
- ✓ Refresh automat transparent când access token expiră
- ✓ Logout invalidează refresh token în DB
- ✓ Rate limiting pe endpoint-ul de login (5 încercări / 15 min)

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-0201 ✅ | [BE] POST /auth/login — verificare credențiale, emitere JWT + refresh token | M | BE | ✅ Done |
| T-0202 ✅ | [BE] POST /auth/refresh — validare refresh token, emitere access token nou | S | BE | ✅ Done |
| T-0203 ✅ | [BE] POST /auth/logout — invalidare refresh token (blacklist în Redis) | S | BE | ✅ Done |
| T-0204 | [FE] Auth interceptor Dio — refresh automat, redirect la login la 401 | M | FE | ⬜ |
| T-0204-W ✅ | [FE Web] HTTP client Next.js — interceptor 401 cu refresh automat (un singur in-flight), token accessor pattern fără circular dep., redirect /login la eșec | M | FE | ✅ Done |
| T-0205 | [FE] Secure token storage cu flutter_secure_storage | S | FE | ⬜ |
| T-0205-W ✅ | [FE Web] Sesiune securizată web — access token în memorie (Zustand), refresh token în httpOnly cookie via Next.js API route; AuthGuard rehydratare silențioasă la mount (funcționează în tab nou) | M | FE | ✅ Done |
| T-0206 | [BE] Rate limiter middleware Go pe /auth/login (Redis sliding window) | S | BE | ✅ |
| T-0207 | [QA] Teste: token expirat, refresh flow, logout, rate limit | M | QA | ⬜ |

---

### US-03 — Profil firmă & abonament

> Ca administrator cont, vreau să pot configura detaliile firmei și gestiona abonamentul.

**Criterii de acceptare:**
- ✓ Ecran profil: nume firmă, CUI, adresă, telefon, logo
- ✓ Vizualizare plan curent + date expiare
- ✓ Upgrade / downgrade plan prin Stripe Checkout
- ✓ Facturile anterioare accesibile din app (via Stripe Portal)
- ✓ Notificare push cu 7 zile înainte de reînnoire

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-0301 | [FE] Ecran Profil Firmă — formular editabil, upload logo | M | FE | ⬜ |
| T-0301-W ✅ | [FE Web] Pagină /workspace — editare nume firmă + CUI (react-hook-form + Zod), management membri (lista + invite + revocare cu confirmare), vizualizare plan | L | FE | ✅ Done |
| T-0302 | [BE] PUT /workspace/profile — validare, update DB, upload logo S3 | S | BE | ✅ |
| T-0303 | [BE] GET /workspace/subscription — plan curent, limite, date | XS | BE | ✅ |
| T-0304 | [BE] POST /billing/create-checkout — Stripe Checkout session | M | BE | ✅ |
| T-0305 | [BE] Webhook handler Stripe — checkout.completed, subscription.updated/deleted | L | BE | ✅ |
| T-0306 | [FE] Ecran Abonament — plan curent, buton upgrade, redirect Stripe | M | FE | ⬜ |
| T-0306-W ✅ | [FE Web] Pagină /billing — plan curent, status, dată expirare, cancel_at_period_end, buton Upgrade → POST /billing/create-checkout → redirect Stripe | M | FE | ✅ Done |
| T-0307 | [BE] Job cron — notificări 7 zile înainte expiare | S | BE | ✅ |
| T-0308 | [QA] Teste Stripe: checkout, webhook, downgrade, card declined | L | QA | ⬜ |

---

## EP-02 · Fotografiere & Capture

### US-04 — Fotografierea etichetei din app

> Ca utilizator mobil, vreau să pot fotografia o etichetă rapid, cu ghidare vizuală pentru o captură optimă.

**Criterii de acceptare:**
- ✓ Buton camera proeminent pe ecranul principal
- ✓ Overlay dreptunghiular de ghidare cu colțuri animate
- ✓ Detectare automată blur — avertisment dacă imaginea nu e clară
- ✓ Flash automat în condiții de lumină slabă
- ✓ Preview imagine + butoane Reface / Confirmă
- ✓ Suport import din galerie și PDF (prima pagină extrasă)
- ✓ Compresie inteligentă: max 2MB trimis la API

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-0401 | [FE] Camera screen Flutter cu overlay dreptunghi + animație colțuri | L | FE | ⬜ |
| T-0402 | [FE] Blur detection algoritm (laplacian variance pe imagine) | M | FE | ⬜ |
| T-0403 | [FE] Flash automat bazat pe luminozitate ambiantă | S | FE | ⬜ |
| T-0404 | [FE] Ecran preview + butoane Reface/Confirmă | S | FE | ⬜ |
| T-0405 | [FE] Import din galerie cu image_picker | S | FE | ⬜ |
| T-0406 | [FE] Import PDF — extragere prima pagina (pdf_render package) | M | FE | ⬜ |
| T-0407 | [FE] Compresie imagine — flutter_image_compress, max 2MB | S | FE | ⬜ |
| T-0408 ✅ | [BE] POST /labels/upload — multipart upload, validare format, stocare S3 | M | BE | ✅ Done (upload real S3 via AWS SDK) |
| T-0409 | [QA] Teste: imagine blur, PDF, format invalid, dimensiune prea mare | M | QA | ⬜ |

---

### US-05 — Gestionare produse repetitive

> Ca utilizator cu volum mare, vreau să pot salva un produs și reutiliza traducerea fără să refotografiez.

**Criterii de acceptare:**
- ✓ După procesare, opțiune "Salvează ca produs" cu SKU / cod intern
- ✓ Librărie produse salvate: search, filtrare pe categorie
- ✓ La fotografierea unui produs existent (QR scan), se propune traducerea salvată
- ✓ Editare producție salvată direct din librărie
- ✓ Contorizare etichete printate per produs

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-0501 | [FE] Bottom sheet "Salvează ca produs" post-procesare cu câmp SKU | M | FE | ⬜ |
| T-0502 | [BE] POST /products — creare produs cu traducere atașată | S | BE | ✅ |
| T-0503 | [FE] Ecran Librărie Produse — search, filtre, card per produs | L | FE | ⬜ |
| T-0503-W ✅ | [FE Web] Pagină /products — tabel produse cu search + filtru categorie + paginare, dialog creare/editare cu validare Zod (SKU, name, category, ingrediente, warnings) | L | FE | ✅ Done |
| T-0504 | [FE] QR/barcode scanner (mobile_scanner) + match cu produse salvate | M | FE | ⬜ |
| T-0505 | [BE] GET /products?q=&category= — search + filtrare paginated | S | BE | ✅ |
| T-0506 | [BE] PATCH /products/:id — editare câmpuri traducere salvată | S | BE | ✅ |
| T-0507 | [BE] Contor print per produs — increment la fiecare job de print | XS | BE | ✅ |
| T-0508 | [QA] Teste: salvare, search, match QR, editare, contorizare | M | QA | ⬜ |

---

## EP-03 · AI Agent Pipeline

> **Arhitectură agent:** Fiecare agent implementează interfața Go `Agent interface { Execute(ctx, Payload) (Result, error) }`. Provider-ul (Claude, Ollama) este injectat la runtime din configurația tenantului. Admin panel-ul permite schimbarea provider-ului per agent per tenant fără redeploy.

### US-06 — Configurare agenți AI per tenant

> Ca administrator cont, vreau să pot alege ce model AI să folosesc pentru fiecare tip de task.

**Criterii de acceptare:**
- ✓ Admin panel web: dropdown per agent (Vision / Translation / Validation)
- ✓ Test connection — apel de test cu imagine demo
- ✓ Fallback configurat: dacă provider-ul primar eșuează, se folosește backup-ul
- ✓ Vizualizare cost estimat per 1000 etichete cu configurația aleasă
- ✓ Config persistat în DB per workspace, aplicat la toate sesiunile

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-0601 ✅ | [BE] Schema DB agent_configs | S | BE | ✅ Done |
| T-0602 ✅ | [BE] Go interface Agent + implementări: ClaudeAgent, OllamaAgent | XL | BE | ✅ Done (VisionAgent Claude+Ollama, TranslationAgent Claude) |
| T-0603 ✅ | [BE] AgentFactory — construiește agentul corect din config workspace la runtime | M | BE | ✅ Done |
| T-0604 ✅ | [BE] PUT /admin/agent-config — actualizare config per agent per workspace | S | BE | ✅ Done (UpdateAgentConfig gRPC) |
| T-0605 ✅ | [BE] POST /admin/agent-config/test — apel test cu imagine demo | M | BE | ✅ Done (TestAgentConfig gRPC) |
| T-0606 ✅ | [BE] Fallback chain — retry pe provider backup la eroare | M | BE | ✅ Done (AgentService.ProcessVision cu GetFallbackVisionAgent) |
| T-0607 | [BE] Secure key management — API keys encrypted AES-256 în DB | L | BE | ✅ |
| T-0608 🔄 | [FE Web] Ecran configurare agenți: dropdown, test connection, cost estimat | L | FE | 🔄 Partial — dropdown provider/model/endpoint + test connection cu latență implementate; cost estimat per 1000 etichete lipsă |
| T-0609 | [QA] Teste: switch provider, fallback, key invalidă, test connection | L | QA | ⬜ |

---

### US-07 — Detecție și extracție text (Vision Agent)

> Ca sistem automat, vreau să detectez automat limba și extrag toate câmpurile textuale dintr-o fotografie de etichetă.

**Criterii de acceptare:**
- ✓ Output JSON structurat: product_name, ingredients, manufacturer, address, quantity, expiry, warnings, country_of_origin, category, detected_language
- ✓ Confidence score per câmp (0.0–1.0) — câmpurile sub 0.7 marcate pentru review manual
- ✓ Suport orice limbă (chineză, arabă, coreeană, japoneză etc.)
- ✓ Procesare async: job în coadă, push notification la finalizare
- ✓ Retry automat de 3 ori la erori de rețea sau timeout

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-0701 ✅ | [BE] VisionAgent.Execute() — prompt structurat, apel provider, parsing JSON | L | AI | ✅ Done |
| T-0702 ✅ | [BE] Prompt engineering Vision: template extracție câmpuri + format JSON strict | L | AI | ✅ Done |
| T-0703 ✅ | [BE] Job handler Asynq — procesare async imagine, update status în DB | M | BE | ✅ Done (label-svc/internal/worker) |
| T-0704 | [BE] Confidence scorer — evaluare calitate output per câmp | M | BE | ✅ |
| T-0705 ✅ | [BE] Retry middleware — exponential backoff (3 retry, max 30s) | S | BE | ✅ Done (Asynq MaxRetry(3)) |
| T-0706 | [BE] POST /labels/:id/process — trigger procesare, returnează job_id | S | BE | ✅ |
| T-0707 ✅ | [BE] GET /labels/:id/status — polling status job | S | BE | ✅ Done |
| T-0708 | [FE] Push notification Flutter la finalizare procesare (FCM/APNs) | M | FE | ⬜ |
| T-0709 | [QA] Teste: limbi diverse (CN/AR/KR/DE), imagini blur, timeout provider | XL | QA | ⬜ |

---

### US-08 — Traducere contextuală în română (Translation Agent)

> Ca sistem automat, vreau să traduc câmpurile extrase în română, respectând terminologia legală.

**Criterii de acceptare:**
- ✓ Prompt contextualizat pe categorie: aliment / cosmetic / electrocasnic / jucărie
- ✓ Terminologie legală: "data durabilității minimale", "ingrediente:", "producător:", "avertismente:"
- ✓ Ingrediente cosmetice: nomenclatură INCI păstrată + traducere nume comune
- ✓ Output JSON cu câmpurile traduse + varianta originală păstrată
- ✓ Câmpurile care nu necesită traducere (cantitate, EAN) rămân neschimbate

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-0801 ✅ | [BE] TranslationAgent.Execute() — prompt per categorie, apel provider, parsing | L | AI | ✅ Done |
| T-0802 ✅ | [AI] Prompt templates per categorie (5): food, cosmetic, electronics, toy, generic | L | AI | ✅ Done |
| T-0803 ✅ | [AI] Prompt pentru terminologie legală română — dicționar termeni obligatorii | M | AI | ✅ Done |
| T-0804 ✅ | [BE] Category-based prompt selector | S | BE | ✅ Done |
| T-0805 ✅ | [BE] Field-level translation — traducere câmp cu câmp | M | BE | ✅ Done |
| T-0806 | [BE] INCI handler — păstrează nomenclatură INCI pentru cosmetice | M | AI | ✅ |
| T-0807 | [QA] Teste per categorie: terminologie, alergeni, avertismente, cantități | L | QA | ⬜ |

---

### US-09 — Validare conformitate legală (Validation Agent)

> Ca sistem automat, vreau să verific automat că traducerea conține toate câmpurile obligatorii conform legii române.

**Criterii de acceptare:**
- ✓ Câmpuri obligatorii globale (OG 21/1992): denumire produs, producător, cantitate, țară origine
- ✓ Câmpuri specifice aliment (Reg. UE 1169/2011): ingrediente, alergeni, valoare nutritivă, termen
- ✓ Câmpuri specifice cosmetic (Reg. CE 1223/2009): ingrediente INCI, avertismente, număr lot
- ✓ Output: lista câmpurilor lipsă cu nivel criticitate (blocker / warning)
- ✓ Score conformitate 0–100% afișat utilizatorului

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-0901 ✅ | [BE] ValidationAgent.Execute() — verificare prezență câmpuri per categorie | M | AI | ✅ Done (rules engine) |
| T-0902 | [BE] Rules engine Go — configurabil YAML: câmpuri obligatorii per categorie | L | BE | ✅ |
| T-0903 ✅ | [BE] Compliance score calculator | S | BE | ✅ Done |
| T-0904 | [BE] Schema YAML câmpuri obligatorii: food.yaml, cosmetic.yaml, electronics.yaml, toy.yaml | M | AI | ✅ |
| T-0905 | [BE] GET /labels/:id/compliance — score + lista câmpuri lipsă + criticitate | S | BE | ✅ |
| T-0906 | [QA] Teste: etichetă completă (100%), câmpuri lipsă diverse categorii | M | QA | ⬜ |

---

## EP-04 · Editor & Validare Utilizator

### US-10 — Editor câmpuri traduse

> Ca utilizator mobil, vreau să pot revizui și corecta câmpurile traduse înainte de a salva eticheta finală.

**Criterii de acceptare:**
- ✓ Ecran editor Flutter cu câmpurile organizate pe secțiuni
- ✓ Câmpurile cu confidence score scăzut (<0.7) marcate cu icon atenție + highlight galben
- ✓ Câmpurile obligatorii lipsă marcate cu roșu — nu se poate salva fără ele
- ✓ Câmpuri editabile inline cu tastatura nativă
- ✓ Buton "Resetează la AI" per câmp individual
- ✓ Preview etichetă finală (render vizual al autocolantului)
- ✓ Salvare draft automată la fiecare modificare (debounced 2s)

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-1001 | [FE] Ecran editor Flutter — layout secțiuni, scroll, state management (Riverpod) | XL | FE | ⬜ |
| T-1001-W ✅ | [FE Web] Pagină /labels/[id] — editor câmpuri cu confidence score (highlight galben <0.7), save draft vs confirm, panel conformitate (score + câmpuri lipsă + avertismente), panel print cu SSE streaming | XL | FE | ✅ Done |
| T-1002 | [FE] Field widget cu indicator confidence: galben <0.7, roșu = lipsă obligatoriu | M | FE | ⬜ |
| T-1002-W ✅ | [FE Web] Confidence indicator pe câmpuri web — highlight galben sub 0.7, afișare score per câmp | S | FE | ✅ Done |
| T-1003 | [FE] Buton "Resetează la AI" per câmp | S | FE | ⬜ |
| T-1004 | [FE] Preview etichetă — render Flutter CustomPaint (50×30mm, 100×50mm etc.) | L | FE | ⬜ |
| T-1005 ✅ | [BE] PATCH /labels/:id/fields — update câmpuri parțial, validare tip câmp | S | BE | ✅ Done |
| T-1006 ✅ | [BE] Auto-save draft — endpoint PATCH cu flag is_draft | XS | BE | ✅ Done |
| T-1007 | [QA] Teste: editare, resetare, validare blocare câmpuri lipsă, preview render | M | QA | ⬜ |

---

### US-11 — Confirmare și salvare etichetă finală

> Ca utilizator mobil, vreau să confirm eticheta validată și o salvez în cont pentru a o putea printa.

**Criterii de acceptare:**
- ✓ Buton "Confirmă & Salvează" activ doar când nu există câmpuri obligatorii lipsă
- ✓ Dialog confirmare cu preview final + compliance score vizibil
- ✓ POST la backend — etichetă marcată ca finalizată (status: confirmed)
- ✓ Feedback vizual de succes + opțiuni: Printează acum / Mergi la Librărie / Etichetă nouă
- ✓ Contorizare automată față de limita planului

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-1101 | [FE] Buton Confirmă cu state disabled când câmpuri lipsă + dialog confirmare | M | FE | ⬜ |
| T-1101-W ✅ | [FE Web] Buton "Confirmă eticheta" în pagina /labels/[id] — avertisment deducere quota, POST /confirm, feedback succes | M | FE | ✅ Done |
| T-1102 ✅ | [BE] POST /labels/:id/confirm — validare finală server-side, marcare confirmed, decrement quota | M | BE | ✅ Done |
| T-1103 ✅ | [BE] Quota check middleware | S | BE | ✅ Done (via workspace-svc CheckAndIncrementQuota) |
| T-1104 | [FE] Ecran succes cu opțiuni acțiuni post-confirmare | S | FE | ⬜ |
| T-1105 ✅ | [BE] Audit log entry — timestamp, user_id, label_id, câmpuri modificate față de AI | S | BE | ✅ Done |
| T-1106 | [QA] Teste: confirmare completă, blocare la câmpuri lipsă, quota depășită, audit log | M | QA | ⬜ |

---

## EP-05 · Print & Export

### US-12 — Generare și descărcare PDF pentru print

> Ca utilizator mobil sau web, vreau să pot genera un PDF gata de printat pe coli autocolante standard.

**Criterii de acceptare:**
- ✓ Selecție dimensiune autocolant: 50×30mm, 62×29mm, 100×50mm, custom
- ✓ Layout PDF respectă marginile de tăiere (crop marks opționale)
- ✓ Font minimum 1.2mm înălțime (cerință Reg. UE 1169/2011) — validat la generare
- ✓ PDF generat server-side în Go (go-pdf), stocat S3, URL pre-signed 24h
- ✓ Download direct din app sau din web dashboard
- ✓ Opțiune reprint fără recalcul

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-1201 ✅ | [BE] PDF generator Go — layout minimal label, no external deps | XL | BE | ✅ Done (print-svc/internal/pdf) |
| T-1202 | [BE] Font size validator — verificare min 1.2mm înălțime | M | BE | ✅ |
| T-1203 ✅ | [BE] POST /labels/:id/print/pdf — trigger generare, returnează job_id | S | BE | ✅ Done (api-gateway → print-svc HTTP) |
| T-1204 ✅ | [BE] Job PDF worker Asynq — generare async, upload S3, pre-signed URL | M | BE | ✅ Done (print-svc/internal/worker) |
| T-1205 | [FE] Bottom sheet Print: selecție dimensiune, preview layout, buton Download | L | FE | ⬜ |
| T-1205-W ✅ | [FE Web] Panel print în /labels/[id] — selecție dimensiune (5 preseturi), trigger job, SSE streaming status în timp real (fetch + ReadableStream, nu EventSource), download link la ready, reprint | L | FE | ✅ Done |
| T-1206 | [FE] Download PDF din Flutter (open_file sau share_plus) | S | FE | ⬜ |
| T-1207 | [BE] Reprint endpoint — reutilizare PDF existent | S | BE | ✅ |
| T-1208 | [QA] Teste: toate dimensiunile, font size validator, reprint, URL expirat | M | QA | ⬜ |

---

### US-13 — Print direct la imprimantă Zebra / Brother

> Ca utilizator enterprise cu imprimantă de etichete, vreau să pot trimite eticheta direct la imprimantă.

**Criterii de acceptare:**
- ✓ Selecție imprimantă din lista disponibilă pe rețeaua locală
- ✓ Suport Zebra ZPL: generare ZPL server-side, trimitere via print gateway local
- ✓ Status job print: pending / sent / confirmed / error
- ✓ Print gateway: microserviciu Go lightweight instalat pe PC-ul din depozit
- ✓ Fallback: dacă gateway offline, oferă download PDF manual

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-1301 ✅ | [BE] ZPL generator Go — ZPL II, 203/300 DPI, transliterare română | XL | BE | ✅ Done (print-svc/internal/zpl) |
| T-1302 | [BE] Print Gateway — microserviciu Go Windows/macOS: primește job, trimite la imprimantă | XL | BE | ⬜ |
| T-1303 | [BE] WebSocket canal Go — comunicare real-time app ↔ gateway | L | BE | ✅ |
| T-1304 ✅ | [BE] POST /print/jobs — creare job print, DB + Asynq worker | M | BE | ✅ Done (print-svc/internal/service + repo) |
| T-1305 | [FE] Ecran Print Direct: selector imprimantă, status real-time, fallback PDF | L | FE | ⬜ |
| T-1306 | [QA] Teste: ZPL pe Zebra real, PDF pe Brother, gateway offline | L | QA | ⬜ |

---

## EP-06 · Web Dashboard & Admin

### US-14 — Dashboard web — vizualizare și gestionare etichete

> Ca manager / administrator, vreau să pot vedea și gestiona toate etichetele procesate de echipa mea, dintr-un browser.

**Criterii de acceptare:**
- ✓ Tabel etichete cu: thumbnail, denumire produs, dată, status, operator, compliance score
- ✓ Filtrare: după dată, status, operator, categorie
- ✓ Search full-text în denumiri produse
- ✓ Preview etichetă în modal fără descărcare
- ✓ Reprint, download PDF, ștergere direct din dashboard
- ✓ Export CSV al istoricului (pentru audit ANPC)

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-1401 🔄 | [FE Web] Next.js pagină /labels — tabel cu filtre, search, paginare | XL | FE | 🔄 Partial — tabel + filtru status + paginare + link la detaliu implementate; lipsă: filtru dată, filtru operator, search full-text input, thumbnail imagine |
| T-1402 ✅ | [BE] GET /labels — endpoint paginated cu filtre multiple, full-text search PG | M | BE | ✅ Done |
| T-1403 | [FE Web] Modal preview etichetă — render câmpuri + imagine originală | M | FE | ⬜ (pagină dedicată /labels/[id] în loc de modal; imagine originală lipsă — necesită backend change) |
| T-1404 | [BE] GET /labels/export?format=csv — generare CSV cu toate etichetele workspace | M | BE | ✅ |
| T-1405 ✅ | [FE Web] Buton Export CSV (admin only) | S | FE | ✅ Done — fetch cu Bearer + download automat; date-range picker lipsă |
| T-1406 | [QA] Teste: filtre combinate, search, export CSV, preview, reprint din web | M | QA | ⬜ |

---

### US-15 — Multi-user: invitare și roluri

> Ca administrator cont, vreau să pot invita membri ai echipei și le pot atribui roluri cu permisiuni diferite.

**Criterii de acceptare:**
- ✓ Roluri: Admin (toate), Operator (procesare + print, fără billing/config), Viewer (readonly)
- ✓ Invitare prin email — link de invitație valid 48h
- ✓ Dashboard web: lista membri, rol, data adăugare, posibilitate revocare
- ✓ Limită membri per plan: Starter=1, Business=5, Enterprise=nelimitat
- ✓ Middleware Go verifică rolul la fiecare request protejat

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-1501 ✅ | [BE] Schema DB workspace_members | S | BE | ✅ Done |
| T-1502 ✅ | [BE] POST /workspace/invite — creare token invitație, email cu link | M | BE | ✅ Done |
| T-1503 ✅ | [BE] GET /workspace/invite/:token — validare + acceptare invitație | S | BE | ✅ Done |
| T-1504 | [BE] RBAC middleware Go — decorator pe route-uri cu cerință de rol minim | M | BE | ✅ |
| T-1505 ✅ | [FE Web] Pagină /workspace — secțiune membri: lista cu rol+dată, invite form (email + rol dropdown), buton revocare cu AlertDialog confirmare | L | FE | ✅ Done |
| T-1505-I ✅ | [FE Web] Pagină /invite/[token] — acceptare invitație: loading / success (rol afișat) / error (expirat/invalid), redirecționare la dashboard | S | FE | ✅ Done |
| T-1506 ✅ | [BE] DELETE /workspace/members/:id — revocare acces, invalidare sesiuni | S | BE | ✅ Done |
| T-1507 | [QA] Teste: invite flow, expirare link, RBAC per rol, revocare, limită plan | L | QA | ⬜ |

---

### US-16 — Admin panel — configurare globală agenți și monitoring

> Ca super-admin platformă, vreau să pot gestiona configurațiile AI per tenant și monitoriza utilizarea și costurile.

**Criterii de acceptare:**
- ✓ Pagină admin: lista tenants cu configurație agenți
- ✓ Editare configurație agent per tenant: provider, model, fallback, API key
- ✓ Test connection per agent cu metrici: latență, cost estimat per apel
- ✓ Dashboard utilizare: etichete procesate / tenant, cost AI acumulat, erori
- ✓ Rate limiting per tenant: setare limite custom pentru enterprise
- ✓ Logs agenți: ultimele 100 apeluri per agent

| Task ID | Descriere | Est. | Owner |
|---|---|---|---|
| T-1601 | [FE Web] Pagină /admin/tenants — tabel tenants, search, link la config | L | FE | ⬜ (arhitectura BE nu are concept de platform superadmin; toate rutele /admin sunt workspace-scoped — funcționalitate imposibil de implementat fără schimbare BE) |
| T-1602 ✅ | [FE Web] Pagină /admin — config agenți workspace curent (provider/model/endpoint per tip), test connection cu latență, fallback config | L | FE | ✅ Done (workspace-scoped, nu multi-tenant) |
| T-1603 ✅ | [BE] GET /admin/metrics — utilizare agregată per tenant | M | BE | ✅ Done (agent-svc adminhttp + api-gateway /admin/workspaces/{id}/metrics) |
| T-1603-W ✅ | [FE Web] Strip metrici în /admin — 4 card-uri: total etichete, confirmate, eșuate, încredere medie | S | FE | ✅ Done |
| T-1604 ✅ | [BE] GET /admin/agent-logs/:tenant_id — ultimele 100 apeluri cu detalii | S | BE | ✅ Done (agent-svc adminhttp + api-gateway /admin/workspaces/{id}/agent-logs) |
| T-1604-W ✅ | [FE Web] Tab "Loguri" în /admin — tabel: tip agent, provider, model, status OK/ERR, latență, tokens, dată | S | FE | ✅ Done |
| T-1605 | [BE] PUT /admin/tenants/:id/rate-limits — configurare limite custom per tenant | M | BE | ✅ |
| T-1605-W ✅ | [FE Web] Tab "Rate Limits" în /admin — form: încărcări/minut + tipăriri/zi, buton actualizare | S | FE | ✅ Done |
| T-1606 ✅ | [BE] Cost tracker — tabel prețuri Claude/OpenAI/Ollama, calculare cost per apel | L | BE | ✅ Done (agent-svc/internal/agent/pricing.go) |
| T-1607 | [QA] Teste: access control admin, config update live, cost tracker acuratețe | M | QA | ⬜ |

---

## Sumar Total

| Epic | Nume | Stories | Tasks | Est. totală | Prio |
|---|---|---|---|---|---|
| EP-01 | Auth & Onboarding | 3 | 21 | ~15 zile | MVP |
| EP-02 | Fotografiere & Capture | 2 | 16 | ~12 zile | MVP |
| EP-03 | AI Agent Pipeline | 4 | 36 | ~25 zile | MVP |
| EP-04 | Editor & Validare | 2 | 13 | ~10 zile | MVP |
| EP-05 | Print & Export | 2 | 14 | ~14 zile | MVP+ |
| EP-06 | Web Dashboard & Admin | 3 | 19 | ~18 zile | MVP+ |
| **TOTAL** | | **16 stories** | **119 tasks** | **~94 zile** | |

> **Notă:** Cu o echipă de 3 (1 BE Go + 1 FE Flutter + 1 Full/QA), timeline MVP (EP-01→04) se comprimă la **~8-10 săptămâni** prin paralelizare.
