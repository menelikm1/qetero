package telegram

import (
	"context"
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/google/uuid"

	"qetero/internal/models"
)

// ── Listing approval ──────────────────────────────────────────────────────────

func (b *Bot) handleListingApprove(q *tgbotapi.CallbackQuery, idStr string) {
	ctx := context.Background()
	listingID, err := uuid.Parse(idStr)
	if err != nil {
		return
	}

	if err := b.listings.UpdateStatus(ctx, listingID, models.ListingStatusActive); err != nil {
		b.send(q.Message.Chat.ID, "Failed to approve listing.")
		return
	}

	listing, err := b.listings.GetByID(ctx, listingID)
	if err != nil {
		return
	}

	b.editMessage(q.Message.Chat.ID, q.Message.MessageID,
		fmt.Sprintf("✅ *Approved:* %s", listing.Title))

	owner, err := b.users.GetByID(ctx, listing.OwnerID)
	if err != nil || owner.TelegramChatID == nil {
		return
	}
	b.send(*owner.TelegramChatID, fmt.Sprintf(
		"✅ Great news! Your listing *%s* has been approved and is now live on Qetero.\n\nRenters can now find and book your equipment.",
		listing.Title,
	))
}

func (b *Bot) handleListingReject(q *tgbotapi.CallbackQuery, idStr string) {
	ctx := context.Background()
	listingID, err := uuid.Parse(idStr)
	if err != nil {
		return
	}

	if err := b.listings.UpdateStatus(ctx, listingID, models.ListingStatusRejected); err != nil {
		b.send(q.Message.Chat.ID, "Failed to reject listing.")
		return
	}

	listing, err := b.listings.GetByID(ctx, listingID)
	if err != nil {
		return
	}

	b.editMessage(q.Message.Chat.ID, q.Message.MessageID,
		fmt.Sprintf("❌ *Rejected:* %s", listing.Title))

	owner, err := b.users.GetByID(ctx, listing.OwnerID)
	if err != nil || owner.TelegramChatID == nil {
		return
	}
	b.send(*owner.TelegramChatID, fmt.Sprintf(
		"Your listing *%s* was not approved.\n\nPlease resubmit with clearer photos and a more detailed description. Type /newlisting to try again.",
		listing.Title,
	))
}

// ── Deposit verification ──────────────────────────────────────────────────────

func (b *Bot) handleDepositVerify(q *tgbotapi.CallbackQuery, idStr string) {
	ctx := context.Background()
	bookingID, err := uuid.Parse(idStr)
	if err != nil {
		return
	}

	booking, err := b.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return
	}

	if err := b.bookings.UpdateDeposit(ctx, bookingID, booking.DepositRef, models.DepositStatusVerified); err != nil {
		b.send(q.Message.Chat.ID, "Failed to verify deposit.")
		return
	}

	listing, err := b.listings.GetByID(ctx, booking.ListingID)
	if err != nil {
		return
	}
	renter, err := b.users.GetByID(ctx, booking.RenterID)
	if err != nil {
		return
	}

	b.editMessage(q.Message.Chat.ID, q.Message.MessageID,
		fmt.Sprintf("✅ *Deposit verified* — %s (%s – %s)", listing.Title,
			booking.StartDate.Format("Jan 2"), booking.EndDate.Format("Jan 2")))

	// Tell renter their deposit is verified
	if renter.TelegramChatID != nil {
		b.send(*renter.TelegramChatID,
			"✅ Your deposit has been verified!\n\nWe've notified the owner. You'll hear back once they confirm the dates.")
	}

	// Notify owner to confirm
	b.notifyOwnerBookingRequest(ctx, booking, listing, renter)
}

func (b *Bot) handleDepositReject(q *tgbotapi.CallbackQuery, idStr string) {
	ctx := context.Background()
	bookingID, err := uuid.Parse(idStr)
	if err != nil {
		return
	}

	booking, err := b.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return
	}

	b.bookings.UpdateDeposit(ctx, bookingID, booking.DepositRef, models.DepositStatusRejected)
	b.bookings.UpdateStatus(ctx, bookingID, models.StatusCancelled, "deposit not verified")

	listing, err := b.listings.GetByID(ctx, booking.ListingID)
	if err != nil {
		return
	}

	b.editMessage(q.Message.Chat.ID, q.Message.MessageID,
		fmt.Sprintf("❌ *Deposit rejected* — %s", listing.Title))

	renter, err := b.users.GetByID(ctx, booking.RenterID)
	if err != nil || renter.TelegramChatID == nil {
		return
	}
	b.send(*renter.TelegramChatID,
		"We could not verify your deposit reference for the booking.\n\nPlease contact us to resolve this or try booking again.")
}

// ── Owner confirm/decline ─────────────────────────────────────────────────────

func (b *Bot) handleOwnerConfirm(q *tgbotapi.CallbackQuery, idStr string) {
	ctx := context.Background()
	bookingID, err := uuid.Parse(idStr)
	if err != nil {
		return
	}

	booking, err := b.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return
	}

	if err := b.bookings.UpdateStatus(ctx, bookingID, models.StatusConfirmed, ""); err != nil {
		b.send(q.Message.Chat.ID, "Failed to confirm booking.")
		return
	}

	listing, err := b.listings.GetByID(ctx, booking.ListingID)
	if err != nil {
		return
	}
	owner, err := b.users.GetByID(ctx, booking.OwnerID)
	if err != nil {
		return
	}

	b.editMessage(q.Message.Chat.ID, q.Message.MessageID,
		fmt.Sprintf("✅ *Confirmed* — %s (%s – %s)",
			listing.Title,
			booking.StartDate.Format("Jan 2"),
			booking.EndDate.Format("Jan 2")))

	renter, err := b.users.GetByID(ctx, booking.RenterID)
	if err != nil || renter.TelegramChatID == nil {
		return
	}
	b.send(*renter.TelegramChatID, fmt.Sprintf(
		"✅ *Booking confirmed!*\n\n*%s*\n%s – %s (%d days)\n\nContact the owner to arrange the rental:\n*%s* — %s\n\nTotal: %.0f ETB. Arrange payment directly with the owner.",
		listing.Title,
		booking.StartDate.Format("Jan 2"), booking.EndDate.Format("Jan 2, 2006"),
		booking.TotalDays,
		owner.Name, owner.Phone,
		booking.TotalPrice,
	))
}

func (b *Bot) handleOwnerDecline(q *tgbotapi.CallbackQuery, idStr string) {
	// Show reason selection buttons instead of immediately cancelling
	buttons := [][]tgbotapi.InlineKeyboardButton{
		{
			tgbotapi.NewInlineKeyboardButtonData("📅 Dates not available", "owner_decline_r:"+idStr+":dates"),
			tgbotapi.NewInlineKeyboardButtonData("🔧 In maintenance", "owner_decline_r:"+idStr+":maintenance"),
		},
		{
			tgbotapi.NewInlineKeyboardButtonData("📋 Other reason", "owner_decline_r:"+idStr+":other"),
		},
	}
	edit := tgbotapi.NewEditMessageReplyMarkup(q.Message.Chat.ID, q.Message.MessageID,
		tgbotapi.NewInlineKeyboardMarkup(buttons...))
	b.api.Send(edit)
}

func (b *Bot) handleOwnerDeclineReason(q *tgbotapi.CallbackQuery, idAndReason string) {
	ctx := context.Background()

	parts := strings.SplitN(idAndReason, ":", 2)
	if len(parts) != 2 {
		return
	}
	bookingID, err := uuid.Parse(parts[0])
	if err != nil {
		return
	}
	reasonCode := parts[1]

	reasonText := map[string]string{
		"dates":       "Dates not available",
		"maintenance": "Equipment in maintenance",
		"other":       "Owner declined",
	}[reasonCode]
	if reasonText == "" {
		reasonText = "Owner declined"
	}

	booking, err := b.bookings.GetByID(ctx, bookingID)
	if err != nil {
		return
	}

	b.bookings.UpdateStatus(ctx, bookingID, models.StatusCancelled, reasonText)

	listing, err := b.listings.GetByID(ctx, booking.ListingID)
	if err != nil {
		return
	}

	b.editMessage(q.Message.Chat.ID, q.Message.MessageID,
		fmt.Sprintf("❌ *Declined* — %s\nReason: %s", listing.Title, reasonText))

	renter, err := b.users.GetByID(ctx, booking.RenterID)
	if err != nil || renter.TelegramChatID == nil {
		return
	}
	b.send(*renter.TelegramChatID, fmt.Sprintf(
		"The owner was unable to accommodate your booking for *%s*.\nReason: %s\n\nYour deposit will be refunded. Contact us if you need help.",
		listing.Title, reasonText,
	))

	// Alert admin to process refund
	if b.adminChatID != 0 {
		b.send(b.adminChatID, fmt.Sprintf(
			"⚠️ Owner declined booking for *%s* (%s – %s). Reason: %s\nProcess deposit refund to renter: %s (%s).",
			listing.Title,
			booking.StartDate.Format("Jan 2"), booking.EndDate.Format("Jan 2"),
			reasonText,
			renter.Name, renter.Phone,
		))
	}
}
