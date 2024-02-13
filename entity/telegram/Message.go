package telegram

import "fmt"

// Message is a Telegram object that can be found in an update.
// Note that not all Update contains a Message. Update for an Inline Query doesn't.
type Message struct {
	Text     string   `json:"text"`
	Chat     Chat     `json:"chat"`
	Audio    Audio    `json:"audio"`
	Voice    Voice    `json:"voice"`
	Document Document `json:"document"`
}

// Implements the fmt.String interface to get the representation of a Message as a string.
func (m Message) String() string {
	return fmt.Sprintf("(text: %s, chat: %s, audio %s)", m.Text, m.Chat, m.Audio)
}
