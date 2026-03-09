package telegram

import (
	"context"
	"fmt"
	"log"
	"math"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

	"qetero/internal/models"
)

// notifyAdminNewListing sends the admin a listing review request with approve/reject buttons.
// If adminChatID is 0, the listing is auto-approved.
func (b *Bot) notifyAdminNewListing(ctx context.Context, listing *models.Listing, owner *models.User) {
	if b.adminChatID == 0 {
		// Dev mode — auto-approve
		if err := b.listings.UpdateStatus(ctx, listing.ID, models.ListingStatusActive); err != nil {
			log.Printf("auto-approve listing %s failed: %v", listing.ID, err)
			return
		}
		if owner.TelegramChatID != nil {
			b.send(*owner.TelegramChatID, fmt.Sprintf(
				"✅ Your listing *%s* is now live on Qetero!",
				listing.Title,
			))
		}
		return
	}

	// Send photos first
	for _, fileID := range listing.Images {
		photo := tgbotapi.NewPhoto(b.adminChatID, tgbotapi.FileID(fileID))
		b.api.Send(photo)
	}

	text := fmt.Sprintf(
		"🏗 *New listing pending review*\n\n*%s*\nCategory: %s | Location: %s\nPrice: %.0f ETB/day (min %d days)\n\nOwner: %s (%s)\n\n_%s_",
		listing.Title, listing.Category, listing.Location,
		listing.PricePerDay, listing.MinimumDays,
		owner.Name, owner.Phone,
		listing.Description,
	)

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ Approve", "listing_approve:"+listing.ID.String()),
			tgbotapi.NewInlineKeyboardButtonData("❌ Reject", "listing_reject:"+listing.ID.String()),
		},
	}
	b.sendWithButtons(b.adminChatID, text, buttons)
}

// notifyAdminNewDeposit sends the admin a deposit verification request.
func (b *Bot) notifyAdminNewDeposit(ctx context.Context, booking *models.Booking, listing *models.Listing, renter *models.User) {
	if b.adminChatID == 0 {
		// Dev mode — auto-verify, notify owner
		if err := b.bookings.UpdateDeposit(ctx, booking.ID, booking.DepositRef, models.DepositStatusVerified); err != nil {
			log.Printf("auto-verify deposit for booking %s failed: %v", booking.ID, err)
			return
		}
		b.notifyOwnerBookingRequest(ctx, booking, listing, renter)
		return
	}

	deposit := math.Round(booking.TotalPrice * 0.20)
	text := fmt.Sprintf(
		"💰 *New deposit — verify payment*\n\nBooking: *%s*\nRenter: %s (%s)\nDates: %s – %s (%d days)\nTotal: %.0f ETB | Deposit (20%%): %.0f ETB\n\nTelebirr ref: `%s`",
		listing.Title,
		renter.Name, renter.Phone,
		booking.StartDate.Format("Jan 2"), booking.EndDate.Format("Jan 2, 2006"),
		booking.TotalDays,
		booking.TotalPrice, deposit,
		booking.DepositRef,
	)

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ Verified", "deposit_verify:"+booking.ID.String()),
			tgbotapi.NewInlineKeyboardButtonData("❌ Reject", "deposit_reject:"+booking.ID.String()),
		},
	}
	b.sendWithButtons(b.adminChatID, text, buttons)
}

// notifyOwnerBookingRequest sends the owner a new booking request with confirm/decline buttons.
func (b *Bot) notifyOwnerBookingRequest(ctx context.Context, booking *models.Booking, listing *models.Listing, renter *models.User) {
	owner, err := b.users.GetByID(ctx, booking.OwnerID)
	if err != nil || owner.TelegramChatID == nil {
		log.Printf("cannot notify owner for booking %s: %v", booking.ID, err)
		return
	}

	text := fmt.Sprintf(
		"📋 *New booking request — deposit verified*\n\n*%s*\n%s – %s (%d days) — %.0f ETB\n\nRenter: %s (%s)",
		listing.Title,
		booking.StartDate.Format("Jan 2"), booking.EndDate.Format("Jan 2, 2006"),
		booking.TotalDays, booking.TotalPrice,
		renter.Name, renter.Phone,
	)

	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("✅ Confirm", "owner_confirm:"+booking.ID.String()),
			tgbotapi.NewInlineKeyboardButtonData("❌ Decline", "owner_decline:"+booking.ID.String()),
		},
	}
	b.sendWithButtons(*owner.TelegramChatID, text, buttons)
}
