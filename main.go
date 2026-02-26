package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
)

var (
	users        = make(map[int64]*User)
	waitingQueue *WaitingQueue
	mu           sync.Mutex
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	waitingQueue = &WaitingQueue{}

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
	}

	b, err := bot.New(os.Getenv("TELEGRAM_BOT_API_KEY"), opts...)
	if err != nil {
		cancel()
		log.Fatal(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, startHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/next", bot.MatchTypeExact, nextHandler)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/stop", bot.MatchTypeExact, stopHandler)

	b.Start(ctx)
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	mu.Lock()

	if update.Message != nil {
		userID := update.Message.Chat.ID
		user := users[userID]

		user, ok := users[userID]
		if !ok {
			mu.Unlock()
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Отправьте /start для входа",
			})
			return
		}

		switch user.State {
		case StateIdle:
			mu.Unlock()
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Для поиска собеседника отправьте /next",
			})
			return
		case StateWaiting:
			mu.Unlock()
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Собеседник еще не найден",
			})
			return
		case StatePaired:
			partnerID := user.PartnerID
			// TODO обработать ошибку при блокировке бота
			mu.Unlock()
			b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: partnerID,
				Text:   update.Message.Text,
			})
			return
		}
		fmt.Println("несуществующее состояние")
		mu.Unlock()
	}
}

func startHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Добро пожаловать, *" + bot.EscapeMarkdown(update.Message.From.Username) + "*",
		ParseMode: models.ParseModeMarkdown,
	})
	_, ok := users[update.Message.From.ID]
	if !ok {
		users[update.Message.From.ID] = &User{
			ID:        update.Message.From.ID,
			State:     StateIdle,
			PartnerID: 0,
			Banned:    false,
		}
	}
}

func nextHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	mu.Lock()

	userID := update.Message.Chat.ID
	user := users[userID]

	user, ok := users[userID]
	if !ok {
		mu.Unlock()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Отправьте /start для входа",
		})
		return
	}

	// проверка бана

	// если в данный момент есть собеседник
	if user.State == StatePaired {
		partner := users[user.PartnerID]
		user.PartnerID = 0
		partner.PartnerID = 0
		user.State, partner.State = StateIdle, StateIdle

		mu.Unlock()

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Вы завершили чат",
		})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: partner.ID,
			Text:   "Собеседник завершил чат",
		})

		mu.Lock()
	}

	mu.Unlock()

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: userID,
		Text:   "Поиск собеседника...",
	})

	mu.Lock()

	// если несколько /next
	if user.State == StateWaiting {
		return
	}

	if !waitingQueue.IsEmpty() {
		partnerID, ok := waitingQueue.Dequeue()
		if !ok {
			mu.Unlock()
			return
		}

		// если быстро дважды нажать /next, другая горутина может успеть вытащить из очереди
		if partnerID == user.ID {
			mu.Unlock()
			return
		}

		user.PartnerID = partnerID
		users[partnerID].PartnerID = user.ID

		users[partnerID].State, user.State = StatePaired, StatePaired

		mu.Unlock()

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Собеседник найден",
		})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: user.PartnerID,
			Text:   "Собеседник найден",
		})
	} else { // если очередь ожидания пуста
		waitingQueue.Enqueue(userID)
		users[userID].State = StateWaiting
		mu.Unlock()
	}
}

func stopHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	mu.Lock()

	userID := update.Message.Chat.ID
	user := users[userID]

	user, ok := users[userID]
	if !ok {
		mu.Unlock()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Отправьте /start для входа",
		})
		return
	}

	// проверка бана

	switch user.State {
	case StateIdle:
		mu.Unlock()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "У вас сейчас нет собеседника",
		})
		return
	case StateWaiting:
		waitingQueue.Remove(userID)
		user.State = StateIdle

		mu.Unlock()
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Поиск собеседника прекращен",
		})
		return
	case StatePaired:
		partner := users[user.PartnerID]
		user.PartnerID = 0
		partner.PartnerID = 0
		user.State, partner.State = StateIdle, StateIdle

		mu.Unlock()

		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: userID,
			Text:   "Вы завершили чат",
		})
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: partner.ID,
			Text:   "Собеседник завершил чат",
		})
	}
}

type WaitingQueue struct {
	users []int64
}

func (w *WaitingQueue) IsEmpty() bool {
	return len(w.users) == 0
}

func (w *WaitingQueue) Dequeue() (int64, bool) {
	if len(w.users) == 0 {
		return 0, false
	}

	userID := w.users[0]
	w.users = w.users[1:]
	return userID, true
}

func (w *WaitingQueue) Enqueue(userID int64) {
	w.users = append(w.users, userID)
}

func (w *WaitingQueue) Remove(userID int64) {
	filtered := w.users[:0]

	for _, id := range w.users {
		if id != userID {
			filtered = append(filtered, id)
		}
	}

	w.users = filtered
}

type User struct {
	ID        int64
	State     string // idle, waiting, paired
	PartnerID int64
	Banned    bool
}

const (
	StateIdle    = "idle"
	StateWaiting = "waiting"
	StatePaired  = "paired"
)
