# Sloan Marketplace

A peer-to-peer marketplace for MIT Sloan students, delivered as a phone-first
web app that works alongside the WhatsApp groups students already use. This
sprint covers **event tickets only**.

Sellers post through the web app or by messaging a WhatsApp bot. Buyers
browse, search and filter listings and tap through to the seller's WhatsApp.
Only people who can receive email at an `mit.edu` address can sign up.

## Stack

- **Backend:** Go 1.26, Echo, pgx, clean architecture
- **Frontend:** React + TypeScript via Vite, phone-first CSS (installable as a home-screen web app)
- **Database:** Postgres 18
- **Email codes:** any free SMTP relay (Gmail app password, Brevo free tier); printed to the log in dev
- **WhatsApp bot:** WhatsApp Business Cloud API (Meta, free test number) for direct chats; `whatsmeow` for the group-chat pilot
- **Local infra:** Docker Compose, Makefile

## Quick start

```sh
make up        # builds images, starts Postgres, backend and frontend
open http://localhost:5174
```

For faster iteration, run the database in Docker and the apps natively:

```sh
make env       # creates .env from .env.example
make install   # once
make dev       # Postgres in Docker, Go + Vite with hot reload
```

With no SMTP configured, sign-in codes are printed to the backend log
(`make logs`, or the terminal running `make dev`). The Vite dev server
listens on your LAN too, so you can open `http://<your-mac-ip>:5174` on a phone.

`make help` lists every target.

## Layout

```
backend/
  cmd/server/            entrypoint; wires the layers together
  internal/domain/       entities and repository interfaces (no dependencies)
  internal/usecase/      business rules: auth codes, listings, WhatsApp bot grammar
  internal/adapter/
    http/                Echo handlers, routing, WhatsApp webhook
    postgres/            pgx implementations of the repositories
    email/               SMTP and log mailers for one-time codes
    whatsapp/            Cloud API client for bot replies
  internal/token/        JWT session and signup tokens
frontend/                Vite React app
  db/migrations/         SQL schema, embedded and applied on start-up
docker-compose.db.yml    Postgres only
docker-compose.yml       backend + frontend (includes the db file)
```

Dependencies point inward: adapters import use cases and domain, use cases
import domain, domain imports nothing from the project.

The backend applies `backend/db/migrations/*.sql` every time it starts,
under an advisory lock, so a fresh database gets its schema with no manual
step. Each file must be idempotent (`CREATE TABLE IF NOT EXISTS`, ...);
to change existing tables, add a new numbered file rather than editing one.

## Sign-up and sign-in

One flow for both, with no passwords:

1. `POST /api/auth/request-code {"email"}` emails a 6-digit code. Only
   `mit.edu` and its subdomains (`sloan.mit.edu`, `alum.mit.edu`) are accepted.
2. `POST /api/auth/verify-code {"email", "code"}` returns `{"session"}` for an
   existing account, or `{"signup_token"}` for a new one.
3. New users call `POST /api/auth/complete-signup {"signup_token", "name", "whatsapp_phone"}`.

Codes expire after 10 minutes, are single use, lock after 5 wrong guesses,
and are stored only as an HMAC. One code per address per 30 seconds, plus a
per-IP limit. Sessions are HS256 JWTs valid for 30 days.

## API

All `/api/listings` routes need `Authorization: Bearer <token>`; listings
carry sellers' phone numbers, so they're members-only.

| Method | Path                         | Notes |
|--------|------------------------------|-------|
| GET    | `/healthz`                   | |
| POST   | `/api/auth/request-code`     | see above |
| POST   | `/api/auth/verify-code`      | see above |
| POST   | `/api/auth/complete-signup`  | see above |
| GET    | `/api/auth/me`               | |
| GET    | `/api/listings`              | active, upcoming. Query: `q`, `date_from`, `date_to` (YYYY-MM-DD), `min_price`, `max_price` (dollars), `min_qty`, `sort` (`soonest`, `newest`, `price_low`, `price_high`), `limit` |
| POST   | `/api/listings`              | `{"event_name", "event_date", "venue", "quantity", "price", "notes"}`, price in dollars per ticket |
| GET    | `/api/listings/mine`         | the caller's listings, any status |
| GET    | `/api/listings/:id`          | |
| PATCH  | `/api/listings/:id/status`   | `{"status": "active" \| "sold" \| "withdrawn"}`, own listings only |
| GET    | `/webhooks/whatsapp`         | Meta subscription handshake |
| POST   | `/webhooks/whatsapp`         | inbound WhatsApp messages |
| POST   | `/api/dev/whatsapp`          | `{"from", "text"}` → `{"reply"}`; bot simulator, only when `APP_ENV=dev` |

## WhatsApp bot

The sender's WhatsApp number identifies the account, which is why each
account's WhatsApp number is unique. The bot runs over two connections:

- **Direct chat** (official Cloud API): sellers message the bot's number.
- **Group chat** (unofficial, pilot): the bot's number is added to a Sloan
  group and answers messages starting with `/marketplace` (or `/mp`).
  Everything else in the group is ignored.

Listings can be written casually; the bot pulls out the date, price and
quantity and treats the rest as the event name:

```
/marketplace post BRUINS TICKETS 80 DOLLARS fri
/marketplace post 2 tix to the Sloan Gala 11/14 $40 each
sell Bruins vs Rangers Sat 10/10 $80 each, 2 tickets
```

Fields also work, separated by pipes:

```
sell Sloan Fall Ball | Nov 14 | $40 | 2 | Boston Harbor Hotel | notes
```

or the long form, one field per line in any order:

```
sell
Event: Sloan Fall Ball
Date: Nov 14
Price: $40
Qty: 2
```

`mine` lists your active listings, `sold 12` marks #12 sold, `remove 12`
withdraws it, anything else replies with help. Dates accept `Nov 14`,
`11/14`, `2026-11-14`, `Sat Nov 14th`, `tomorrow` and similar.

Try it locally without Meta (prices without `$` here, since make expands it):

```sh
make bot FROM=16175550123 TEXT="sell Sloan Fall Ball | Nov 14 | 40 | 2"
```

### Group bot pilot (unofficial)

Meta's official API can only post to groups the business creates, capped at
8 members, so the group bot links to an ordinary WhatsApp account the way
WhatsApp Web does (via the `whatsmeow` library). This is against WhatsApp's
terms of service and the number can be banned, so use a dedicated SIM and
keep the Cloud API bot as the fallback.

1. Put the bot's number in `.env` as `WHATSAPP_WEB_PHONE` and run
   `make db-up && make wa-link`. It prints an 8-character code.
2. On the bot's phone: WhatsApp → Settings → Linked devices → Link a device →
   "Link with phone number instead", and enter the code within two minutes.
3. Set `WHATSAPP_WEB_ENABLED=true` and restart the backend. The session is
   stored in Postgres (`whatsmeow_*` tables), so only one backend may run it.
4. Have a group admin add the number to the group.

WhatsApp is rolling out hidden phone numbers in groups. When the bot can't
see a member's number it asks them to send it one direct message, after
which it recognises them in groups too.

### Connecting a real WhatsApp number (Cloud API)

1. Create an app at developers.facebook.com, add the WhatsApp product, and
   note the test **phone number ID** and a **temporary access token** from
   *API Setup*. Add your own phone as a test recipient.
2. Expose the backend publicly (deployed, or `cloudflared tunnel --url http://localhost:8081` locally).
3. Under *Configuration → Webhook*, set the callback URL to
   `https://<host>/webhooks/whatsapp` and a verify token of your choosing,
   then subscribe to the `messages` field.
4. Set `WHATSAPP_VERIFY_TOKEN`, `WHATSAPP_APP_SECRET` (*App settings → Basic*),
   `WHATSAPP_PHONE_NUMBER_ID` and `WHATSAPP_ACCESS_TOKEN` in `.env`, and
   `VITE_BOT_NUMBER` in `.env` (or `frontend/.env` for `make dev`) so the Account
   page links to the bot.

## Email

Set `SMTP_*` in `.env` (see `.env.example`). The simplest free option is a
Gmail account with 2-step verification and an app password
(`smtp.gmail.com:587`). Brevo's free plan (300 emails/day) also works.

## Deploying

Frontend on Vercel, backend and Postgres on Railway (about $5/month), both
deploying on every push to `main`.

**Railway (backend + database)**

1. New project → Deploy from GitHub repo → this repo. In the service settings
   set **Root Directory** to `backend`; `backend/railway.json` does the rest
   (Dockerfile build, one replica, `/healthz` health check).
2. Add a **Postgres** service to the project. On the backend service add a
   variable `DATABASE_URL` = `${{Postgres.DATABASE_URL}}`.
3. Set the remaining variables on the backend service: `APP_ENV=prod`,
   `JWT_SECRET` (`openssl rand -base64 48`), `APP_URL` and `CORS_ORIGIN`
   (your Vercel URL, no trailing slash), the `SMTP_*` values, and
   `WHATSAPP_WEB_ENABLED=true` if the group bot should run there.
4. Generate a public domain under *Settings → Networking*. That's the API URL.

To move the WhatsApp link to the hosted database, run
`DATABASE_URL='postgres://…' make wa-link` against it once (unlink the
local one first on the phone), and keep the local backend's
`WHATSAPP_WEB_ENABLED=false` so the two don't fight over the session.

**Vercel (frontend)**

1. New project → import this repo, **Root Directory** `frontend`. Vercel
   detects Vite; `frontend/vercel.json` adds the SPA rewrite.
2. Environment variables: `VITE_API_URL` = the Railway API URL, and
   `VITE_BOT_NUMBER` = the bot's number in digits.
3. Deploy, then put the resulting URL into Railway's `APP_URL` and
   `CORS_ORIGIN` and redeploy the backend.

## Ports

| Service  | Port |
|----------|------|
| Frontend | 5174 |
| Backend  | 8081 |
| Postgres | 5434 |

Chosen not to collide with petter-help (5173, 8080, 5433). Override any of
these in a `.env` file (see `.env.example`).
