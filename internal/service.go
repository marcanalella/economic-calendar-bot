package internal

import (
	"bot/conf"
	"bot/entity"
	"bot/entity/telegram"
	"encoding/json"
	"fmt"
	"github.com/enescakir/emoji"
	"github.com/go-co-op/gocron"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Service interface {
	GetEconomicCalendarForNextDay(tomorrowDate time.Time) ([]entity.CalendarEvent, error)

	PrepareEconomicCalendarForNextDayMessage(tomorrowDate time.Time, events []entity.CalendarEvent) string

	SendTextToTelegramChat(chatId int, messageThreadId int, text string) (string, error)

	ScheduledNotification(recipients []telegram.Recipient)

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

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Printf("error in parsing telegram answer %s", err.Error())
		return "", err
	}
	bodyString := string(bodyBytes)
	log.Printf("Body of Telegram Response: %s", bodyString)

	return bodyString, nil
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

func (s service) ScheduledNotification(recipients []telegram.Recipient) {
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

func (s service) Readyz(recipients []telegram.Recipient) {
	var message string
	s2 := gocron.NewScheduler(time.UTC)
	_, err := s2.Every(1).Day().At("21:59").Do(func() {
		if time.Now().Weekday() != 6 {
			message = "EconomicCalendarAndNewsBot Running " + emoji.BeamingFaceWithSmilingEyes.String()
			for _, recipient := range recipients {
				// Send the punchline back to Telegram
				log.Printf("send to chatId, %s", strconv.Itoa(recipient.ChatId))
				telegramResponseBody, err := s.SendTextToTelegramChat(recipient.ChatId, recipient.MessageThreadId, message)
				if err != nil {
					log.Printf("got error %s from telegram, response body is %s", err.Error(), telegramResponseBody)
				} else {
					log.Printf("turbina vestas infos successfully distributed to chat id %d", recipient.ChatId)
				}
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
