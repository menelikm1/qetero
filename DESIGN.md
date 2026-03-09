# Qetero — Platform Design Document
**Version:** 1.1
**Date:** 2026-03-08
**Market:** Ethiopia (primary)
**Status:** In Progress

---

## 1. Overview

Qetero is a peer-to-peer rental marketplace for heavy machinery and construction materials in Ethiopia. Owners list equipment they want to monetize during idle periods; renters (contractors, builders, individuals) find and book what they need without buying.

Think Turo, but for excavators, cranes, scaffolding, and concrete mixers — built for the Ethiopian market.

### MVP Goals
- API-first backend in Go
- PostgreSQL database
- Telegram bot as the first user-facing interface (Telegram is widely used in Ethiopia)
- Web frontend deferred to Phase 2

---

## 2. Tech Stack

| Layer | Choice | Reason |
|---|---|---|
| Language | Go 1.23 | Fast, simple, low overhead |
| Router | Gin | Team familiarity, great middleware + validation |
| Database | PostgreSQL | Relational, great for date-range queries and transactions |
| Migrations | golang-migrate | Version-controlled schema changes, single binary |
| Auth | Phone + OTP (Phase 2) / JWT now | Phone-first auth suits Ethiopian market |
| Password hashing | bcrypt | Industry standard |
| Telegram client | go-telegram-bot-api | Well-maintained, Telegram is dominant in Ethiopia |
| Config | godotenv + env vars | Simple, 12-factor app compliant |
| Deployment | Railway or Fly.io | Simple, cheap, no AWS complexity for MVP |
| File storage | Cloudflare R2 | S3-compatible, generous free tier |
| SMS / OTP (Phase 2) | Africa's Talking | Best SMS provider for Ethiopia |
| Payments (Phase 2) | Chapa | Ethiopian fintech — supports Telebirr, CBE Birr, bank transfer |

---

## 3. Market Context — Ethiopia

### Why these choices matter
- **Stripe does not operate in Ethiopia** — Chapa is the primary payment processor
- **Telebirr** (Ethio Telecom's mobile money) has massive adoption — must support it
- **Phone numbers are the primary identity** — many users don't actively use email
- **Telegram is heavily used** — makes it an ideal first interface
- **Amharic support** will matter for non-technical users on the bot (Phase 2)
- **Currency is ETB (Ethiopian Birr)** — all prices in ETB

### Revenue model for manual payment phase (MVP)
Since payments are offline (Telebirr, CBE bank transfer, cash), automatic fee collection
isn't possible yet. Two approaches:

| Model | How it works | When to use |
|---|---|---|
| **Owner subscription** | Owners pay flat monthly fee (e.g. 500 ETB/month) to keep listings active | MVP — simplest to collect |
| **Transaction fee** | Platform takes 15% of each booking, invoiced monthly to owner | Phase 2 — once Chapa is integrated |

MVP uses subscription model. Switch to transaction fees when Chapa goes live.

### Future automated revenue (Phase 2 with Chapa)
- Renter pays full amount through Chapa
- Platform automatically splits: 85% to owner, 15% platform fee
- Telebirr payout to owner

---

## 4. Core Domain Entities

### 4.1 User
Represents both owners and renters. A user can be both.

| Field | Type | Notes |
|---|---|---|
| id | UUID | Primary key |
| name | VARCHAR(100) | Full name |
| phone | VARCHAR(30) | Required — primary identity in Ethiopia |
| email | VARCHAR(255) | Optional, unique if provided |
| password_hash | TEXT | bcrypt — replaced by OTP in Phase 2 |
| role | ENUM | owner, renter, both |
| telegram_chat_id | BIGINT | For Telegram bot linking |
| verified | BOOLEAN | Phone verified |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

### 4.2 Listing
A piece of equipment or material available for rent.

| Field | Type | Notes |
|---|---|---|
| id | UUID | Primary key |
| owner_id | UUID | FK → users |
| title | VARCHAR(200) | e.g. "CAT 320 Excavator" |
| category | ENUM | See categories below |
| description | TEXT | |
| location | VARCHAR(255) | City/region (e.g. "Addis Ababa", "Hawassa") |
| price_per_day | DECIMAL(10,2) | In ETB |
| minimum_days | INT | Default 1 |
| images | TEXT[] | Array of URLs |
| specs | JSONB | Flexible: weight, capacity, fuel type, etc. |
| is_available | BOOLEAN | Owner can toggle off |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

**Equipment categories:**
`excavator`, `crane`, `scaffold`, `compactor`, `loader`, `forklift`, `generator`,
`water_truck`, `concrete_mixer`, `dump_truck`, `dozer`, `roller`, `other`

### 4.3 Booking
A rental transaction between a renter and an owner.

| Field | Type | Notes |
|---|---|---|
| id | UUID | Primary key |
| listing_id | UUID | FK → listings |
| renter_id | UUID | FK → users |
| owner_id | UUID | FK → users (denormalized for easy queries) |
| start_date | DATE | |
| end_date | DATE | |
| total_days | INT | Computed |
| total_price | DECIMAL(10,2) | In ETB, computed at booking time |
| status | ENUM | pending, confirmed, active, completed, cancelled |
| payment_method | VARCHAR(50) | telebirr, cbe, cash, bank_transfer (manual MVP) |
| cancellation_reason | TEXT | Optional |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

### 4.4 Payment
Tracks financial transactions per booking. Manual in MVP, automated via Chapa in Phase 2.

| Field | Type | Notes |
|---|---|---|
| id | UUID | Primary key |
| booking_id | UUID | FK → bookings |
| amount | DECIMAL(10,2) | In ETB |
| platform_fee | DECIMAL(10,2) | Qetero's cut |
| owner_payout | DECIMAL(10,2) | What owner receives |
| status | ENUM | pending, paid, refunded, failed |
| provider | VARCHAR(50) | chapa, telebirr, manual |
| provider_ref | VARCHAR(255) | External transaction ID |
| created_at | TIMESTAMPTZ | |

### 4.5 Review
Post-rental ratings. Bidirectional: renters review listings, owners review renters.

| Field | Type | Notes |
|---|---|---|
| id | UUID | Primary key |
| booking_id | UUID | FK → bookings (one review per party per booking) |
| reviewer_id | UUID | FK → users |
| reviewee_id | UUID | FK → users (for renter reviews) |
| listing_id | UUID | FK → listings (for listing reviews) |
| rating | INT | 1–5 |
| comment | TEXT | |
| created_at | TIMESTAMPTZ | |

---

## 5. API Design

Base URL: `https://api.qetero.com/v1`

### Auth
| Method | Endpoint | Description |
|---|---|---|
| POST | /auth/register | Register new user (phone required) |
| POST | /auth/login | Login, returns JWT |
| POST | /auth/refresh | Refresh access token |

### Listings
| Method | Endpoint | Description |
|---|---|---|
| GET | /listings | Browse listings (filterable) |
| POST | /listings | Create listing (owner only) |
| GET | /listings/:id | Get single listing |
| PUT | /listings/:id | Update listing (owner only) |
| DELETE | /listings/:id | Soft-delete listing (owner only) |
| GET | /listings/:id/availability | Get booked date ranges |
| GET | /listings/:id/reviews | Get reviews for a listing |

**GET /listings query params:**
- `category` — filter by type
- `location` — filter by location string
- `min_price`, `max_price` — price range per day (ETB)
- `available_from`, `available_to` — date range check
- `page`, `limit` — pagination

### Bookings
| Method | Endpoint | Description |
|---|---|---|
| POST | /bookings | Create booking request |
| GET | /bookings/:id | Get booking details |
| PUT | /bookings/:id/confirm | Owner confirms booking |
| PUT | /bookings/:id/cancel | Cancel booking (owner or renter) |
| GET | /users/me/bookings | My bookings as renter |
| GET | /users/me/listings/bookings | Incoming bookings for my listings |

### Reviews
| Method | Endpoint | Description |
|---|---|---|
| POST | /reviews | Submit a review (requires completed booking) |

### Users
| Method | Endpoint | Description |
|---|---|---|
| GET | /users/me | Get own profile |
| PUT | /users/me | Update profile |
| GET | /users/:id | Public profile (listings + rating) |

---

## 6. Project Structure

```
qetero-backend/
├── internal/
│   ├── auth/
│   │   └── jwt.go
│   ├── config/
│   │   └── config.go
│   ├── db/
│   │   └── db.go
│   ├── handlers/
│   │   ├── auth.go
│   │   ├── listings.go
│   │   ├── bookings.go
│   │   └── users.go
│   ├── middleware/
│   │   └── auth.go
│   ├── models/
│   │   ├── user.go
│   │   ├── listing.go
│   │   └── booking.go
│   ├── repository/
│   │   ├── user_repo.go
│   │   ├── listing_repo.go
│   │   └── booking_repo.go
│   └── telegram/
│       ├── bot.go
│       └── commands.go
├── migrations/
│   ├── 001_create_users.up.sql / .down.sql
│   ├── 002_create_listings.up.sql / .down.sql
│   ├── 003_create_bookings.up.sql / .down.sql
│   ├── 004_create_payments.up.sql / .down.sql
│   └── 005_create_reviews.up.sql / .down.sql
├── main.go
├── .env.example
├── go.mod
└── DESIGN.md
```

---

## 7. Security Checklist

- [x] JWT with short expiry (15 min access, 7 day refresh)
- [x] bcrypt for all passwords (cost factor 12)
- [x] Input validation on all handler inputs before DB write
- [x] SQL via parameterized queries only (no string concatenation)
- [x] Soft deletes — nothing is permanently deleted
- [ ] Rate limiting: 100 req/min per IP globally, 10 req/min on auth endpoints
- [ ] Owner contact info hidden until booking is confirmed
- [ ] CORS restricted to known origins
- [ ] Secrets via environment variables, never in code
- [ ] HTTPS enforced in production

---

## 8. Telegram Bot — MVP Flow

The bot acts as a thin client over the REST API.
Commands should eventually support both English and Amharic.

### Commands
| Command | Description |
|---|---|
| /start | Onboarding, link Telegram to account |
| /browse | Browse available listings |
| /search [category] [location] | Filtered search |
| /listing [id] | View a specific listing |
| /book [id] [start] [end] | Initiate a booking |
| /mybookings | View my active bookings |
| /help | List commands |

### Flow: Browse & Book
```
User: /browse
Bot: Available equipment:
     1. CAT 320 Excavator — Addis Ababa — 450 ETB/day
     2. 50T Crane — Hawassa — 900 ETB/day
     Type /listing <number> for details

User: /listing 1
Bot: CAT 320 Excavator
     Location: Addis Ababa
     Price: 450 ETB/day (min 3 days)
     Specs: 20T, 1.2m³ bucket, diesel
     Available: Now
     Book: /book 1 2026-03-15 2026-03-18

User: /book 1 2026-03-15 2026-03-18
Bot: Booking request sent to owner.
     Total: 1,350 ETB for 3 days.
     Payment: arrange directly with owner (Telebirr/CBE/cash).
     You'll be notified when confirmed.
```

---

## 9. Phased Roadmap

### Phase 1 — MVP (current)
- [x] Project structure
- [x] DB schema & migrations
- [x] Auth (register, login, JWT)
- [x] Listings CRUD
- [x] Availability check
- [x] Bookings (create, confirm, cancel)
- [ ] Telegram bot (browse, search, book)
- [ ] Owner subscription billing (manual, 500 ETB/month)

### Phase 2 — Growth
- [ ] Chapa payment integration (Telebirr, CBE Birr, bank transfer)
- [ ] Automatic platform fee (15%) on each booking
- [ ] Phone OTP auth via Africa's Talking
- [ ] Reviews & ratings
- [ ] SMS/Telegram notifications on booking events
- [ ] Image uploads (Cloudflare R2)
- [ ] Web frontend (Next.js or SvelteKit)
- [ ] Amharic language support in Telegram bot

### Phase 3 — Scale
- [ ] Mobile app
- [ ] Insurance / damage protection per booking
- [ ] Fleet management for large equipment owners
- [ ] Analytics dashboard for owners
- [ ] Expansion to other East African markets

---

## 10. Environment Variables

```env
# Server
PORT=8080

# Database
DATABASE_URL=postgres://postgres:password@localhost:5432/qetero?sslmode=disable

# Auth
JWT_SECRET=change-me-before-deploying

# Telegram
TELEGRAM_BOT_TOKEN=your-bot-token

# Phase 2
# CHAPA_SECRET_KEY=
# AFRICASTALKING_API_KEY=
# AFRICASTALKING_USERNAME=
# R2_ACCOUNT_ID=
# R2_ACCESS_KEY=
# R2_SECRET_KEY=
# R2_BUCKET=
```
