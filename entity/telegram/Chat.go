package telegram

import "fmt"

// A Chat indicates the conversation to which the Message belongs.
type Chat struct {
	Id int `json:"id"`
}

// Implements the fmt.String interface to get the representation of a Chat as a string.
func (c Chat) String() string {
	return fmt.Sprintf("(id: %d)", c.Id)
}
