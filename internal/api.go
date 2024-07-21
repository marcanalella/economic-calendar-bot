package internal

import (
	"bot/entity/telegram"
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func RegisterHandlers(router *mux.Router, service Service) {
	router.HandleFunc("/handle", HandleTelegramWebHook(service)).Methods(http.MethodPost)
}

func HandleTelegramWebHook(service Service) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var update telegram.Update
		var message string
		var siteInfo string

		err := json.NewDecoder(r.Body).Decode(&update)
		if err != nil {
			log.Printf("could not decode incoming update %s\n", err.Error())
			return
		}

		command := getCommand(update.Message.Text)
		switch command {
		case 1:
			message = service.PrepareStartMessageToTelegramChat()
			log.Printf("send to chatId, %s", strconv.Itoa(update.Message.Chat.Id))
			telegramResponseBody, err := service.SendTextToTelegramChat(update.Message.Chat.Id, 0, message)
			if err != nil {
				log.Printf("got error %s from telegram, response body is %s", err.Error(), telegramResponseBody)
			} else {
				log.Printf("turbine infos %s successfully distributed to chat id %d", siteInfo, update.Message.Chat.Id)
			}
			return
		default:
			message = service.PrepareCommandNotFoundMessageToTelegramChat()
			log.Printf("send to chatId, %s", strconv.Itoa(update.Message.Chat.Id))
			telegramResponseBody, err := service.SendTextToTelegramChat(update.Message.Chat.Id, 0, message)
			if err != nil {
				log.Printf("got error %s from telegram, response body is %s", err.Error(), telegramResponseBody)
			} else {
				log.Printf("turbine infos %s successfully distributed to chat id %d", siteInfo, update.Message.Chat.Id)
			}
			return
		}
	}
}

func getCommand(command string) int {
	if strings.Contains(command, "/start") {
		return 1
	}

	return 0
}
