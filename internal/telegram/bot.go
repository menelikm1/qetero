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
	api      *tgbotapi.BotAPI
	users    *repository.UserRepo
	listings *repository.ListingRepo
	bookings *repository.BookingRepo
	sessions *SessionStore
}

func New(token string, db *pgxpool.Pool) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	log.Printf("Telegram bot authorized as @%s", api.Self.UserName)

	return &Bot{
		api:      api,
		users:    repository.NewUserRepo(db),
		listings: repository.NewListingRepo(db),
		bookings: repository.NewBookingRepo(db),
		sessions: newSessionStore(),
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

func (b *Bot) send(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "Markdown"
	if _, err := b.api.Send(msg); err != nil {
		log.Printf("failed to send message to %d: %v", chatID, err)
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
