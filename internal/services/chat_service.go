package services

import (
	"errors"
	"fmt"
	botmodels "go-telegrambot-test/internal/models"
	"go-telegrambot-test/internal/queue"
	"strconv"
	"strings"
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

func (s *ChatService) Next(userID int64) (ServiceResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := ServiceResult{}

	user, ok := s.users[userID]
	if !ok {
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Отправьте /start для входа."})
		return res, nil
	}

	if user.Banned {
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Доступ к боту временно ограничен."})
		return res, nil
	}
	resetDailyChats(user)
	if isRestricted(user) {
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Дневной лимит чатов исчерпан."})
		return res, nil
	}

	if user.State == botmodels.StatePaired {
		partner, ok := s.users[user.PartnerID]
		if !ok {
			user.State = botmodels.StateIdle
			user.PartnerID = 0
			return res, nil
		}
		unpair(user, partner)
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Вы завершили чат.\nДля поиска собеседника отправьте /next."})
		res.Messages = append(res.Messages, BotMessage{ChatID: partner.ID, Message: "Собеседник завершил чат.\nДля поиска собеседника отправьте /next."})
		res.ChatEnded = true
		res.UserIDs = []int64{userID, partner.ID}
	}

	// отправка /next до завершения поиска
	if user.State == botmodels.StateWaiting {
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Поиск собеседника..."})
		return res, nil
	}

	if !s.waitingQueue.IsEmpty() {
		partnerID, ok := s.waitingQueue.Dequeue()
		if !ok {
			return res, nil
		}
		partner, ok := s.users[partnerID]
		if !ok {
			user.State = botmodels.StateIdle
			user.PartnerID = 0
			return res, nil
		}
		pair(user, partner)

		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Собеседник найден"})
		res.Messages = append(res.Messages, BotMessage{ChatID: partnerID, Message: "Собеседник найден"})
		return res, nil
	}

	s.waitingQueue.Enqueue(userID)
	s.users[userID].State = botmodels.StateWaiting
	res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Поиск собеседника..."})
	return res, nil
}

func (s *ChatService) Stop(userID int64) (ServiceResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := ServiceResult{}

	user, ok := s.users[userID]
	if !ok {
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Отправьте /start для входа."})
		return res, nil
	}

	if user.Banned {
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Доступ к боту временно ограничен."})
		return res, nil
	}
	resetDailyChats(user)
	if isRestricted(user) {
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Дневной лимит чатов исчерпан."})
		return res, nil
	}

	switch user.State {
	case botmodels.StateIdle:
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "У вас сейчас нет собеседника.\nДля поиска собеседника отправьте /next."})
	case botmodels.StateWaiting:
		s.waitingQueue.Remove(userID)
		user.State = botmodels.StateIdle
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Поиск собеседника прекращен."})
	case botmodels.StatePaired:
		partner, ok := s.users[user.PartnerID]
		if !ok {
			user.State = botmodels.StateIdle
			user.PartnerID = 0
			return res, nil
		}
		unpair(user, partner)
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Вы завершили чат.\nДля поиска собеседника отправьте /next."})
		res.Messages = append(res.Messages, BotMessage{ChatID: partner.ID, Message: "Собеседник завершил чат.\nДля поиска собеседника отправьте /next."})
		res.ChatEnded = true
		res.UserIDs = []int64{userID, partner.ID}
	}
	return res, nil
}

func (s *ChatService) Start(userID int64, username string) (ServiceResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := ServiceResult{}

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
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Добро пожаловать, " + username})
		return res, nil
	}

	res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Вы уже вошли.\nДля поиска собеседника отправьте /next"})
	return res, nil
}

func (s *ChatService) Default(userID int64, userMessage string) (ServiceResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := ServiceResult{}

	user, ok := s.users[userID]
	if !ok {
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Отправьте /start для входа."})
		return res, nil
	}

	if user.Banned {
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Доступ к боту временно ограничен."})
		return res, nil
	}
	resetDailyChats(user)
	if isRestricted(user) {
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Дневной лимит чатов исчерпан."})
		return res, nil
	}

	switch user.State {
	case botmodels.StateIdle:
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Вы не в диалоге.\nДля поиска собеседника отправьте /next."})
	case botmodels.StateWaiting:
		res.Messages = append(res.Messages, BotMessage{ChatID: userID, Message: "Собеседник еще не найден."})
	case botmodels.StatePaired:
		partnerID := user.PartnerID
		res.Messages = append(res.Messages, BotMessage{ChatID: partnerID, Message: userMessage})

	}
	return res, nil
}

func (s *ChatService) ChangeRating(data string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	parts := strings.Split(data, ":")
	if len(parts) != 2 {
		return errors.New("incorrect CallbackData")
	}
	userID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return err
	}
	switch parts[0] {
	case "like":
		s.users[userID].Rating++
	case "dislike":
		s.users[userID].Rating--
	}
	fmt.Printf("ID: %d | Rating: %d\n", userID, s.users[userID].Rating)
	return nil
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

func (s *ChatService) GetPartner(userID int64) int64 {
	return s.users[userID].PartnerID
}

type BotMessage struct {
	ChatID  int64
	Message string
}

type ServiceResult struct {
	Messages  []BotMessage
	ChatEnded bool
	UserIDs   []int64
}
