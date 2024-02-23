package telegram

type Recipient struct {
	ChatId          int `json:"chatId"`
	MessageThreadId int `json:"messageThreadId"`
}
