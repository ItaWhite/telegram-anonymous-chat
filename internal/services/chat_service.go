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
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Отправьте /start для входа"})
		return msgs, nil
	}

	// TODO проверка бана

	if user.State == models2.StatePaired {
		partner := s.users[user.PartnerID]
		user.PartnerID = 0
		partner.PartnerID = 0
		user.State, partner.State = models2.StateIdle, models2.StateIdle

		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Вы завершили чат"})
		msgs = append(msgs, BotMessage{ChatID: partner.ID, Message: "Собеседник завершил чат"})
	}

	msgs = append(msgs, BotMessage{ChatID: userID, Message: "Поиск собеседника..."})

	// отправка /next до завершения поиска
	if user.State == models2.StateWaiting {
		return msgs, nil
	}

	if !s.waitingQueue.IsEmpty() {
		partnerID, ok := s.waitingQueue.Dequeue()
		if !ok {
			return msgs, nil
		}

		// если повторно быстро отправить /next, другая горутина может вытащить этого же юзера из очереди
		if partnerID == userID {
			// TODO вытащить другого
			return msgs, nil
		}

		user.PartnerID = partnerID
		s.users[partnerID].PartnerID = userID
		s.users[partnerID].State, user.State = models2.StatePaired, models2.StatePaired

		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Собеседник найден"})
		msgs = append(msgs, BotMessage{ChatID: partnerID, Message: "Собеседник найден"})
		return msgs, nil
	}

	s.waitingQueue.Enqueue(userID)
	s.users[userID].State = models2.StateWaiting
	return msgs, nil
}

func (s *ChatService) Stop(userID int64) ([]BotMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var msgs []BotMessage

	user, ok := s.users[userID]
	if !ok {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Отправьте /start для входа"})
		return msgs, nil
	}

	// TODO проверка бана

	switch user.State {
	case models2.StateIdle:
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "У вас сейчас нет собеседника"})
	case models2.StateWaiting:
		s.waitingQueue.Remove(userID)
		user.State = models2.StateIdle
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Поиск собеседника прекращен"})
	case models2.StatePaired:
		partner := s.users[user.PartnerID]
		user.PartnerID = 0
		partner.PartnerID = 0
		user.State, partner.State = models2.StateIdle, models2.StateIdle
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Вы завершили чат"})
		msgs = append(msgs, BotMessage{ChatID: partner.ID, Message: "Собеседник завершил чат"})
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

	msgs = append(msgs, BotMessage{ChatID: userID, Message: "Вы уже вошли"})
	return msgs, nil
}

func (s *ChatService) Default(userID int64, userMessage string) ([]BotMessage, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var msgs []BotMessage

	user, ok := s.users[userID]
	if !ok {
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Отправьте /start для входа"})
		return msgs, nil
	}

	switch user.State {
	case models2.StateIdle:
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Для поиска собеседника отправьте /next"})
	case models2.StateWaiting:
		msgs = append(msgs, BotMessage{ChatID: userID, Message: "Собеседник еще не найден"})
	case models2.StatePaired:
		partnerID := user.PartnerID
		// TODO обработать ошибку при блокировке бота собеседником
		msgs = append(msgs, BotMessage{ChatID: partnerID, Message: userMessage})

	}
	return msgs, nil
}

type BotMessage struct {
	ChatID  int64
	Message string
}
