package internal

import (
	"bot/conf"
	"bot/entity"
	"bot/entity/telegram"
	"encoding/json"
	"fmt"
	"github.com/enescakir/emoji"
	"github.com/go-co-op/gocron"
	"google.golang.org/api/sheets/v4"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Service interface {
	GetEconomicCalendarForNextDay(tomorrowDate time.Time) ([]entity.CalendarEvent, error)

	GetXauRateFromYesterday(url string) (entity.FmpResponse, error)

	PrepareEconomicCalendarForNextDayMessage(tomorrowDate time.Time, events []entity.CalendarEvent) string

	SendTextToTelegramChat(chatId int, messageThreadId int, text string) (string, error)

	ScheduledNewsNotification(recipients []telegram.Recipient)

	ScheduledXauNotification(recipients []telegram.Recipient, spreadsheetId string, readRange string, sheetService *sheets.Service)

	ScheduledXauSheetUpdate(recipients []telegram.Recipient, spreadsheetId string, readRange string, sheetId int, url string, sheetService *sheets.Service)

	Readyz(recipients []telegram.Recipient)
}

type service struct {
	config conf.Config
}

func NewService(config conf.Config) Service {
	return service{config}
}

func (s service) GetEconomicCalendarForNextDay(tomorrowDate time.Time) ([]entity.CalendarEvent, error) {

	u, err := url.Parse(s.config.EconomicCalendarUrl)
	if err != nil {
		log.Fatal(err)
	}

	formattedTomorrowDate := tomorrowDate.Format("2006-01-02")
	formattedCurrentDate := time.Now().Format("2006-01-02")

	q := u.Query()
	q.Set("from", formattedCurrentDate)
	q.Set("to", formattedTomorrowDate)
	q.Set("apikey", s.config.EconomicCalendarApyKey)

	u.RawQuery = q.Encode()
	log.Println("Calling " + u.String())

	response, err := http.Get(u.String())
	if err != nil {
		log.Printf("error while calling Economic Calendar %s", err.Error())
		return []entity.CalendarEvent{}, err
	}
	log.Println(response.Status)

	var events []entity.CalendarEvent
	body, err := io.ReadAll(response.Body)
	if err := json.Unmarshal(body, &events); err != nil {
		log.Printf("error while parsing Economic Calendar response %s", err.Error())
		panic(err)
	}

	return events, nil
}

func (s service) GetXauRateFromYesterday(providerUrl string) (entity.FmpResponse, error) {

	u, err := url.Parse(providerUrl)
	if err != nil {
		log.Fatal(err)
	}

	response, err := http.Get(u.String())
	if err != nil {
		log.Printf("error while calling Economic Calendar %s", err.Error())
		return entity.FmpResponse{}, err
	}
	log.Println(response.Status)

	var fmpResponse entity.FmpResponse
	body, err := io.ReadAll(response.Body)
	if err := json.Unmarshal(body, &fmpResponse); err != nil {
		log.Printf("error while parsing Economic Calendar response %s", err.Error())
		panic(err)
	}

	return fmpResponse, nil
}

func (s service) SendTextToTelegramChat(chatId int, messageThreadId int, text string) (string, error) {

	log.Printf("Sending %s to chat_id: %d", text, chatId)
	response, err := http.PostForm(
		s.config.TelegramApiBaseUrl+s.config.TelegramApiSendMessage,
		url.Values{
			"chat_id":           {strconv.Itoa(chatId)},
			"message_thread_id": {strconv.Itoa(messageThreadId)},
			"via_bot":           {"@EconomicCalendarAndNewsBot"},
			"text":              {text},
		})

	if err != nil {
		log.Printf("error when posting text to the chat: %s", err.Error())
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Printf("error when posting text to the chat: %s", err.Error())
		}
	}(response.Body)

	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		log.Printf("error in parsing telegram answer %s", err.Error())
		return "", err
	}
	bodyString := string(bodyBytes)
	log.Printf("Body of Telegram Response: %s", bodyString)

	return bodyString, nil
}

func (s service) PrepareXauMessage(short float64, long float64) string {
	return emoji.Butter.String() + " XAUUSD " + time.Now().Weekday().String() + "statistics \n\n" +
		"LONG " + strconv.FormatFloat(long, 'f', -1, 64) + "% \n\n" +
		"SHORT " + strconv.FormatFloat(short, 'f', -1, 64) + "% \n\n"
}

func (s service) PrepareXauUpdateMessage() string {
	return emoji.Butter.String() + "XAUUSD DAILY FILE UPDATED :) "
}

func (s service) PrepareEconomicCalendarForNextDayMessage(tomorrowDate time.Time, events []entity.CalendarEvent) string {

	formattedTomorrowDate := tomorrowDate.Format("2006-01-02")
	message := "Calendario Economico del " + formattedTomorrowDate + " \n\n"

	if len(events) == 0 {
		message = message + "Nessuna Notizia Rilevante :("
	} else {
		for _, e := range events {
			message = message +
				emoji.Calendar.String() + "  DATE: " + e.Date + "\n" +
				emoji.Megaphone.String() + "  EVENT: " + e.Event + "\n" +
				emoji.GlobeShowingEuropeAfrica.String() + "  COUNTRY: " + e.Country + "  " + GetEmojiCountry(e.Country) + "\n" +
				emoji.CurrencyExchange.String() + "  CURRENCY: " + e.Currency + "\n" +
				emoji.VerticalTrafficLight.String() + "  IMPACT: " + e.Impact + "  " + GetEmojiSemaphore(e.Impact) + "\n\n"
		}
	}
	return message
}

func GetEmojiCountry(country string) string {
	switch country {
	case "UK":
		return emoji.FlagForUnitedKingdom.String()
	case "US":
		return emoji.FlagForUnitedStates.String()
	case "JP":
		return emoji.FlagForJapan.String()
	case "EU":
		return emoji.FlagForEuropeanUnion.String()
	default:
		return ""
	}
}

func GetEmojiSemaphore(impact string) string {
	switch impact {
	case "Low":
		return emoji.GreenCircle.String()
	case "Medium":
		return emoji.YellowCircle.String()
	case "High":
		return emoji.RedCircle.String()
	default:
		return ""
	}
}

func (s service) ScheduledNewsNotification(recipients []telegram.Recipient) {
	var message string
	s1 := gocron.NewScheduler(time.UTC)
	_, err := s1.Every(1).Day().At("22:00").Do(func() {
		// Add 1 day to the current date to get tomorrow's date
		tomorrowDate := time.Now().AddDate(0, 0, 1)

		events, err := s.GetEconomicCalendarForNextDay(tomorrowDate)
		if err != nil {
			log.Printf("got error when calling Economic Calendar API %s", err.Error())
			return
		}

		var eventsFiltered []entity.CalendarEvent
		for _, e := range events {
			if e.Currency == "EUR" || e.Currency == "GBP" || e.Currency == "USD" || e.Currency == "JPY" {
				if e.Impact == "High" { //|| e.Impact == "Medium" {

					parsedDate, err := time.Parse("2006-01-02 15:04:05", e.Date)
					if err != nil {
						fmt.Println("Error parsing date:", err)
						return
					}

					if tomorrowDate.Year() == parsedDate.Year() &&
						tomorrowDate.Month() == parsedDate.Month() &&
						tomorrowDate.Day() == parsedDate.Day() {
						eventsFiltered = append(eventsFiltered, e)
					}
				}
			}
		}

		message = s.PrepareEconomicCalendarForNextDayMessage(tomorrowDate, eventsFiltered)

		for _, recipient := range recipients {
			// Send the punchline back to Telegram
			log.Printf("send to chatId, %s", strconv.Itoa(recipient.ChatId))
			telegramResponseBody, err := s.SendTextToTelegramChat(recipient.ChatId, recipient.MessageThreadId, message)
			if err != nil {
				log.Printf("got error %s from telegram, response body is %s", err.Error(), telegramResponseBody)
			} else {
				log.Printf("economic calendar successfully distributed to chat id %d", recipient.ChatId)
			}
		}

		message = ""
	})
	s1.StartAsync()
	if err != nil {
		log.Printf("error creating job: %v", err)
	}
	_, t := s1.NextRun()
	log.Printf("next run at: %s", t)
}

func (s service) ScheduledXauSheetUpdate(recipients []telegram.Recipient, spreadsheetId string,
	writeRange string, sheetId int, url string, sheetService *sheets.Service) {
	var message string
	s1 := gocron.NewScheduler(time.UTC)
	_, err := s1.Every(1).Day().Do(func() {
		if time.Now().Weekday() != 0 {

			insertRequest := &sheets.Request{
				InsertDimension: &sheets.InsertDimensionRequest{
					Range: &sheets.DimensionRange{
						SheetId:    int64(sheetId),
						Dimension:  "ROWS",
						StartIndex: 1,
						EndIndex:   2,
					},
					InheritFromBefore: false,
				},
			}

			batchUpdateRequest := &sheets.BatchUpdateSpreadsheetRequest{
				Requests: []*sheets.Request{insertRequest},
			}

			_, err := sheetService.Spreadsheets.BatchUpdate(spreadsheetId, batchUpdateRequest).Do()
			if err != nil {
				log.Fatalf("Unable to insert row: %v", err)
			}

			response, err := s.GetXauRateFromYesterday(url)
			if err != nil {
				log.Fatalf("Unable to get xau data: %v", err)
			}
			yesterday := time.Now().AddDate(0, 0, -1)
			formattedYesterday := yesterday.Format("02/01/2006")

			// Definisci i dati da inserire
			values := []interface{}{
				formattedYesterday,
				response.Historical[1].Close,
				response.Historical[1].Open,
				response.Historical[1].High,
				response.Historical[1].Low,
				response.Historical[1].Volume,
				0,
				//response.Historical[1].ChangePercent, //TODO
				"@EconomicCalendarAndNewsBot"}

			valueRange := &sheets.ValueRange{
				Range:  writeRange,
				Values: [][]interface{}{values},
			}

			// Inserisci i dati nella nuova riga
			_, err = sheetService.Spreadsheets.Values.Update(spreadsheetId, valueRange.Range, valueRange).ValueInputOption("RAW").Do()
			if err != nil {
				log.Fatalf("Unable to update data: %v", err)
			}

			fmt.Println("Riga inserita con successo alla seconda posizione")

			message = s.PrepareXauUpdateMessage()
			log.Printf(message)
			for _, recipient := range recipients {
				// Send the punchline back to Telegram
				log.Printf("send to chatId, %s", strconv.Itoa(recipient.ChatId))
				telegramResponseBody, err := s.SendTextToTelegramChat(recipient.ChatId, recipient.MessageThreadId, message)
				if err != nil {
					log.Printf("got error %s from telegram, response body is %s", err.Error(), telegramResponseBody)
				} else {
					log.Printf("xau history successfully distributed to chat id %d", recipient.ChatId)
				}
			}

			message = ""
		}
	})
	s1.StartAsync()
	if err != nil {
		log.Printf("error creating job: %v", err)
	}
	_, t := s1.NextRun()
	log.Printf("next run at: %s", t)
}

func (s service) ScheduledXauNotification(
	recipients []telegram.Recipient,
	spreadsheetId string, readRange string, sheetService *sheets.Service) {
	var message string
	s1 := gocron.NewScheduler(time.UTC)
	_, err := s1.Every(1).Day().Do(func() {

		resp, err := sheetService.Spreadsheets.Values.Get(spreadsheetId, readRange).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve data from sheet: %v", err)
		}

		total := 0
		long := 0
		short := 0

		for i, row := range resp.Values {

			if i == 0 {
				continue
			}

			dateStr := row[0].(string)
			closeStr := row[1].(string)
			openStr := row[2].(string)

			date, err := time.Parse("02/01/2006", dateStr)
			if err != nil {
				log.Printf("%v", dateStr)
				log.Printf("Unable to parse date: %v", err)
			}

			closeFloat, _ := strconv.ParseFloat(strings.TrimSpace(closeStr), 64)
			openFloat, _ := strconv.ParseFloat(strings.TrimSpace(openStr), 64)

			// Controlla se è oggi
			if date.Weekday() == time.Now().Weekday() {
				if openFloat > closeFloat {
					short++
				} else {
					long++
				}
			}
		}

		total = long + short

		longPer := (float64((long) / total)) * 100
		shortPer := (float64((short) / total)) * 100

		message = s.PrepareXauMessage(longPer, shortPer)
		log.Printf(message)
		for _, recipient := range recipients {
			// Send the punchline back to Telegram
			log.Printf("send to chatId, %s", strconv.Itoa(recipient.ChatId))
			telegramResponseBody, err := s.SendTextToTelegramChat(recipient.ChatId, recipient.MessageThreadId, message)
			if err != nil {
				log.Printf("got error %s from telegram, response body is %s", err.Error(), telegramResponseBody)
			} else {
				log.Printf("xau history successfully distributed to chat id %d", recipient.ChatId)
			}
		}

		message = ""
	})
	s1.StartAsync()
	if err != nil {
		log.Printf("error creating job: %v", err)
	}
	_, t := s1.NextRun()
	log.Printf("next run at: %s", t)
}

func (s service) Readyz(recipients []telegram.Recipient) {
	var message string
	s2 := gocron.NewScheduler(time.UTC)
	_, err := s2.Every(1).Day().At("21:59").Do(func() {
		message = "EconomicCalendarAndNewsBot Running " + emoji.BeamingFaceWithSmilingEyes.String()
		for _, recipient := range recipients {
			// Send the punchline back to Telegram
			log.Printf("send to chatId, %s", strconv.Itoa(recipient.ChatId))
			telegramResponseBody, err := s.SendTextToTelegramChat(recipient.ChatId, recipient.MessageThreadId, message)
			if err != nil {
				log.Printf("got error %s from telegram, response body is %s", err.Error(), telegramResponseBody)
			} else {
				log.Printf("Readyz successfully distributed to chat id %d", recipient.ChatId)
			}
		}
	})
	s2.StartAsync()
	if err != nil {
		log.Printf("error creating job: %v", err)
	}
	_, t := s2.NextRun()
	log.Printf("next run at: %s", t)
}
