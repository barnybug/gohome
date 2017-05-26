// Service to send telegram messages.
package telegram

import (
	"fmt"
	"log"
	"strconv"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"

	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

// Service telegram
type Service struct {
	bot *tgbotapi.BotAPI
}

func (self *Service) ID() string {
	return "telegram"
}

func (self *Service) sendMessage(ev *pubsub.Event, remote int) {
	if filename, ok := ev.Fields["filename"].(string); ok {
		log.Printf("Sending telegram picture: %s", filename)
		msg := tgbotapi.NewPhotoUpload(services.Config.Telegram.Chat_id, filename)
		msg.Caption = ev.StringField("message")
		if remote != 0 {
			msg.ReplyToMessageID = remote
		}
		self.bot.Send(msg)
	} else if message, ok := ev.Fields["message"].(string); ok {
		log.Printf("Sending telegram message: %s", message)
		msg := tgbotapi.NewMessage(services.Config.Telegram.Chat_id, message)
		if remote != 0 {
			msg.ReplyToMessageID = remote
		}
		self.bot.Send(msg)
	}
}

func (self *Service) Run() error {
	bot, err := tgbotapi.NewBotAPI(services.Config.Telegram.Token)
	if err != nil {
		log.Fatalln(err)
	}

	// services.Config.Pushbullet.Token
	self.bot = bot

	go func() {
		// Just share the chat ID back to updates
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60

		updates, err := bot.GetUpdatesChan(u)
		if err != nil {
			log.Fatalln(err)
		}

		for update := range updates {
			if update.Message == nil {
				continue
			}

			if services.Config.Telegram.Chat_id == update.Message.Chat.ID {
				remote := fmt.Sprint(update.Message.MessageID)
				services.SendQuery(update.Message.Text, "telegram", remote, "alert")
			} else {
				text := fmt.Sprintf("This is chat %d, configure this in gohome telgram->chat_id.", update.Message.Chat.ID)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				bot.Send(msg)
			}
		}
	}()

	events := services.Subscriber.FilteredChannel("alert")
	for ev := range events {
		if ev.Target() == "telegram" {
			remote := ev.StringField("remote")
			i, _ := strconv.Atoi(remote)
			self.sendMessage(ev, i)
		}
	}
	return nil
}
