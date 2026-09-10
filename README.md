# Seki (席)

A production-grade concurrent reservation engine written in Go. **Seki (席)** is Japanese for "seat" or "slot" — the exact thing this system books, and the name matches its purpose.

The core engineering problem this project solves: **when many clients race to book the same resource for the same time slot at the same instant, exactly one must win and every other request must fail cleanly** — zero double-bookings, regardless of how many API instances are running or how requests interleave. This is a classic backend interview theme (seat reservations, meeting rooms, hotel bookings, inventory holds) and this repo doesn't just claim to solve it — it includes an automated test, a load test, and a click-to-run interactive demo that all prove it.

日本語の概要は本ファイル末尾の **「日本語版概要」** をご覧ください。

---

## The core guarantee, and how it's enforced

Correctness here does **not** come from application-level locking, mutexes, or "check then insert" logic in Go — all of that breaks the moment you run more than one instance of the API. Instead, it comes from PostgreSQL itself:

```sql
ALTER TABLE bookings
    ADD CONSTRAINT no_overlapping_confirmed_bookings
    EXCLUDE USING gist (
        resource_id WITH =,
        during WITH &&
    )
    WHERE (status = 'confirmed');
```

This tells Postgres: *no two confirmed rows may share a `resource_id` and have overlapping time ranges — full stop.* Every `INSERT` runs through this constraint transactionally, so it holds true no matter how many goroutines, processes, or containers are inserting concurrently. The Go code's only job is to catch the resulting `23P01` exclusion-violation error and translate it into a clean `409 Conflict`.

This is proven **three separate ways** in this repo:
- **`test/integration/booking_concurrency_test.go`** — spins up a real ephemeral Postgres (via testcontainers-go), fires 25 goroutines at the same slot simultaneously, asserts exactly 1 succeeds and 24 are rejected.
- **`test/load/booking_load_test.js`** — a k6 load test firing 50 concurrent virtual users over real HTTP against the live Docker stack.
- **The "Concurrency Demo" tab in the Streamlit UI** — click a button, watch real concurrent requests race for the same slot from your browser, see the real result. Nothing here is simulated.

## What's in the stack

| Concern | How it's handled |
|---|---|
| Concurrency safety | PostgreSQL `EXCLUDE` constraint (not app-level locking) |
| Idempotent writes | `Idempotency-Key` header, checked/stored in the same transaction as the write |
| Audit trail | Append-only, SHA-256 hash-chained log; `/api/v1/audit/verify` recomputes and verifies the whole chain |
| Auth | JWT (HS256), bcrypt password hashing, role-based access (`customer` / `admin`) |
| Metrics | Prometheus counters + latency histograms, visualized in a pre-provisioned Grafana dashboard |
| Tracing | OpenTelemetry spans exported to Jaeger — every request gets a trace ID, visible in both logs and the Jaeger UI |
| Structured logs | `slog` JSON logs, correlated with trace IDs |
| Graceful shutdown | `SIGTERM` drains in-flight requests (up to 15s) before exiting |
| Liveness vs readiness | `/status/live` (process is up) vs `/status/ready` (Postgres + Redis actually reachable) — different operational questions, different endpoints |
| Demo/ops layer | Thin Streamlit dashboard — a pure HTTP client of the real API, no mocked data, bilingual EN/日本語 UI |

## Architecture

```
cmd/api                → entrypoint, graceful shutdown, tracing/DB/Redis init
internal/
  config                → environment-based configuration
  db                     → pgx connection pool setup
  domain                  → models, validation, sentinel errors (no framework deps)
  repository               → all SQL lives here; owns transactions, locking, the audit chain
  service                    → business logic, orchestrates repositories
  handler                     → HTTP layer: decode, call service, map errors to status codes
  middleware                   → JWT auth, structured request logging, OpenTelemetry tracing
  metrics                       → Prometheus metric definitions
  tracing                        → OpenTelemetry tracer provider setup (OTLP → Jaeger)
migrations/               → versioned SQL migrations (golang-migrate)
observability/
  prometheus.yml            → Prometheus scrape config
  grafana/                    → pre-provisioned datasource + dashboard JSON
streamlit_app/               → thin demo/ops UI — a pure client of the API (see below)
test/integration/              → real-Postgres concurrency + idempotency proofs
test/load/                       → k6 load test against the live HTTP API
```

Layering is deliberate: `handler` never touches SQL, `repository` never touches HTTP, and `domain` depends on nothing else in the project. This is what makes the concurrency test possible at all — the repository layer can be tested against a real database with zero HTTP or auth scaffolding in the way.

## Prerequisites

- Docker and Docker Compose
- (Optional, for running tests locally outside Docker) Go 1.22+, [golang-migrate CLI](https://github.com/golang-migrate/migrate), [k6](https://k6.io/docs/get-started/installation/)

## Run it

```bash
unzip seki.zip
cd seki
make up
```

That single command builds the API image and the Streamlit UI image, starts Postgres, Redis, and Jaeger, **runs all database migrations automatically** (a one-shot `migrate` container the API waits on before it starts), then boots the API, Prometheus, and Grafana. It finishes by confirming the API is actually reachable.

```
Seki API is up.

  API:         http://localhost:8080/api/v1
  Demo UI:     http://localhost:8501
  Grafana:     http://localhost:3000  (admin/admin)
  Prometheus:  http://localhost:9090
  Jaeger:      http://localhost:16686
```

**Confirm Postgres is actually connected** — this isn't a fake healthcheck, it actively pings the pgx pool and Redis client on every call:

```bash
curl http://localhost:8080/status/ready
# {"postgres":"connected","redis":"connected"}
```

Seed demo data (an admin account + two bookable meeting rooms):

```bash
make seed
```

| Email | Password | Role |
|---|---|---|
| `admin@seki.dev` | `AdminPass123` | admin |

## See it running — the demo UI

Open **http://localhost:8501**.

1. Switch the sidebar language to 日本語 or English — the whole UI relabels.
2. Register a new account, or log in as `admin@seki.dev`.
3. **"Book a Resource"** tab — pick a room, pick a time, book it. Try booking the exact same slot again and watch it get a clean, expected conflict.
4. **"Concurrency Demo"** tab — pick a resource, choose how many concurrent requests to fire (2–30), click the button. Watch real HTTP requests race for the same far-future slot from your own browser: exactly one succeeds, the rest are correctly rejected. This is the single most convincing thing in the whole project to show someone live.
5. **"Audit Trail"** tab — paste a booking ID to see its hash-chained history, or click "Verify entire audit chain integrity" to confirm nothing has been tampered with.
6. **"System Status"** tab — live `/status/live` and `/status/ready` output, live booking counters pulled straight from `/metrics`, and links to Jaeger/Prometheus/Grafana.

This is the fastest way to show a recruiter or hiring manager the whole system working without touching a terminal — and unlike a typical portfolio demo, every button here is a real request against a real backend enforcing a real database constraint.

## Test it from the command line instead

Register a user and grab a token:

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"AlicePass123"}' \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
```

List bookable resources:

```bash
curl -s http://localhost:8080/api/v1/resources | python3 -m json.tool
```

Create a booking (with an idempotency key — safe to retry):

```bash
RESOURCE_ID="<paste-a-resource-id-here>"

curl -s -X POST http://localhost:8080/api/v1/bookings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: demo-key-001" \
  -d "{\"resource_id\":\"$RESOURCE_ID\",\"start_time\":\"2027-01-15T10:00:00Z\",\"end_time\":\"2027-01-15T11:00:00Z\",\"notes\":\"team sync\"}"
```

Book the **exact same slot again** — watch it correctly fail with `409`:

```bash
curl -s -o /dev/null -w "%{http_code}\n" -X POST http://localhost:8080/api/v1/bookings \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: demo-key-002" \
  -d "{\"resource_id\":\"$RESOURCE_ID\",\"start_time\":\"2027-01-15T10:00:00Z\",\"end_time\":\"2027-01-15T11:00:00Z\"}"
# 409
```

View your bookings, check availability, verify the audit chain:

```bash
curl -s http://localhost:8080/api/v1/bookings/mine -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
curl -s "http://localhost:8080/api/v1/resources/$RESOURCE_ID/availability?from=2027-01-01T00:00:00Z&to=2027-02-01T00:00:00Z" | python3 -m json.tool
curl -s http://localhost:8080/api/v1/audit/verify | python3 -m json.tool
# {"valid": true}
```

## Prove the concurrency guarantee yourself

**Option A — integration test against real Postgres (fastest, no HTTP layer):**

```bash
go test ./test/integration/... -v -timeout 5m
```

(Requires Docker running locally and the `migrate` CLI installed — `brew install golang-migrate` or see the [releases page](https://github.com/golang-migrate/migrate/releases).)

**Option B — full load test over HTTP against the live stack:**

```bash
make up
make seed
make load-test
```

**Option C — the Streamlit "Concurrency Demo" tab** described above — the most visual option, good for live demos.

## Observability walkthrough

- **Metrics** — `curl http://localhost:8080/metrics` for raw Prometheus text, or open **http://localhost:9090** to query directly, or **http://localhost:3000** (admin/admin) for the pre-built Grafana dashboard (booking throughput, conflict rate, p50/p95/p99 latency by route).
- **Traces** — every request gets an OpenTelemetry span, exported to Jaeger. Open **http://localhost:16686**, select service `seki-api`, and find traces — including ones from the load test or the concurrency demo, so you can see individual requests racing for the same slot on a timeline.
- **Logs** — `docker compose logs -f api` shows structured JSON logs; each line includes a `trace_id` field that matches the Jaeger trace ID, so logs and traces correlate directly.

## Run everything locally without Docker (for active development)

```bash
docker compose up -d postgres redis jaeger

export DATABASE_URL=postgres://seki:seki@localhost:5432/seki?sslmode=disable
migrate -path migrations -database "$DATABASE_URL" up

cp .env.example .env
go run ./cmd/api

# in another terminal, for the UI:
cd streamlit_app
pip install -r requirements.txt
API_BASE_URL=http://localhost:8080/api/v1 streamlit run app.py
```

## Why these design choices

- **PostgreSQL exclusion constraints over application-level locking.** Distributed locks (Redis, etcd) add an entire extra failure mode and don't survive a lock-holder crashing mid-request. A database constraint is enforced by the one component that's already the source of truth, and it's atomic by construction.
- **Idempotency keys stored in the same transaction as the write.** Checking idempotency outside the transaction opens a window where a crash between the two steps corrupts the guarantee. Doing both in one Postgres transaction means it's all-or-nothing.
- **Endpoints named for what they check, not generically.** `/status/live` and `/status/ready` are two different operational questions with two different correct responses from an orchestrator (restart vs. pull from rotation) — collapsing them into one generic `/health` hides that distinction.
- **The Streamlit layer is a client, not a component.** It contains zero booking logic — it can be deleted entirely and the backend is unaffected. This is deliberate: the UI exists to make the system's correctness *visible*, not to be part of what makes it correct.
- **Three observability pillars, not just metrics.** Metrics alone tell you *that* something is wrong; traces tell you *where* in a request's path; logs tell you *what exactly* happened at that point. Having all three, correlated by trace ID, is what "observable" actually means in production.
- **No ORM.** Raw SQL via `pgx` keeps the exclusion-constraint error handling explicit and visible — an ORM would either hide this error entirely or require raw-SQL escape hatches anyway.

---

## 日本語版概要

**Seki（席）** は、Go言語で実装された本番運用を想定した「予約の同時実行制御」を証明するバックエンドシステムです。名前は日本語の「席」（座席・枠）に由来し、このシステムが実際に予約する対象そのものを表しています。

### 解決している技術課題

**同じリソース・同じ時間枠に対して複数のクライアントが同時に予約リクエストを送った場合、必ず1件だけが成功し、他はすべて正しく拒否されなければならない**（二重予約ゼロ）——これは会議室予約、座席予約、ホテル予約、在庫確保など、日本国内外を問わずバックエンド面接で頻出するテーマです。本プロジェクトは「解決した」と主張するだけでなく、自動テスト・負荷テスト・ブラウザから実行できるデモの3通りで実際に証明します。

### 保証の実現方法

アプリケーション側のロック（mutexや「確認してから挿入」処理）には一切依存していません。これらは複数インスタンス運用の時点で破綻するためです。代わりに **PostgreSQLの排他制約（EXCLUDE制約）** を利用しています：

```sql
EXCLUDE USING gist (resource_id WITH =, during WITH &&) WHERE (status = 'confirmed')
```

これにより、確定済み（confirmed）の予約は「同じリソース」かつ「時間帯が重なる」組み合わせを一切許さないことをデータベース自身が保証します。Goのコードは、この制約違反エラー（`23P01`）を検知して `409 Conflict` に変換するだけです。

### 主な構成要素

- **Go + chi** によるレイヤードアーキテクチャ（handler → service → repository → domain）
- **PostgreSQL** の排他制約による同時実行の正当性保証
- **冪等性キー**（Idempotency-Key）— リトライ時の重複予約を防止
- **ハッシュチェーン監査ログ** — すべての予約作成・キャンセルを改ざん検知可能な形で記録
- **JWT認証**、bcryptによるパスワードハッシュ化、ロールベースアクセス制御
- **Prometheus + Grafana** によるメトリクス可視化
- **OpenTelemetry + Jaeger** による分散トレーシング
- **薄いStreamlit製デモUI**（日本語/English切り替え対応）— 実際のAPIを呼び出すだけのクライアントで、業務ロジックは一切含みません

### 実行方法

```bash
unzip seki.zip
cd seki
make up
```

起動後、以下にアクセスできます：

- デモUI: http://localhost:8501 （サイドバーで「日本語」を選択可能）
- API: http://localhost:8080/api/v1
- Grafana: http://localhost:3000 （admin/admin）
- Jaeger: http://localhost:16686

デモUIの「同時予約デモ」タブでは、同一の時間枠に対して複数の同時リクエストを実際に送信し、1件のみが成功し残りが正しく拒否される様子をその場で確認できます。

初期管理者アカウント（`make seed` 実行後）：

| メールアドレス | パスワード | 権限 |
|---|---|---|
| `admin@seki.dev` | `AdminPass123` | 管理者 |
# Seki-Reservation-System
