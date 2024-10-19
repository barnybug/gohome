// Service to send telegram messages.
package telegram

import (
	"fmt"
	"io"
	"log"
	"net/http"
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

func downloadUrl(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func (self *Service) sendMessage(ev *pubsub.Event, remote int) {
	if url, ok := ev.Fields["url"].(string); ok {
		log.Printf("Sending telegram picture: %s", url)
		bytes, err := downloadUrl(url)
		if err != nil {
			log.Printf("Error downloading picture: %s", err)
			return
		}
		file := tgbotapi.FileBytes{
			Name: "snapshot.jpg", Bytes: bytes,
		}
		msg := tgbotapi.NewPhoto(services.Config.Telegram.Chat_id, file)
		msg.Caption = ev.StringField("message")
		if remote != 0 {
			msg.ReplyToMessageID = remote
		}
		if ev.Fields["markdown"] == true {
			msg.ParseMode = tgbotapi.ModeMarkdownV2
		}
		if ev.Fields["quiet"] == true {
			msg.DisableNotification = true
		}
		_, err = self.bot.Send(msg)
		if err != nil {
			log.Printf("Error sending picture: %s", err)
		}
	} else if message, ok := ev.Fields["message"].(string); ok {
		log.Printf("Sending telegram message: %s", message)
		msg := tgbotapi.NewMessage(services.Config.Telegram.Chat_id, message)
		if remote != 0 {
			msg.ReplyToMessageID = remote
		}
		if ev.Fields["markdown"] == true {
			msg.ParseMode = tgbotapi.ModeMarkdownV2
		}
		if ev.Fields["quiet"] == true {
			msg.DisableNotification = true
		}
		_, err := self.bot.Send(msg)
		if err != nil {
			log.Printf("Error sending message: %s", err)
		}
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
