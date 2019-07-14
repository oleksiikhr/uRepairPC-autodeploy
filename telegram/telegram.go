package telegram

import (
	"fmt"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/uRepairPC/autodeploy/config"
)

var bot *tgbotapi.BotAPI

func NewTelegram(token string) error {
	var err error
	bot, err = tgbotapi.NewBotAPI(token)
	return err
}

func SendMe(message string) tgbotapi.Message {
	if !config.Data.Telegram.Enable || bot.Self.ID == 0 {
		return tgbotapi.Message{}
	}

	msg := tgbotapi.NewMessage(config.Data.Telegram.UserId, "["+config.RepAutodeploy+"] "+message)
	resp, err := bot.Send(msg)

	if err != nil {
		fmt.Println(err)
		return tgbotapi.Message{}
	}

	return resp
}
