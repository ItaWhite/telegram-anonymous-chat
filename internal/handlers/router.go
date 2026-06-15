package handlers

import (
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func matchFunc(update *models.Update) bool {
	return update.MyChatMember != nil
}

func RegisterHandlers(b *bot.Bot, h *TelegramHandler) {
	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, h.StartHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/next", bot.MatchTypeExact, h.NextHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stop", bot.MatchTypeExact, h.StopHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "", bot.MatchTypePrefix, h.CallbackHandler)
	b.RegisterHandlerMatchFunc(matchFunc, h.MyChatMemberHandler)
}
