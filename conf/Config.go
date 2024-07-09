package conf

import (
	"bot/entity/telegram"
	"encoding/json"
	"log"
	"os"
)

type Config struct {
	Address                  string `json:"address"`
	Port                     string `json:"port"`
	TelegramBotToken         string `json:"telegram_bot_token"`
	TelegramApiBaseUrl       string `json:"telegram_api_base_url"`
	TelegramApiSendMessage   string `json:"telegram_api_send_message"`
	EconomicCalendarUrl      string `json:"economic_calendar_url"`
	EconomicCalendarApyKey   string `json:"economic_calendar_apy_key"`
	FinancialModelingPrepUrl string `json:"financial_modeling_prep_url"`
	SheetId                  int    `json:"sheet_id"`
	SpreadsheetId            string `json:"spread_sheet_id"`
	ReadRange                string `json:"read_range"`
	WriteRange               string `json:"write_range"`
	KeyFile                  string `json:"key_file"`
	RecipientsFile           string `json:"recipients_file"`
}

func Load() (Config, error) {
	var config Config
	configFile, err := os.Open("config.json")
	defer func(configFile *os.File) {
		err := configFile.Close()
		if err != nil {
			log.Printf("could not decode json config %s\n", err.Error())
		}
	}(configFile)
	jsonParser := json.NewDecoder(configFile)
	err = jsonParser.Decode(&config)
	return config, err
}

func LoadRecipients(pathFile string) ([]telegram.Recipient, error) {
	var arr []telegram.Recipient
	recipientsFile, err := os.Open(pathFile)
	defer func(configFile *os.File) {
		err := configFile.Close()
		if err != nil {
			log.Printf("could not decode json recipients %s\n", err.Error())
		}
	}(recipientsFile)
	jsonParser := json.NewDecoder(recipientsFile)
	err = jsonParser.Decode(&arr)
	return arr, err
}
