package services

import (
	botmodels "go-telegrambot-test/internal/models"
	"go-telegrambot-test/internal/queue"
	"sync"
	"time"
)

const DailyChatLimit = 20

type ChatService struct {
	users        map[int64]*botmodels.User
	waitingQueue *queue.WaitingQueue
	mu           sync.Mutex
}

func NewChatService() *ChatService {
	s := &ChatService{}
	s.users = make(map[int64]*botmodels.User)
	s.waitingQueue = &queue.WaitingQueue{}
	s.mu = sync.Mutex{}
	return s
}

func (s *ChatService) Next(userID int64) ([]BotMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var msgs []BotMessage

	user, ok := s.users[userID]
	if !ok {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Отправьте /start для входа."})
		return msgs, nil
	}

	if user.Banned {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Доступ к боту временно ограничен."})
		return msgs, nil
	}
	resetDailyChats(user)
	if isRestricted(user) {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Дневной лимит чатов исчерпан."})
		return msgs, nil
	}

	if user.State == botmodels.StatePaired {
		partner, ok := s.users[user.PartnerID]
		if !ok {
			user.State = botmodels.StateIdle
			user.PartnerID = 0
			return msgs, nil
		}
		unpair(user, partner)
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Вы завершили чат."})
		msgs = append(msgs, BotMessage{ChatID: partner.ID, Message: "Собеседник завершил чат."})
	}

	// отправка /next до завершения поиска
	if user.State == botmodels.StateWaiting {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Поиск собеседника..."})
		return msgs, nil
	}

	if !s.waitingQueue.IsEmpty() {
		partnerID, ok := s.waitingQueue.Dequeue()
		if !ok {
			return msgs, nil
		}
		partner, ok := s.users[partnerID]
		if !ok {
			user.State = botmodels.StateIdle
			user.PartnerID = 0
			return msgs, nil
		}
		pair(user, partner)

		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Собеседник найден"})
		msgs = append(msgs, BotMessage{ChatID: partnerID, Message: "Собеседник найден"})
		return msgs, nil
	}

	s.waitingQueue.Enqueue(userID)
	s.users[userID].State = botmodels.StateWaiting
	msgs = append(msgs, BotMessage{ChatID: userID, Message: "Поиск собеседника..."})
	return msgs, nil
}

func (s *ChatService) Stop(userID int64) ([]BotMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var msgs []BotMessage

	user, ok := s.users[userID]
	if !ok {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Отправьте /start для входа."})
		return msgs, nil
	}

	if user.Banned {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Доступ к боту временно ограничен."})
		return msgs, nil
	}
	resetDailyChats(user)
	if isRestricted(user) {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Дневной лимит чатов исчерпан."})
		return msgs, nil
	}

	switch user.State {
	case botmodels.StateIdle:
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "У вас сейчас нет собеседника.\nДля поиска собеседника отправьте /next."})
	case botmodels.StateWaiting:
		s.waitingQueue.Remove(userID)
		user.State = botmodels.StateIdle
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Поиск собеседника прекращен."})
	case botmodels.StatePaired:
		partner, ok := s.users[user.PartnerID]
		if !ok {
			user.State = botmodels.StateIdle
			user.PartnerID = 0
			return msgs, nil
		}
		unpair(user, partner)
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Вы завершили чат.\nДля поиска собеседника отправьте /next."})
		msgs = append(msgs, BotMessage{ChatID: partner.ID, Message: "Собеседник завершил чат.\nДля поиска собеседника отправьте /next."})
	}
	return msgs, nil
}

func (s *ChatService) Start(userID int64, username string) ([]BotMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var msgs []BotMessage

	_, ok := s.users[userID]
	if !ok {
		s.users[userID] = &botmodels.User{
			ID:         userID,
			State:      botmodels.StateIdle,
			PartnerID:  0,
			Banned:     false,
			Rating:     10,
			DailyChats: 0,
			LastReset:  time.Now(),
		}
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Добро пожаловать, " + username})
		return msgs, nil
	}

	msgs = append(msgs, BotMessage{ChatID: userID, Message: "Вы уже вошли.\nДля поиска собеседника отправьте /next"})
	return msgs, nil
}

func (s *ChatService) Default(userID int64, userMessage string) ([]BotMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var msgs []BotMessage

	user, ok := s.users[userID]
	if !ok {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Отправьте /start для входа."})
		return msgs, nil
	}

	if user.Banned {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Доступ к боту временно ограничен."})
		return msgs, nil
	}
	resetDailyChats(user)
	if isRestricted(user) {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Дневной лимит чатов исчерпан."})
		return msgs, nil
	}

	switch user.State {
	case botmodels.StateIdle:
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Вы не в диалоге.\nДля поиска собеседника отправьте /next."})
	case botmodels.StateWaiting:
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Собеседник еще не найден."})
	case botmodels.StatePaired:
		partnerID := user.PartnerID
		msgs = append(msgs, BotMessage{ChatID: partnerID, Message: userMessage})

	}
	return msgs, nil
}

func pair(u1, u2 *botmodels.User) {
	u1.PartnerID = u2.ID
	u2.PartnerID = u1.ID
	u1.State, u2.State = botmodels.StatePaired, botmodels.StatePaired
}

func unpair(u1, u2 *botmodels.User) {
	u1.PartnerID = 0
	u2.PartnerID = 0
	u1.DailyChats++
	u2.DailyChats++
	u1.State, u2.State = botmodels.StateIdle, botmodels.StateIdle
}

func resetDailyChats(u *botmodels.User) {
	if time.Since(u.LastReset) >= 24*time.Hour {
		u.LastReset = time.Now()
		u.DailyChats = 0
	}
}

func isRestricted(u *botmodels.User) bool {
	return u.Rating < 0 && u.DailyChats >= DailyChatLimit
}

type BotMessage struct {
	ChatID  int64
	Message string
}
