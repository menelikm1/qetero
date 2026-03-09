package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"qetero/internal/models"
	"qetero/internal/repository"
)

// ── /start ───────────────────────────────────────────────────────────────────

func (b *Bot) handleStart(msg *tgbotapi.Message) {
	b.send(msg.Chat.ID, `*Welcome to Qetero* 🇪🇹

Ethiopia's equipment rental marketplace.

*New here?* Create your account:
/register

*Already have an account?* Link it:
/link +251912345678

Type /help to see all commands.`)
}

// ── /help ────────────────────────────────────────────────────────────────────

func (b *Bot) handleHelp(msg *tgbotapi.Message) {
	b.send(msg.Chat.ID, `*Available commands:*

*Account*
/register — Create a new Qetero account
/link [phone] — Link an existing account by phone

*Browse & Book*
/browse — Browse all available equipment
/search [category] [location] — Filter listings
/listing [number] — View listing details
/book [number] [start] [end] — Book equipment (dates: YYYY\-MM\-DD)
/mybookings — View your bookings

*Owners*
/newlisting — Add your equipment

*Categories:* excavator, crane, scaffold, compactor, loader, forklift, generator, water\_truck, concrete\_mixer, dump\_truck, dozer, roller, other`)
}

// ── /register ────────────────────────────────────────────────────────────────

func (b *Bot) handleRegister(msg *tgbotapi.Message) {
	ctx := context.Background()
	if _, err := b.users.GetByChatID(ctx, msg.Chat.ID); err == nil {
		b.send(msg.Chat.ID, "You already have a linked account.\nUse /browse to find equipment or /newlisting to add yours.")
		return
	}

	b.sessions.setState(msg.Chat.ID, StateRegisterName)
	b.send(msg.Chat.ID, "Let's create your Qetero account.\n\nWhat's your full name?")
}

// ── /link ────────────────────────────────────────────────────────────────────

func (b *Bot) handleLink(msg *tgbotapi.Message, args string) {
	phone := strings.TrimSpace(args)
	if phone == "" {
		b.send(msg.Chat.ID, "Please provide your phone number.\nExample: /link +251912345678")
		return
	}

	ctx := context.Background()
	user, err := b.users.GetByPhone(ctx, phone)
	if err != nil {
		b.send(msg.Chat.ID, "No account found with that phone number. Use /register to create one.")
		return
	}

	if err := b.users.LinkTelegramChatID(ctx, user.ID, msg.Chat.ID); err != nil {
		b.send(msg.Chat.ID, "Failed to link account. Please try again.")
		return
	}

	b.send(msg.Chat.ID, fmt.Sprintf("Account linked successfully. Welcome, *%s*!\n\nType /browse to see available equipment.", user.Name))
}

// ── /newlisting ──────────────────────────────────────────────────────────────

func (b *Bot) handleNewListing(msg *tgbotapi.Message) {
	ctx := context.Background()
	user, err := b.users.GetByChatID(ctx, msg.Chat.ID)
	if err != nil {
		b.send(msg.Chat.ID, "You need an account first. Use /register to sign up.")
		return
	}
	if user.Role == models.RoleRenter {
		b.send(msg.Chat.ID, "Your account is set to renter only.\nContact support to switch your role to owner.")
		return
	}

	b.sessions.setState(msg.Chat.ID, StateListingTitle)
	b.sessions.setData(msg.Chat.ID, "owner_id", user.ID.String())
	b.send(msg.Chat.ID, "Let's add your equipment to Qetero.\n\nWhat's the title? (e.g. CAT 320 Excavator, 50T Liebherr Crane)")
}

// ── /browse ──────────────────────────────────────────────────────────────────

func (b *Bot) handleBrowse(msg *tgbotapi.Message, args string) {
	ctx := context.Background()

	listings, err := b.listings.Browse(ctx, repository.ListingFilter{Page: 1, Limit: 10})
	if err != nil {
		b.send(msg.Chat.ID, "Failed to fetch listings. Please try again.")
		return
	}
	if len(listings) == 0 {
		b.send(msg.Chat.ID, "No equipment available right now. Check back soon.")
		return
	}

	b.sendListingResults(msg.Chat.ID, listings)
}

// ── /search ──────────────────────────────────────────────────────────────────

func (b *Bot) handleSearch(msg *tgbotapi.Message, args string) {
	parts := strings.Fields(args)
	f := repository.ListingFilter{Page: 1, Limit: 10}

	if len(parts) >= 1 {
		f.Category = parts[0]
	}
	if len(parts) >= 2 {
		f.Location = strings.Join(parts[1:], " ")
	}

	if f.Category == "" && f.Location == "" {
		b.send(msg.Chat.ID, "Usage: /search [category] [location]\nExample: /search excavator Addis")
		return
	}

	ctx := context.Background()
	listings, err := b.listings.Browse(ctx, f)
	if err != nil {
		b.send(msg.Chat.ID, "Failed to fetch listings. Please try again.")
		return
	}
	if len(listings) == 0 {
		b.send(msg.Chat.ID, "No listings found matching your search.")
		return
	}

	b.sendListingResults(msg.Chat.ID, listings)
}

func (b *Bot) sendListingResults(chatID int64, listings []models.Listing) {
	ids := make([]uuid.UUID, len(listings))
	var sb strings.Builder
	sb.WriteString("*Available equipment:*\n\n")

	for i, l := range listings {
		ids[i] = l.ID
		sb.WriteString(fmt.Sprintf(
			"%d. *%s*\n   %s — %.0f ETB/day (min %d days)\n\n",
			i+1, l.Title, l.Location, l.PricePerDay, l.MinimumDays,
		))
	}
	sb.WriteString("Type /listing [number] for details.")

	b.sessions.setListings(chatID, ids)
	b.send(chatID, sb.String())
}

// ── /listing ─────────────────────────────────────────────────────────────────

func (b *Bot) handleListing(msg *tgbotapi.Message, args string) {
	n, err := strconv.Atoi(strings.TrimSpace(args))
	if err != nil || n < 1 {
		b.send(msg.Chat.ID, "Usage: /listing [number]\nBrowse first with /browse to get numbers.")
		return
	}

	sess := b.sessions.get(msg.Chat.ID)
	if n > len(sess.LastListings) {
		b.send(msg.Chat.ID, "Invalid number. Use /browse to refresh the list.")
		return
	}

	ctx := context.Background()
	l, err := b.listings.GetByID(ctx, sess.LastListings[n-1])
	if err != nil {
		b.send(msg.Chat.ID, "Listing not found.")
		return
	}

	today := time.Now().Format("2006-01-02")
	weekLater := time.Now().AddDate(0, 0, 7).Format("2006-01-02")

	b.send(msg.Chat.ID, fmt.Sprintf(
		"*%s*\nCategory: %s\nLocation: %s\nPrice: *%.0f ETB/day* (min %d days)\n\n%s\n\nTo book:\n`/book %d %s %s`",
		l.Title, l.Category, l.Location, l.PricePerDay, l.MinimumDays,
		l.Description,
		n, today, weekLater,
	))
}

// ── /book ────────────────────────────────────────────────────────────────────

func (b *Bot) handleBook(msg *tgbotapi.Message, args string) {
	ctx := context.Background()

	user, err := b.users.GetByChatID(ctx, msg.Chat.ID)
	if err != nil {
		b.send(msg.Chat.ID, "You need to link your account first.\nUse /register to sign up or /link +251912345678.")
		return
	}

	parts := strings.Fields(args)
	if len(parts) != 3 {
		b.send(msg.Chat.ID, "Usage: /book [number] [start] [end]\nExample: /book 1 2026-03-15 2026-03-18")
		return
	}

	n, err := strconv.Atoi(parts[0])
	if err != nil || n < 1 {
		b.send(msg.Chat.ID, "Invalid listing number. Use /browse to see listings.")
		return
	}

	sess := b.sessions.get(msg.Chat.ID)
	if n > len(sess.LastListings) {
		b.send(msg.Chat.ID, "Invalid number. Use /browse to refresh the list.")
		return
	}

	start, err := time.Parse("2006-01-02", parts[1])
	if err != nil {
		b.send(msg.Chat.ID, "Invalid start date. Use format: YYYY-MM-DD")
		return
	}
	end, err := time.Parse("2006-01-02", parts[2])
	if err != nil {
		b.send(msg.Chat.ID, "Invalid end date. Use format: YYYY-MM-DD")
		return
	}

	if !end.After(start) {
		b.send(msg.Chat.ID, "End date must be after start date.")
		return
	}
	if start.Before(time.Now().Truncate(24 * time.Hour)) {
		b.send(msg.Chat.ID, "Start date cannot be in the past.")
		return
	}

	listingID := sess.LastListings[n-1]
	listing, err := b.listings.GetByID(ctx, listingID)
	if err != nil {
		b.send(msg.Chat.ID, "Listing not found.")
		return
	}
	if !listing.IsAvailable {
		b.send(msg.Chat.ID, "Sorry, this listing is not currently available.")
		return
	}
	if listing.OwnerID == user.ID {
		b.send(msg.Chat.ID, "You cannot book your own listing.")
		return
	}

	days := int(end.Sub(start).Hours()/24) + 1
	if days < listing.MinimumDays {
		b.send(msg.Chat.ID, fmt.Sprintf("Minimum rental is %d days.", listing.MinimumDays))
		return
	}

	conflict, err := b.bookings.HasConflict(ctx, listingID, start, end)
	if err != nil {
		b.send(msg.Chat.ID, "Failed to check availability. Please try again.")
		return
	}
	if conflict {
		b.send(msg.Chat.ID, "Those dates are already booked. Use /listing to see another option.")
		return
	}

	booking := &models.Booking{
		ID:         uuid.New(),
		ListingID:  listingID,
		RenterID:   user.ID,
		OwnerID:    listing.OwnerID,
		StartDate:  start,
		EndDate:    end,
		TotalDays:  days,
		TotalPrice: float64(days) * listing.PricePerDay,
		Status:     models.StatusPending,
	}

	if err := b.bookings.Create(ctx, booking); err != nil {
		b.send(msg.Chat.ID, "Failed to create booking. Please try again.")
		return
	}

	b.send(msg.Chat.ID, fmt.Sprintf(
		"Booking request sent!\n\n*%s*\n%s to %s (%d days)\nTotal: *%.0f ETB*\n\nPayment: arrange directly with the owner via Telebirr, CBE, or cash.\n\nYou'll be notified when the owner confirms.",
		listing.Title,
		start.Format("Jan 2"),
		end.Format("Jan 2, 2006"),
		days,
		booking.TotalPrice,
	))
}

// ── /mybookings ──────────────────────────────────────────────────────────────

func (b *Bot) handleMyBookings(msg *tgbotapi.Message) {
	ctx := context.Background()

	user, err := b.users.GetByChatID(ctx, msg.Chat.ID)
	if err != nil {
		b.send(msg.Chat.ID, "You need to link your account first.\nUse /register to sign up or /link +251912345678.")
		return
	}

	bookings, err := b.bookings.GetByRenter(ctx, user.ID)
	if err != nil {
		b.send(msg.Chat.ID, "Failed to fetch bookings.")
		return
	}
	if len(bookings) == 0 {
		b.send(msg.Chat.ID, "You have no bookings yet.\nUse /browse to find equipment.")
		return
	}

	var sb strings.Builder
	sb.WriteString("*Your bookings:*\n\n")
	for _, bk := range bookings {
		sb.WriteString(fmt.Sprintf(
			"• %s to %s — *%s* — %.0f ETB\n",
			bk.StartDate.Format("Jan 2"),
			bk.EndDate.Format("Jan 2"),
			strings.ToUpper(string(bk.Status)),
			bk.TotalPrice,
		))
	}

	b.send(msg.Chat.ID, sb.String())
}

// ── Wizard step handler ──────────────────────────────────────────────────────

func (b *Bot) handleWizardStep(msg *tgbotapi.Message, sess *Session) {
	text := strings.TrimSpace(msg.Text)
	chatID := msg.Chat.ID
	ctx := context.Background()

	switch sess.State {

	// ── Registration ──────────────────────────────────────────────────────────

	case StateRegisterName:
		if len(text) < 2 {
			b.send(chatID, "Please enter your full name (at least 2 characters).")
			return
		}
		b.sessions.setData(chatID, "name", text)
		b.sessions.setState(chatID, StateRegisterPhone)
		b.send(chatID, "What's your phone number? (e.g. +251912345678)")

	case StateRegisterPhone:
		if len(text) < 9 {
			b.send(chatID, "Please enter a valid phone number (e.g. +251912345678).")
			return
		}
		if _, err := b.users.GetByPhone(ctx, text); err == nil {
			b.sessions.reset(chatID)
			b.send(chatID, "That phone number is already registered. Use /link to connect your account.")
			return
		}
		b.sessions.setData(chatID, "phone", text)
		b.sessions.setState(chatID, StateRegisterPassword)
		b.send(chatID, "Create a password (minimum 8 characters).")

	case StateRegisterPassword:
		if len(text) < 8 {
			b.send(chatID, "Password must be at least 8 characters. Try again.")
			return
		}
		b.sessions.setData(chatID, "password", text)
		b.sessions.setState(chatID, StateRegisterRole)
		b.send(chatID, "What's your role?\n\n1. Renter — I want to rent equipment\n2. Owner — I want to list my equipment\n3. Both — I want to do both")

	case StateRegisterRole:
		var role models.UserRole
		switch text {
		case "1", "renter":
			role = models.RoleRenter
		case "2", "owner":
			role = models.RoleOwner
		case "3", "both":
			role = models.RoleBoth
		default:
			b.send(chatID, "Please reply with 1, 2, or 3.")
			return
		}

		data := sess.Data
		hash, err := bcrypt.GenerateFromPassword([]byte(data["password"]), 12)
		if err != nil {
			b.sessions.reset(chatID)
			b.send(chatID, "Something went wrong. Please try /register again.")
			return
		}

		user := &models.User{
			ID:           uuid.New(),
			Name:         data["name"],
			Phone:        data["phone"],
			PasswordHash: string(hash),
			Role:         role,
		}
		if err := b.users.Create(ctx, user); err != nil {
			b.sessions.reset(chatID)
			b.send(chatID, "Registration failed. That phone number may already be registered.")
			return
		}
		if err := b.users.LinkTelegramChatID(ctx, user.ID, chatID); err != nil {
			log.Printf("failed to link telegram chat ID after registration for user %s: %v", user.ID, err)
		}

		b.sessions.reset(chatID)

		next := "Browse available equipment with /browse."
		if role == models.RoleOwner || role == models.RoleBoth {
			next = "Add your first listing with /newlisting, or browse with /browse."
		}
		b.send(chatID, fmt.Sprintf("Welcome to Qetero, *%s*!\n\nYour account is ready. %s", user.Name, next))

	// ── Listing creation ──────────────────────────────────────────────────────

	case StateListingTitle:
		if len(text) < 3 {
			b.send(chatID, "Please enter a title (e.g. CAT 320 Excavator).")
			return
		}
		b.sessions.setData(chatID, "title", text)
		b.sessions.setState(chatID, StateListingCategory)
		b.send(chatID, "What category is it?\n\n"+categoryMenu())

	case StateListingCategory:
		cat, ok := parseCategory(text)
		if !ok {
			b.send(chatID, "Please enter a number from the list.\n\n"+categoryMenu())
			return
		}
		b.sessions.setData(chatID, "category", string(cat))
		b.sessions.setState(chatID, StateListingLocation)
		b.send(chatID, "Where is the equipment located? (e.g. Addis Ababa, Hawassa)")

	case StateListingLocation:
		if len(text) < 2 {
			b.send(chatID, "Please enter a location.")
			return
		}
		b.sessions.setData(chatID, "location", text)
		b.sessions.setState(chatID, StateListingPrice)
		b.send(chatID, "What's your price per day in ETB? (numbers only, e.g. 4500)")

	case StateListingPrice:
		price, err := strconv.ParseFloat(text, 64)
		if err != nil || price <= 0 {
			b.send(chatID, "Please enter a valid price (e.g. 4500).")
			return
		}
		b.sessions.setData(chatID, "price", text)
		b.sessions.setState(chatID, StateListingMinDays)
		b.send(chatID, "What's the minimum rental period in days? (enter 1 if no minimum)")

	case StateListingMinDays:
		days, err := strconv.Atoi(text)
		if err != nil || days < 1 {
			b.send(chatID, "Please enter a whole number of days (e.g. 1, 3, 7).")
			return
		}
		b.sessions.setData(chatID, "min_days", text)
		b.sessions.setState(chatID, StateListingDescription)
		b.send(chatID, "Add a description — specs, condition, what's included.\n(e.g. 20T capacity, 1.2m³ bucket, diesel, 2019 model, good condition)")

	case StateListingDescription:
		if len(text) < 10 {
			b.send(chatID, "Please add a description (at least 10 characters) to help renters.")
			return
		}

		data := sess.Data
		ownerID, _ := uuid.Parse(data["owner_id"])
		price, _ := strconv.ParseFloat(data["price"], 64)
		minDays, _ := strconv.Atoi(data["min_days"])

		listing := &models.Listing{
			ID:          uuid.New(),
			OwnerID:     ownerID,
			Title:       data["title"],
			Category:    models.ListingCategory(data["category"]),
			Location:    data["location"],
			PricePerDay: price,
			MinimumDays: minDays,
			Description: text,
			Images:      []string{},
			Specs:       json.RawMessage("{}"),
			IsAvailable: true,
		}

		if err := b.listings.Create(ctx, listing); err != nil {
			b.sessions.reset(chatID)
			b.send(chatID, "Failed to create listing. Please try again with /newlisting.")
			return
		}

		b.sessions.reset(chatID)
		b.send(chatID, fmt.Sprintf(
			"Listing created!\n\n*%s*\n%s — %.0f ETB/day (min %d days)\n\nRenters can now find and book your equipment.",
			listing.Title, listing.Location, listing.PricePerDay, listing.MinimumDays,
		))
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func categoryMenu() string {
	return `1. Excavator
2. Crane
3. Scaffold
4. Compactor
5. Loader
6. Forklift
7. Generator
8. Water Truck
9. Concrete Mixer
10. Dump Truck
11. Dozer
12. Roller
13. Other`
}

func parseCategory(text string) (models.ListingCategory, bool) {
	categories := []models.ListingCategory{
		models.CategoryExcavator,
		models.CategoryCrane,
		models.CategoryScaffold,
		models.CategoryCompactor,
		models.CategoryLoader,
		models.CategoryForklift,
		models.CategoryGenerator,
		models.CategoryWaterTruck,
		models.CategoryConcreteMixer,
		models.CategoryDumpTruck,
		models.CategoryDozer,
		models.CategoryRoller,
		models.CategoryOther,
	}

	n, err := strconv.Atoi(text)
	if err == nil && n >= 1 && n <= len(categories) {
		return categories[n-1], true
	}

	// Also accept typed category name
	for _, c := range categories {
		if strings.EqualFold(string(c), strings.ReplaceAll(text, " ", "_")) {
			return c, true
		}
	}
	return "", false
}
