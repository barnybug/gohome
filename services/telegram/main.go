// Service to send telegram messages.
package telegram

import (
	"fmt"
	"log"

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

func (self *Service) sendMessage(ev *pubsub.Event) {
	if message, ok := ev.Fields["message"].(string); ok {
		log.Printf("Sending telegram message: %s", message)
		msg := tgbotapi.NewMessage(services.Config.Telegram.Chat_id, message)
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

			text := fmt.Sprintf("This is chat %d", update.Message.Chat.ID)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
			// msg.ReplyToMessageID = update.Message.MessageID

			bot.Send(msg)
		}
	}()

	events := services.Subscriber.FilteredChannel("alert")
	for ev := range events {
		if ev.Target() == "telegram" {
			self.sendMessage(ev)
		}
	}
	return nil
}
