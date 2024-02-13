package telegram

import "fmt"

// Audio message has extra attributes
type Audio struct {
	FileId   string `json:"file_id"`
	Duration int    `json:"duration"`
}

// Implements the fmt.String interface to get the representation of an Audio as a string.
func (a Audio) String() string {
	return fmt.Sprintf("(file id: %s, duration: %d)", a.FileId, a.Duration)
}
