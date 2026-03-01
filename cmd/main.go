package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"telegram-anonymous-chat/internal/handlers"
	"telegram-anonymous-chat/internal/services"

	"github.com/go-telegram/bot"
	"github.com/joho/godotenv"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	s := services.NewChatService()
	h := handlers.NewTelegramHandler(s)

	opts := []bot.Option{
		bot.WithDefaultHandler(h.DefaultHandler),
	}

	b, err := bot.New(os.Getenv("TELEGRAM_BOT_API_KEY"), opts...)
	if err != nil {
		cancel()
		log.Fatal(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, h.StartHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/next", bot.MatchTypeExact, h.NextHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stop", bot.MatchTypeExact, h.StopHandler)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "", bot.MatchTypePrefix, h.CallbackHandler)

	b.Start(ctx)
}
