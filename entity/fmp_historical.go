package entity

type FmpHistorical struct {
	Close         float32 `json:"close"`
	Open          float32 `json:"open"`
	High          float32 `json:"high"`
	Low           float32 `json:"low"`
	Volume        float32 `json:"volume"`
	ChangePercent float32 `json:"changePercent"`
}
