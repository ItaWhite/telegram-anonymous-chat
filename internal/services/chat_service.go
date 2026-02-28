package services

import (
	models2 "go-telegrambot-test/internal/models"
	"go-telegrambot-test/internal/queue"
	"sync"
)

type ChatService struct {
	users        map[int64]*models2.User
	waitingQueue *queue.WaitingQueue
	mu           sync.Mutex
}

func NewChatService() *ChatService {
	s := &ChatService{}
	s.users = make(map[int64]*models2.User)
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

	if user.State == models2.StatePaired {
		partner, ok := s.users[user.PartnerID]
		if !ok {
			user.State = models2.StateIdle
			user.PartnerID = 0
			return msgs, nil
		}
		unpair(user, partner)
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Вы завершили чат."})
		msgs = append(msgs, BotMessage{ChatID: partner.ID, Message: "Собеседник завершил чат."})
	}

	// отправка /next до завершения поиска
	if user.State == models2.StateWaiting {
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
			user.State = models2.StateIdle
			user.PartnerID = 0
			return msgs, nil
		}
		pair(user, partner)

		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Собеседник найден"})
		msgs = append(msgs, BotMessage{ChatID: partnerID, Message: "Собеседник найден"})
		return msgs, nil
	}

	s.waitingQueue.Enqueue(userID)
	s.users[userID].State = models2.StateWaiting
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

	switch user.State {
	case models2.StateIdle:
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "У вас сейчас нет собеседника.\nДля поиска собеседника отправьте /next."})
	case models2.StateWaiting:
		s.waitingQueue.Remove(userID)
		user.State = models2.StateIdle
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Поиск собеседника прекращен."})
	case models2.StatePaired:
		partner, ok := s.users[user.PartnerID]
		if !ok {
			user.State = models2.StateIdle
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
		s.users[userID] = &models2.User{
			ID:        userID,
			State:     models2.StateIdle,
			PartnerID: 0,
			Banned:    false,
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

	switch user.State {
	case models2.StateIdle:
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Вы не в диалоге.\nДля поиска собеседника отправьте /next."})
	case models2.StateWaiting:
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Собеседник еще не найден."})
	case models2.StatePaired:
		partnerID := user.PartnerID
		msgs = append(msgs, BotMessage{ChatID: partnerID, Message: userMessage})

	}
	return msgs, nil
}

func pair(u1, u2 *models2.User) {
	u1.PartnerID = u2.ID
	u2.PartnerID = u1.ID
	u1.State, u2.State = models2.StatePaired, models2.StatePaired
}

func unpair(u1, u2 *models2.User) {
	u1.PartnerID = 0
	u2.PartnerID = 0
	u1.State, u2.State = models2.StateIdle, models2.StateIdle
}

type BotMessage struct {
	ChatID  int64
	Message string
}
