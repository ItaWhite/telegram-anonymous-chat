package handlers

import (
	"context"
	"fmt"
	"go-telegrambot-test/internal/services"
	"strconv"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type TelegramHandler struct {
	service *services.ChatService
}

func NewTelegramHandler(s *services.ChatService) *TelegramHandler {
	return &TelegramHandler{service: s}
}

func (h *TelegramHandler) DefaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil {
		return
	}

	userID := update.Message.From.ID
	res, err := h.service.Default(userID, update.Message.Text)
	if err != nil {
		fmt.Println(err)
	}

	sendMessages(ctx, b, res)
	partnerID := h.service.GetPartner(userID)
	if partnerID == 0 {
		return
	}
	if update.Message.Photo != nil && len(update.Message.Photo) != 0 { // если отправлено фото
		h.handlePhoto(ctx, b, update)
	}
	if update.Message.Video != nil {
		h.handleVideo(ctx, b, update)
	}
}

func (h *TelegramHandler) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	username := update.Message.From.Username
	res, err := h.service.Start(userID, username)
	if err != nil {
		fmt.Println(err)
	}

	sendMessages(ctx, b, res)
}

func (h *TelegramHandler) NextHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID := update.Message.Chat.ID
	res, err := h.service.Next(userID)
	if err != nil {
		fmt.Println(err)
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
		fmt.Println(err)
	}
	sendMessages(ctx, b, res)
	if res.ChatEnded {
		sendRatingKeyboard(ctx, b, res.UserIDs)
	}
}

func sendMessages(ctx context.Context, b *bot.Bot, res services.ServiceResult) {
	for _, msg := range res.Messages {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.ChatID,
			Text:   msg.Message,
		})
	}
}

func (h *TelegramHandler) CallbackHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	chatID := update.CallbackQuery.Message.Message.Chat.ID
	data := update.CallbackQuery.Data
	err := h.service.ChangeRating(data)
	if err != nil {
		fmt.Println(err)
	}
	messageID := update.CallbackQuery.Message.Message.ID
	changeRatingKeyboard(ctx, b, chatID, messageID)
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
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:      id,
			Text:        "Оцените собеседника.",
			ReplyMarkup: keyboard,
		})
	}
}

func changeRatingKeyboard(ctx context.Context, b *bot.Bot, chatID int64, messageID int) {
	b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        "Спасибо за оценку.",
		ReplyMarkup: nil,
	})
}

func (h *TelegramHandler) handlePhoto(ctx context.Context, b *bot.Bot, update *models.Update) {
	photo := update.Message.Photo[len(update.Message.Photo)-1]
	userID := update.Message.From.ID
	partnerID := h.service.GetPartner(userID)
	if partnerID == 0 {
		return
	}

	b.SendPhoto(ctx, &bot.SendPhotoParams{
		ChatID:  partnerID,
		Photo:   &models.InputFileString{Data: photo.FileID},
		Caption: update.Message.Caption,
	})
}

func (h *TelegramHandler) handleVideo(ctx context.Context, b *bot.Bot, update *models.Update) {
	video := update.Message.Video
	userID := update.Message.From.ID
	partnerID := h.service.GetPartner(userID)

	b.SendVideo(ctx, &bot.SendVideoParams{
		ChatID:  partnerID,
		Video:   &models.InputFileString{Data: video.FileID},
		Caption: update.Message.Caption,
	})
}
