package handlers

import (
	"context"
	"log/slog"
	"strconv"
	"telegram-anonymous-chat/internal/services"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type TelegramHandler struct {
	service *services.ChatService
}

func NewTelegramHandler(s *services.ChatService) *TelegramHandler {
	return &TelegramHandler{service: s}
}

func sendMessages(ctx context.Context, b *bot.Bot, res services.ServiceResult) {
	for _, msg := range res.Messages {
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.ChatID,
			Text:   msg.Message,
		})
		if err != nil {
			slog.Error("sendMessage", "error", err, "chat_id", msg.ChatID)
		}
	}
}

func (h *TelegramHandler) handlePhoto(ctx context.Context, b *bot.Bot, update *models.Update, partnerID int64) {
	photo := update.Message.Photo[len(update.Message.Photo)-1]
	if partnerID == 0 {
		return
	}

	_, err := b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:  partnerID,
		Photo:   &models.InputFileString{Data: photo.FileID},
		Caption: update.Message.Caption,
	})
	if err != nil {
		slog.Error("handlePhoto", "error", err, "chat_id", partnerID)
	}
}

func (h *TelegramHandler) handleVideo(ctx context.Context, b *bot.Bot, update *models.Update, partnerID int64) {
	video := update.Message.Video

	_, err := b.SendVideo(ctx, &bot.SendVideoParams{
		ChatID:  partnerID,
		Video:   &models.InputFileString{Data: video.FileID},
		Caption: update.Message.Caption,
	})
	if err != nil {
		slog.Error("handleVideo", "error", err, "chat_id", partnerID)
	}
}

func (h *TelegramHandler) handleVoice(ctx context.Context, b *bot.Bot, update *models.Update, partnerID int64) {
	voice := update.Message.Voice

	_, err := b.SendVoice(ctx, &bot.SendVoiceParams{
		ChatID:  partnerID,
		Voice:   &models.InputFileString{Data: voice.FileID},
		Caption: update.Message.Caption,
	})
	if err != nil {
		slog.Error("handleVoice", "error", err, "chat_id", partnerID)
	}
}

func (h *TelegramHandler) handleVideoMessage(ctx context.Context, b *bot.Bot, update *models.Update, partnerID int64) {
	videoNote := update.Message.VideoNote

	_, err := b.SendVideoNote(ctx, &bot.SendVideoNoteParams{
		ChatID:    partnerID,
		VideoNote: &models.InputFileString{Data: videoNote.FileID},
	})
	if err != nil {
		slog.Error("handleVideoMessage", "error", err, "chat_id", partnerID)
	}
}

func (h *TelegramHandler) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	userID := update.Message.From.ID
	res, err := h.service.Default(userID, update.Message.Text)
	if err != nil {
		slog.Error("DefaultHandler", "error", err, "user_id", userID)
		return
	}

	sendMessages(ctx, b, res)
	partnerID := h.service.GetPartner(userID)
	if partnerID == 0 {
		return
	}
	if update.Message.Photo != nil && len(update.Message.Photo) != 0 {
		h.handlePhoto(ctx, b, update, partnerID)
	}
	if update.Message.Video != nil {
		h.handleVideo(ctx, b, update, partnerID)
	}
	if update.Message.Voice != nil {
		h.handleVoice(ctx, b, update, partnerID)
	}
	if update.Message.VideoNote != nil {
		h.handleVideoMessage(ctx, b, update, partnerID)
	}
}

func (h *TelegramHandler) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	username := update.Message.From.Username
	res, err := h.service.Start(userID, username)
	if err != nil {
		slog.Error("StartHandler", "error", err, "user_id", userID)
		return
	}

	sendMessages(ctx, b, res)
}

func sendRatingKeyboard(ctx context.Context, b *bot.Bot, userIDs []int64) {
	for i, id := range userIDs {
		keyboard := &models.InlineKeyboardMarkup{
			InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{Text: "👍", CallbackData: "like:"},
					{Text: "👎", CallbackData: "dislike:"},
				},
			},
		}
		keyboard.InlineKeyboard[0][0].CallbackData += strconv.FormatInt(userIDs[1-i], 10)
		keyboard.InlineKeyboard[0][1].CallbackData += strconv.FormatInt(userIDs[1-i], 10)
		_, err := b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      id,
			Text:        "Оцените собеседника.",
			ReplyMarkup: keyboard,
		})
		if err != nil {
			slog.Error("sendRatingKeyboard", "error", err, "chat_id", id)
		}
	}
}

func (h *TelegramHandler) NextHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID := update.Message.Chat.ID
	res, err := h.service.Next(userID)
	if err != nil {
		slog.Error("NextHandler", "error", err, "user_id", userID)
		return
	}
	sendMessages(ctx, b, res)
	if res.ChatEnded {
		sendRatingKeyboard(ctx, b, res.UserIDs)
	}
}

func (h *TelegramHandler) StopHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID := update.Message.Chat.ID
	res, err := h.service.Stop(userID)
	if err != nil {
		slog.Error("StopHandler", "error", err, "user_id", userID)
		return
	}
	sendMessages(ctx, b, res)
	if res.ChatEnded {
		sendRatingKeyboard(ctx, b, res.UserIDs)
	}
}

func changeRatingKeyboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int) {
	_, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        "Спасибо за оценку.",
		ReplyMarkup: nil,
	})
	if err != nil {
		slog.Error("changeRatingKeyboard", "error", err, "chat_id", chatID)
	}
}

func (h *TelegramHandler) CallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})
	chatID := update.CallbackQuery.Message.Message.Chat.ID
	if err != nil {
		slog.Error("CallbackHandler_AnswerCallbackQuery", "error", err, "chat_id", chatID)
		return
	}

	data := update.CallbackQuery.Data
	err = h.service.ChangeRating(data)
	if err != nil {
		slog.Error("CallbackHandler_ChangeRating", "error", err, "chat_id", chatID)
		return
	}
	messageID := update.CallbackQuery.Message.Message.ID
	changeRatingKeyboard(ctx, b, chatID, messageID)
}

func (h *TelegramHandler) MyChatMemberHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID := update.MyChatMember.From.ID
	partnerID := h.service.GetPartner(userID)
	if partnerID == 0 {
		return
	}
	res, err := h.service.ManageBlocking(userID)
	if err != nil {
		slog.Error("MyChatMemberHandler", "error", err, "partner_id", partnerID)
		return
	}
	sendMessages(ctx, b, res)
}
