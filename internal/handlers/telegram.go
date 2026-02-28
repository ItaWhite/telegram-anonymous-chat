package handlers

import (
	"context"
	"go-telegrambot-test/internal/services"
	"log"

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
	msgs, err := h.service.Default(userID, update.Message.Text)
	if err != nil {
		log.Fatal(err)
	}

	sendMessages(ctx, b, msgs)
}

func (h *TelegramHandler) StartHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID := update.Message.From.ID
	username := update.Message.From.Username
	msgs, err := h.service.Start(userID, username)
	if err != nil {
		log.Fatal(err)
	}

	sendMessages(ctx, b, msgs)
}

func (h *TelegramHandler) NextHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID := update.Message.Chat.ID
	msgs, err := h.service.Next(userID)
	if err != nil {
		log.Fatal(err)
	}

	sendMessages(ctx, b, msgs)
}

func (h *TelegramHandler) StopHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	userID := update.Message.Chat.ID
	msgs, err := h.service.Stop(userID)
	if err != nil {
		log.Fatal(err)
	}

	sendMessages(ctx, b, msgs)
}

func sendMessages(ctx context.Context, b *bot.Bot, msgs []services.BotMessage) {
	for _, msg := range msgs {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: msg.ChatID,
			Text:   msg.Message,
		})
	}
}
