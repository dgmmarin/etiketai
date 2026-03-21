# EtiketAI — Brainstorming & Arhitectură

> **Fotografiază. Traduce. Tipărește.**
> Aplicație mobilă (iOS + Android) · Web Dashboard · REST API
> Draft · Martie 2026 · Confidențial

---

## 1. Context & Problemă

### Problema de piață

Orice produs importat comercializat în România trebuie să aibă eticheta tradusă în limba română, conform **OG 21/1992 (art. 20 alin. 5)** și regulamentelor UE. Nerespectarea atrage amenzi ANPC între **1.000 și 30.000 lei** per contravenție. În 2025, ANPC a intensificat controalele, aplicând sancțiuni de peste **110 milioane lei** în 30 de zile.

### Situația actuală a importatorilor

- Traduc manual etichetele — lent, costisitor, predispus la erori
- Folosesc birouri de traduceri — zile de așteptare, costuri ridicate per document
- Unii ignoră obligația legală — risc major de amenzi și retragere produse
- Nu există un tool dedicat, simplu, care să facă asta rapid de pe telefon

### Oportunitatea

O aplicație care fotografiază o etichetă în orice limbă și în câteva secunde generează varianta tradusă, conformă legal, gata de tipărit pe autocolant — **nu există pe piața românească**.

---

## 2. Segmente de Utilizatori

| Segment | Nevoie principală | Volum estimat etichete/lună |
|---|---|---|
| Importatori / distribuitori | Batch processing, istoric audit, rapoarte conformitate | 500 – 5.000+ |
| Magazine / retaileri | Viteză, simplitate, tipărire imediată | 50 – 500 |
| Producători locali | Componente importate, re-etichetare producție | 100 – 1.000 |
| Revânzători individuali | Ușurință de utilizare, cost mic, fără tehnicitate | 10 – 100 |

---

## 3. Fluxul Principal al Utilizatorului

Cei 5 pași de la produs în mână la autocolant pe raft:

### 01 — Fotografiere
Utilizatorul deschide app-ul și fotografiează eticheta originală. Camera afișează un overlay de ghidare (dreptunghi + linii de aliniere). Suport pentru import din galerie sau PDF scanat.

### 02 — Detecție AI
Imaginea este trimisă la backend. Claude Vision / GPT-4o Vision identifică: tipul produsului, limba sursă, toate câmpurile textuale. Se returnează date structurate JSON cu câmpurile extrase.

### 03 — Traducere & structurare
Câmpurile detectate sunt traduse automat în română. Sistemul aplică template-ul legal corect pe categoria de produs (aliment, cosmetic, electrocasnic etc.) și marchează câmpurile obligatorii lipsă.

### 04 — Validare utilizator
Utilizatorul vede rezultatul pe ecran, poate edita câmpuri individual înainte de confirmare. Câmpurile lipsă obligatorii sunt marcate cu roșu și blochează salvarea până la completare.

### 05 — Generare & tipărire
Eticheta finală este salvată în backend. Se generează automat fișierul pentru print (PDF dimensionat sau ZPL). Utilizatorul trimite job-ul la imprimantă din app sau din dashboard-ul web.

---

## 4. Arhitectura Tehnică

### 4.1 Stack recomandat

| Layer | Tehnologie | Justificare |
|---|---|---|
| Mobile | React Native + Expo | O singură codebase iOS + Android, acces nativ cameră |
| Web Dashboard | Next.js (React) | SSR pentru SEO, API routes integrate |
| Backend API | FastAPI (Python) | Performant, async nativ, integrare naturală cu AI |
| Baza de date | PostgreSQL | Relational robust, JSONB pentru date structurate flexibile |
| File Storage | S3 / Cloudflare R2 | Imagini originale + PDF-uri generate; R2 fără costuri egress |
| AI Vision + Traducere | Claude Vision API | Detecție + extracție + traducere într-un singur apel |
| Print Queue | BullMQ + Redis | Job-uri asincrone fiabile, retry automat, status real-time |
| Auth | JWT + Refresh tokens | Simplu, stateless, ușor de scalat; OAuth2 opțional |

> **Notă:** Documentele ulterioare (TechArch_v1) detaliază stack-ul final adoptat: **Go microservices + Flutter + Claude/Ollama**.

### 4.2 Pipeline AI — Detaliu

Fiecare fotografie trece printr-un pipeline în 3 faze: Vision → Translation → Validation.

**Exemplu output JSON de la API:**
```json
{
  "product_name": "Cremă hidratantă",
  "ingredients_ro": "Aqua, Glycerin, Niacinamide...",
  "manufacturer": "Cosme Corp Ltd",
  "country_of_origin": "Coreea de Sud",
  "quantity": "50ml",
  "warnings_ro": "A se păstra la loc uscat, ferit de căldură",
  "missing_required": ["importer_address"],
  "category": "cosmetic",
  "confidence_score": 0.94
}
```

### 4.3 Abstracție Print

Arhitectura print este construită pe un **layer de abstracție cu drivere interschimbabile**:

- **Driver Generic PDF** — coli autocolante A4 (disponibile în orice papetărie) — recomandat pentru MVP
- **Driver Zebra ZPL** — imprimante de etichete profesionale — adăugat în V2 pentru clienți enterprise

---

## 5. Funcționalități

### 5.1 MVP — Prima versiune lansabilă

| Funcționalitate | Prioritate | Notă |
|---|---|---|
| 📷 Fotografiere etichetă din app (iOS + Android) | MVP | Camera nativă cu overlay ghidare |
| 🤖 Detecție AI automată a câmpurilor | MVP | Claude Vision API |
| 🌐 Traducere automată în română | MVP | Orice limbă sursă |
| ✏️ Editor câmpuri traduse înainte de salvare | MVP | Corecție manuală |
| ⚠️ Verificator câmpuri obligatorii OG 21/1992 | MVP | Alert vizual + blocare |
| 💾 Salvare etichetă în backend + istoric | MVP | Per utilizator/firmă |
| 🖨️ Generare PDF gata de print | MVP | Generic A4 + coli autocolante |
| 🌐 Web dashboard — vizualizare + reprint | MVP | Browser, orice device |
| 🔐 Autentificare utilizator + management cont | MVP | JWT, email/parolă |
| 💳 Sistem abonamente (Stripe) | MVP | Lunar + anual |

### 5.2 Versiunea 2 — Post-lansare

- Batch processing (upload multiple etichete)
- Template-uri personalizate per firmă
- Rapoarte conformitate ANPC (export PDF)
- Multi-user (echipe)
- Driver Zebra ZPL pentru enterprise
- API public pentru integrare cu sisteme ERP

---

## 6. Model de Business

### 6.1 Planuri de abonament

| Plan | Starter | Business | Enterprise |
|---|---|---|---|
| Preț / lună | ~49 lei | ~149 lei | ~399 lei |
| Etichete / lună | 100 | 500 | Nelimitat |
| Utilizatori | 1 | 5 | Nelimitat |
| Batch processing | — | — | ✓ |
| Rapoarte conformitate | — | ✓ | ✓ |
| Driver Zebra ZPL | — | — | ✓ |
| API access | — | — | ✓ |
| Suport | Email | Email + chat | Dedicat |

> Discount 20% pentru abonament anual. Prețuri necesită validare prin interviuri cu clienți.

### 6.2 Proiecție venituri

| Scenariu | Clienți activi | ARPU / lună | MRR |
|---|---|---|---|
| Conservator (12 luni post-lansare) | 80 | 100 lei | ~8.000 lei |
| Realist (18 luni post-lansare) | 300 | 120 lei | ~36.000 lei |
| Optimist (24 luni post-lansare) | 800 | 150 lei | ~120.000 lei |

---

## 7. Roadmap de Dezvoltare

| Fază | Durată estimată | Deliverables |
|---|---|---|
| Faza 0 — Discovery | 2–3 săptămâni | 10–15 interviuri utilizatori, validare prețuri, alegere imprimantă MVP |
| Faza 1 — MVP Backend | 4–6 săptămâni | API, integrare Claude Vision, pipeline AI, PostgreSQL, S3, JWT |
| Faza 2 — MVP Mobile | 6–8 săptămâni | App Flutter iOS + Android, flux fotografiere, editor, PDF |
| Faza 3 — MVP Web | 3–4 săptămâni | Dashboard Next.js, gestionare etichete, reprint, Stripe |
| Faza 4 — Lansare Beta | 2–3 săptămâni | Testing cu 20–50 utilizatori reali, bugfixing, optimizare prompts |
| Faza 5 — V2 Features | 8–12 săptămâni | Batch, template-uri, rapoarte, multi-user, Zebra, API public |

> **Timeline total MVP:** ~17–24 săptămâni (4–6 luni) cu o echipă de 2–3 developeri.

---

## 8. Riscuri & Mitigare

| Risc | Impact | Strategie de mitigare |
|---|---|---|
| Acuratețe AI scăzută (limbi rare, imagini blur) | Mare | Editor manual obligatoriu, feedback loop, confidence score afișat |
| Modificare legislație OG 21/1992 | Mare | Monitorizare automată modificări legislative, update rapid template-uri |
| Costuri API AI ridicate la volum | Mediu | Caching rezultate, fallback DeepL pentru traduceri simple |
| Adopție lentă | Mediu | Trial gratuit 14 zile fără card, onboarding ghidat |
| Concurență (big tech) | Mic | Focus pe nișă legislativă românească — context local greu de replicat |

---

## 9. Cadrul Legal de Referință

| Act normativ | Relevanță pentru EtiketAI |
|---|---|
| OG 21/1992, art. 20 alin. 5 | Obligativitate traducere etichetă în română |
| Regulament UE 1169/2011 | Informare consumatori produse alimentare — câmpuri obligatorii |
| Regulament CE 1223/2009 | Produse cosmetice — ingrediente INCI, avertismente |
| Directiva 2011/65/UE (RoHS) | Echipamente electrice — simboluri, restricții substanțe |
| Directiva 2009/48/CE | Siguranța jucăriilor — marcaj CE, avertismente vârstă |
| Amenzi ANPC 2025 | 1.000 – 30.000 lei per contravenție; retragere produse; suspendare activitate |
