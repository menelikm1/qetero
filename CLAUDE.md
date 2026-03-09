# Qetero — Claude Context

## Claude Preferences
- **Never add `Co-Authored-By: Claude` to commit messages.**

Ethiopia's P2P rental marketplace for heavy machinery and construction equipment. Think Turo for excavators, cranes, concrete mixers — built for the Ethiopian market. Private project.

## Stack
- **Go 1.23 + Gin** — REST API
- **PostgreSQL** + golang-migrate (7 migrations)
- **JWT auth** — bcrypt cost 12, short-lived access tokens
- **Telegram bot** (`@qetero_ethiopia_bot`) — primary user-facing interface, calls repo layer directly (not HTTP)
- **Currency:** ETB (Ethiopian Birr)
- **Deployment target:** Railway or Fly.io

## Project Structure
```
main.go
internal/
  auth/jwt.go
  config/config.go
  db/db.go
  handlers/         auth.go, bookings.go, listings.go, users.go
  middleware/       auth.go
  models/           user.go, listing.go, booking.go
  repository/       user_repo.go, listing_repo.go, booking_repo.go
  telegram/         bot.go, commands.go, session.go
migrations/         001–007
DESIGN.md           full platform design doc — read this for deep context
README.md
```

## Data Models

**User:** id(UUID), name, phone(required, unique, VARCHAR(50)), email(optional, unique), password_hash, role(owner|renter|both), telegram_chat_id(BIGINT), verified

**Listing:** id, owner_id, title, category(enum), description, location, price_per_day(ETB), minimum_days, images(TEXT[]), specs(JSONB), is_available, deleted_at (soft delete)

**Booking:** id, listing_id, renter_id, owner_id, start_date, end_date, total_days, total_price, status(pending|confirmed|active|completed|cancelled), payment_method, cancellation_reason

**Payment** (migration 004 exists, no handlers yet): amount, platform_fee, owner_payout, status, provider, provider_ref

**Review** (migration 005 exists, no handlers yet): booking_id, reviewer_id, reviewee_id, listing_id, rating(1–5), comment

**Equipment categories:** excavator, crane, scaffold, compactor, loader, forklift, generator, water_truck, concrete_mixer, dump_truck, dozer, roller, other

## API Routes
```
Public:
  POST   /v1/auth/register
  POST   /v1/auth/login
  GET    /v1/listings
  GET    /v1/listings/:id
  GET    /v1/listings/:id/availability

Protected (JWT):
  POST   /v1/listings
  PUT    /v1/listings/:id
  DELETE /v1/listings/:id
  POST   /v1/bookings
  GET    /v1/bookings/:id
  PUT    /v1/bookings/:id/confirm
  PUT    /v1/bookings/:id/cancel
  GET    /v1/users/me
  PUT    /v1/users/me
  GET    /v1/users/me/bookings
  GET    /v1/users/me/listings/bookings
```

## Telegram Bot
Commands: `/start`, `/help`, `/link [phone]`, `/browse`, `/search [category] [location]`, `/listing [number]`, `/book [number] [start] [end]`, `/mybookings`

Session store: in-memory `map[chatID]Session` — stores `LastListings []uuid.UUID` so users reference listings by number (1, 2, 3...) instead of raw UUIDs. Bot calls repo layer directly, not via HTTP.

## Key Design Decisions
- **Phone is primary identity** (not email) — suits Ethiopian market where many users lack active email
- **Telegram first** — Telegram is dominant in Ethiopia, web frontend deferred to Phase 2
- **Soft deletes** on listings via `deleted_at` column
- **Conflict check** (`HasConflict`) only blocks `confirmed` and `active` bookings — `pending` and `cancelled` don't block dates
- **Manual payments MVP** — renters pay owners directly via Telebirr, CBE, cash; platform not in the money flow yet
- **Revenue MVP:** Owner subscription 500 ETB/month (manual collection)
- **Revenue Phase 2:** 15% platform fee per booking via Chapa, auto-split to owner

## Ethiopian Market Context
- Stripe does not operate in Ethiopia — Chapa is the payment processor
- Telebirr (Ethio Telecom mobile money) has massive adoption — must support
- Africa's Talking is the SMS provider for OTP
- Amharic support planned for bot in Phase 2
- All prices in ETB

## Phase Status

### Phase 1 — MVP (branch: feature/telegram-bot)
- [x] Auth (register, login, JWT)
- [x] Listings CRUD + availability
- [x] Bookings (create, confirm, cancel)
- [x] Telegram bot (browse, search, book, mybookings, link)
- [ ] Owner subscription billing (manual, 500 ETB/month)

### Phase 2 — Growth
- [ ] Chapa payment integration (Telebirr, CBE Birr, bank transfer)
- [ ] 15% platform fee, auto-split payout to owner
- [ ] Africa's Talking OTP auth (replace password)
- [ ] Reviews & ratings (DB schema exists, handlers not built)
- [ ] SMS/Telegram notifications on booking events
- [ ] Image uploads via Cloudflare R2
- [ ] Web frontend (Next.js or SvelteKit)
- [ ] Amharic language support in bot

### Phase 3 — Scale
- [ ] Mobile app
- [ ] Insurance / damage protection
- [ ] Fleet management for large owners
- [ ] Analytics dashboard
- [ ] East Africa expansion

## Environment Variables
```
PORT=8080
DATABASE_URL=postgres://postgres:password@localhost:5432/qetero?sslmode=disable
JWT_SECRET=...
TELEGRAM_BOT_TOKEN=...

# Phase 2:
# CHAPA_SECRET_KEY=
# AFRICASTALKING_API_KEY=
# AFRICASTALKING_USERNAME=
# R2_ACCOUNT_ID=
# R2_ACCESS_KEY=
# R2_SECRET_KEY=
# R2_BUCKET=
```

## Security (from DESIGN.md)
- [x] JWT, bcrypt, input validation, parameterized queries only, soft deletes
- [ ] Rate limiting (100 req/min global, 10 req/min auth)
- [ ] Owner contact hidden until booking confirmed
- [ ] CORS restricted to known origins
