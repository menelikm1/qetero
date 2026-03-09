# Qetero

Ethiopia's peer-to-peer rental marketplace for heavy machinery and construction equipment.

## What is Qetero?

Qetero connects equipment owners with contractors, builders, and individuals who need machinery without buying it. Think Turo, but for excavators, cranes, concrete mixers, and more — built for the Ethiopian market.

## Stack

- **Go 1.23** + Gin
- **PostgreSQL** — migrations via golang-migrate
- **JWT** authentication
- **Telegram bot** — primary user interface (`@qetero_ethiopia_bot`)
- **Chapa** — payment integration (Phase 2)

## Getting Started

### Prerequisites
- Go 1.23+
- Docker (for local Postgres)
- [golang-migrate CLI](https://github.com/golang-migrate/migrate)

### Setup

**1. Start the database**
```bash
docker run -d --name qetero-db -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=password -e POSTGRES_DB=qetero -p 5432:5432 postgres:16
```

**2. Configure environment**
```bash
cp .env.example .env
# Edit .env and fill in JWT_SECRET and optionally TELEGRAM_BOT_TOKEN
```

**3. Run migrations**
```bash
migrate -path migrations -database "postgres://postgres:password@localhost:5432/qetero?sslmode=disable" up
```

**4. Start the server**
```bash
go run main.go
```

The API will be available at `http://localhost:8080`.
The Telegram bot starts automatically if `TELEGRAM_BOT_TOKEN` is set.

## API

Base URL: `/v1`

### Auth
| Method | Endpoint | Auth | Description |
|---|---|---|---|
| POST | `/auth/register` | — | Register (phone required) |
| POST | `/auth/login` | — | Login, returns JWT |

### Listings
| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/listings` | — | Browse listings |
| POST | `/listings` | Required | Create listing |
| GET | `/listings/:id` | — | Get listing |
| PUT | `/listings/:id` | Required | Update listing |
| DELETE | `/listings/:id` | Required | Delete listing |
| GET | `/listings/:id/availability` | — | Get booked date ranges |

### Bookings
| Method | Endpoint | Auth | Description |
|---|---|---|---|
| POST | `/bookings` | Required | Create booking request |
| GET | `/bookings/:id` | Required | Get booking |
| PUT | `/bookings/:id/confirm` | Required | Owner confirms |
| PUT | `/bookings/:id/cancel` | Required | Cancel booking |
| GET | `/users/me/bookings` | Required | My bookings |
| GET | `/users/me/listings/bookings` | Required | Incoming bookings |

### Users
| Method | Endpoint | Auth | Description |
|---|---|---|---|
| GET | `/users/me` | Required | Get profile |
| PUT | `/users/me` | Required | Update profile |

## Telegram Bot

Commands available via `@qetero_ethiopia_bot`:

| Command | Description |
|---|---|
| `/start` | Welcome + onboarding |
| `/link [phone]` | Link Telegram to your account |
| `/browse` | Browse available equipment |
| `/search [category] [location]` | Filter listings |
| `/listing [number]` | View listing details |
| `/book [number] [start] [end]` | Book equipment |
| `/mybookings` | View your bookings |
| `/help` | List all commands |

## Equipment Categories

`excavator` `crane` `scaffold` `compactor` `loader` `forklift` `generator` `water_truck` `concrete_mixer` `dump_truck` `dozer` `roller` `other`

## Roadmap

- **Phase 1 (current):** API + Telegram bot, manual payments
- **Phase 2:** Chapa integration, OTP auth via Africa's Talking, image uploads, web frontend
- **Phase 3:** Mobile app, agricultural equipment vertical, East Africa expansion

## License

Private — all rights reserved.
