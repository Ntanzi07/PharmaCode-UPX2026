# PharmaCode

A UPX project (Facens, 2026). Point your phone camera at the barcode on a medicine box and the app shows a **simplified leaflet** (*bula simplificada*): what the drug is for, how to take it, what to do if you miss a dose, precautions and side effects, in plain language.

The data comes from Anvisa's official sources. Each summary is written by the team and **reviewed by a pharmacist** before it shows up in the app.

The repository has three parts:

| Part | Stack | What it does |
|---|---|---|
| API | Go (`net/http`, sqlc, pgx) + PostgreSQL | Stores drugs, packages and leaflets. The only public route is the EAN lookup used by the app. |
| Admin panel (`web/`) | React + Vite | Create, edit and review leaflets, behind a login. |
| Swagger | swaggo | API docs at `/docs`. |

## Getting started

Requirements: Docker (with Compose) and Go 1.26 (only to run outside Docker or to generate code).

```bash
cp .env.example .env
```

In `.env`, set the first admin. It is only created when the database has no users yet:

```env
ADMIN_NAME=Your Name
ADMIN_EMAIL=you@example.com
ADMIN_PASSWORD=a-strong-password
```

Then start everything:

```bash
docker compose up --build -d
```

The `migrate` container applies the migrations automatically.

| Service | URL |
|---|---|
| Admin panel | http://localhost:3000 |
| API | http://localhost:8080 |
| Swagger | http://localhost:8080/docs |
| Postgres | `localhost:5433` (user, password and database in `.env`) |

Log in to the panel with the admin from `.env` and change the password in the user menu.

### Development without Docker

```bash
docker compose up -d db migrate       # database and migrations only
go run ./cmd/api                      # API on :8080 (reads .env)
cd web && npm install && npm run dev  # panel on :5173, proxying to the API
```

### Useful commands

```bash
go test ./...                         # API tests
sqlc generate                         # after changing db/queries or db/migrations
swag init -g cmd/api/main.go -o docs --overridesFile .swaggo --parseDependency --useStructName
```

### Example data (seed)

`db/seed/advil_12h.sql` fills the database with Advil 12h: the drug, its three
presentations with their EANs and a simplified leaflet. Handy for a demo.

```bash
docker compose exec -T db psql -U bula -d bula < db/seed/advil_12h.sql
```

The leaflet comes in as not reviewed on purpose: review it in the panel to see it
show up in `GET /drugs/ean/{ean}`.

### Spreadsheet import

The panel has an **Importar** tab (editor and up) to register many drugs at once:

1. Download the template (`GET /imports/template`): an `.xlsx` with the sheets
   `remedios` (drug + optional leaflet), `embalagens` (presentation + up to three
   EANs) and `instrucoes` (what each column means).
2. Upload the filled file and press **Conferir** (`POST /imports/preview`): the API
   validates every row and runs the whole import inside a transaction it rolls back,
   so the counts are real and nothing is written.
3. Press **Importar** (`POST /imports/apply`) to save it, all in one transaction.

Rows are matched by registration number, presentation registration and EAN, so
importing the same file again updates instead of duplicating. Imported leaflets
come in as not reviewed, and an EAN that already belongs to another package is
reported as an error instead of being moved.

## Permissions

| Role | Can |
|---|---|
| no login (the app) | `GET /drugs/ean/{ean}`, which only returns reviewed leaflets |
| `editor` | read and create/update/delete drugs, packages and leaflets |
| `reviewer` (pharmacist) | everything an editor can + mark leaflets as reviewed |
| `admin` | everything a reviewer can + manage users |

Editing the text of a reviewed leaflet removes the review: the leaflet leaves the app until it is reviewed again.

## Where the data comes from

Registering a drug uses two official Anvisa sources.

### 1. Bulário Eletrônico: the drug and its leaflet

**https://consultas.anvisa.gov.br/#/bulario/**

Search by the drug name. The product detail page fills in the **Remédios** (drugs) screen of the panel, and the leaflet PDF is the basis for the summary in the **Bulas** (leaflets) screen.

| Field in the Bulário | In the API | Example (Advil 12h) |
|---|---|---|
| Nome do Produto | `drugs.brand_name` | Advil 12h |
| Número da Regularização (9 digits) | `drugs.registration_number` | 192900007 |
| Princípio Ativo | `drugs.active_ingredient` | ibuprofeno |
| Empresa Detentora da Regularização | `drugs.manufacturer` | PF Consumer Healthcare Brazil… |
| Bulário Eletrônico → leaflet link (PDF) | `summaries.source_url` | PDF link |
| Leaflet *expediente* (filing number) | `summaries.leaflet_expedient` | filing number |
| Leaflet publication date | `summaries.leaflet_published_at` | YYYY-MM-DD |

The filing number and the date tell **which version of the leaflet** was summarized. When Anvisa publishes a new leaflet, you can see which summaries are out of date.

The text fields of the simplified leaflet (`what_is_it_for`, `posology`, `missed_dose`, `warnings`, `contraindications`, `adverse_effects`, etc.) are written by the team from the sections of the PDF.

Bulário fields we **don't** store: Número do Processo, CNPJ, regularization dates, regulatory category, therapeutic class.

### 2. CMED price list: packages and EANs

**https://www.gov.br/anvisa/pt-br/assuntos/medicamentos/cmed/precos**

Download the price spreadsheet (`.xlsx`) and filter by the product. Each row is a **presentation** (one specific package) and becomes a record in the **Embalagens** (packages) screen.

| xlsx column | In the API | Example |
|---|---|---|
| REGISTRO (13 digits) | `packages.presentation_registration` | 1929000070034 |
| EAN 1, EAN 2, EAN 3 | `package_eans.ean` (one record per filled EAN; `-` means empty) | 7896015592752 |
| APRESENTAÇÃO | `packages.description` | 600 MG COM REV LIB PRO… |
| PRODUTO / SUBSTÂNCIA / LABORATÓRIO | cross-check with the drug (`brand_name`, `active_ingredient`, `manufacturer`) | ADVIL 12H / IBUPROFENO |

**How CMED links to the Bulário:** the first 9 digits of the presentation's REGISTRO are the drug's Número da Regularização.

```
1929000070034  ← REGISTRO in CMED (presentation)
192900007      ← Número da Regularização in the Bulário (drug)
```

In the example, Advil 12h has three presentations in CMED (…0034, …0069, …0107). Each one is a package with its own EAN.

**Why a package can have several EANs:** the same box sometimes circulates with more than one barcode, for example the old code and the new one on the shelf at the same time. That's why the spreadsheet has the EAN 1, 2 and 3 columns. Register all of them: the app finds the leaflet with any of them.

CMED columns we **don't** store: CNPJ, CÓDIGO GGREM, therapeutic class, product type and the price columns.

### Data model in short

```
drugs (Bulário: one drug, 9-digit registration)
 ├── packages (CMED: one per presentation, 13-digit registration)
 │    └── package_eans (CMED EAN 1..3; what the app reads from the box)
 └── summaries (simplified leaflet, written from the Bulário PDF)
```
