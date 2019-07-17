package telegram

import (
  "fmt"

  tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
  "github.com/urepairpc/autodeploy/pkg/config"
)

var bot *tgbotapi.BotAPI

// NewTelegram initializes a new bot
func NewTelegram(token string) error {
  var err error
  bot, err = tgbotapi.NewBotAPI(token)
  return err
}

// SendMe - sends a message to the user from the configuration
func SendMe(message string) tgbotapi.Message {
  if !config.Data.Telegram.Enable || bot.Self.ID == 0 {
    return tgbotapi.Message{}
  }

  msg := tgbotapi.NewMessage(config.Data.Telegram.UserID, "["+config.RepAutodeploy+"] "+message)
  resp, err := bot.Send(msg)

  if err != nil {
    fmt.Println(err)
    return tgbotapi.Message{}
  }

  return resp
}
