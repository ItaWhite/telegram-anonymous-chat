package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
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

	dailyChatLimit, err := strconv.Atoi(os.Getenv("DAILY_CHAT_LIMIT"))
	if err != nil {
		log.Println("Суточный лимит не задан")
		dailyChatLimit = 20
	}
	lowRatingLimit, err := strconv.Atoi(os.Getenv("LOW_RATING_LIMIT"))
	if err != nil {
		log.Println("Нижняя граница рейтинга не задана")
		lowRatingLimit = -10
	}

	s := services.NewChatService(dailyChatLimit, lowRatingLimit)
	h := handlers.NewTelegramHandler(s)

	opts := []bot.Option{
		bot.WithDefaultHandler(h.DefaultHandler),
	}

	b, err := bot.New(os.Getenv("TELEGRAM_BOT_API_KEY"), opts...)
	if err != nil {
		cancel()
		log.Fatal(err)
	}

	handlers.RegisterHandlers(b, h)

	b.Start(ctx)
}
