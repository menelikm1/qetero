package telegram

import (
	"context"
	"log"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"qetero/internal/repository"
)

type Bot struct {
	api           *tgbotapi.BotAPI
	users         *repository.UserRepo
	listings      *repository.ListingRepo
	bookings      *repository.BookingRepo
	sessions      *SessionStore
	adminChatID   int64  // 0 = no admin, listings auto-approved
	adminTelebirr string // Telebirr number for deposit payments
}

func New(token string, db *pgxpool.Pool, adminChatID int64, adminTelebirr string) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	log.Printf("Telegram bot authorized as @%s", api.Self.UserName)

	return &Bot{
		api:           api,
		users:         repository.NewUserRepo(db),
		listings:      repository.NewListingRepo(db),
		bookings:      repository.NewBookingRepo(db),
		sessions:      newSessionStore(),
		adminChatID:   adminChatID,
		adminTelebirr: adminTelebirr,
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			b.api.StopReceivingUpdates()
			return
		case update := <-updates:
			if update.CallbackQuery != nil {
				go b.handleCallbackQuery(update.CallbackQuery)
				continue
			}
			if update.Message == nil {
				continue
			}
			go b.handleMessage(update.Message)
		}
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	sess := b.sessions.get(msg.Chat.ID)

	// If mid-wizard, handle input — but let commands interrupt and restart
	if sess.State != StateIdle {
		if msg.IsCommand() {
			b.sessions.reset(msg.Chat.ID)
			// fall through to command handling below
		} else if msg.Photo != nil || msg.Text != "" {
			b.handleWizardStep(msg, sess)
			return
		}
	}

	if !msg.IsCommand() {
		return
	}

	cmd := msg.Command()
	args := strings.TrimSpace(msg.CommandArguments())

	switch cmd {
	case "start":
		b.handleStart(msg)
	case "help":
		b.handleHelp(msg)
	case "cancel":
		b.handleCancel(msg)
	case "register":
		b.handleRegister(msg)
	case "link":
		b.handleLink(msg, args)
	case "browse":
		b.handleBrowse(msg, args)
	case "search":
		b.handleSearch(msg, args)
	case "listing":
		b.handleListing(msg, args)
	case "book":
		b.handleBook(msg, args)
	case "mybookings":
		b.handleMyBookings(msg)
	case "newlisting":
		b.handleNewListing(msg)
	default:
		b.send(msg.Chat.ID, "Unknown command. Type /help to see available commands.")
	}
}

func (b *Bot) handleCallbackQuery(q *tgbotapi.CallbackQuery) {
	// Acknowledge immediately so Telegram removes the loading spinner
	b.api.Request(tgbotapi.NewCallback(q.ID, ""))

	parts := strings.SplitN(q.Data, ":", 2)
	if len(parts) != 2 {
		return
	}
	action, id := parts[0], parts[1]

	switch action {
	case "listing_approve":
		b.handleListingApprove(q, id)
	case "listing_reject":
		b.handleListingReject(q, id)
	case "deposit_verify":
		b.handleDepositVerify(q, id)
	case "deposit_reject":
		b.handleDepositReject(q, id)
	case "owner_confirm":
		b.handleOwnerConfirm(q, id)
	case "owner_decline":
		b.handleOwnerDecline(q, id)
	case "owner_decline_r":
		b.handleOwnerDeclineReason(q, id)
	}
}

// ── Send helpers ─────────────────────────────────────────────────────────────

func (b *Bot) send(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("failed to send message to %d: %v", chatID, err)
	}
}

func (b *Bot) sendWithButtons(chatID int64, text string, buttons [][]tgbotapi.InlineKeyboardButton) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(buttons...)
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("failed to send message with buttons to %d: %v", chatID, err)
	}
}

func (b *Bot) sendPhoto(chatID int64, fileID, caption string) {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileID(fileID))
	photo.Caption = caption
	photo.ParseMode = "Markdown"
	if _, err := b.api.Send(photo); err != nil {
		log.Printf("failed to send photo to %d: %v", chatID, err)
	}
}

func (b *Bot) editMessage(chatID int64, messageID int, text string) {
	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = "Markdown"
	if _, err := b.api.Send(edit); err != nil {
		log.Printf("failed to edit message %d: %v", messageID, err)
	}
}
