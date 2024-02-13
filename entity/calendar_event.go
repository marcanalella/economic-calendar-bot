package entity

type CalendarEvent struct {
	Date     string `json:"date"`
	Country  string `json:"country"`
	Event    string `json:"event"`
	Currency string `json:"currency"`
	Impact   string `json:"impact"`
}
