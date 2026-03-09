package services

type BotMessage struct {
	ChatID  int64
	Message string
}

type ServiceResult struct {
	Messages  []BotMessage
	ChatEnded bool
	UserIDs   []int64
}
