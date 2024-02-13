package telegram

import "fmt"

// Document Message refer to a file sent.
type Document struct {
	FileId   string `json:"file_id"`
	FileName string `json:"file_name"`
}

// Implements the fmt.String interface to get the representation of an Document as a string.
func (d Document) String() string {
	return fmt.Sprintf("(file id: %s, file name: %s)", d.FileId, d.FileName)
}
