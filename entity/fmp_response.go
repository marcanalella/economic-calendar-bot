package entity

type FmpResponse struct {
	Symbol     string          `json:"symbol"`
	Historical []FmpHistorical `json:"historical"`
}
