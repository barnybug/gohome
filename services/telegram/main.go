// Service to send telegram messages.
package telegram

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/barnybug/gohome/pubsub"
	"github.com/barnybug/gohome/services"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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
		msg := tgbotapi.NewPhoto(services.Config.Telegram.Chat_id, tgbotapi.FilePath(filename))
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

func rewriteTelegramCommands(s string) string {
	// Rewrite "/telegram_command ..." -> "telegram/command ..."
	s = strings.TrimLeft(s, "/")
	i := strings.Index(s, " ")
	if i == -1 {
		i = len(s)
	}
	return strings.Replace(s[:i], "_", "/", -1) + s[i:]
}

func (self *Service) Run() error {
	bot, err := tgbotapi.NewBotAPI(services.Config.Telegram.Token)
	if err != nil {
		log.Fatalln(err)
	}

	self.bot = bot

	go func() {
		// Just share the chat ID back to updates
		u := tgbotapi.NewUpdate(0)
		u.Timeout = 60

		updates := bot.GetUpdatesChan(u)

		for update := range updates {
			if update.Message == nil {
				continue
			}

			if services.Config.Telegram.Chat_id == update.Message.Chat.ID {
				remote := fmt.Sprint(update.Message.MessageID)
				text := rewriteTelegramCommands(update.Message.Text)
				services.SendQuery(text, "telegram", remote, "alert")
			} else {
				text := fmt.Sprintf("This is chat %d, configure this in gohome telgram->chat_id.", update.Message.Chat.ID)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, text)
				bot.Send(msg)
			}
		}
	}()

	events := services.Subscriber.Subscribe(pubsub.Prefix("alert"))
	for ev := range events {
		if ev.Target() == "telegram" {
			remote := ev.StringField("remote")
			i, _ := strconv.Atoi(remote)
			self.sendMessage(ev, i)
		}
	}
	return nil
}
