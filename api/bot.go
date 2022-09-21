package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"unicode"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var mdV2escaper = strings.NewReplacer(
	"_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(",
	"\\(", ")", "\\)", "~", "\\~", "`", "\\`", ">", "\\>",
	"#", "\\#", "+", "\\+", "-", "\\-", "=", "\\=", "|",
	"\\|", "{", "\\{", "}", "\\}", ".", "\\.", "!", "\\!",
)

type Response struct {
	Msg                   string `json:"text"`
	ChatID                int64  `json:"chat_id"`
	ReplyTo               int64  `json:"reply_to_message_id"`
	ParseMode             string `json:"parse_mode"`
	Method                string `json:"method"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

func BotHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, _ := ioutil.ReadAll(r.Body)

	var update tgbotapi.Update

	err := json.Unmarshal(body, &update)
	if err != nil {
		log.Println(err)
		return
	}

	if update.Message != nil {
		replyMsg := QuoteReply(update.Message)
		if replyMsg == "" {
			return
		}

		data := Response{
			Msg:                   replyMsg,
			Method:                "sendMessage",
			ParseMode:             "MarkdownV2",
			ChatID:                update.Message.Chat.ID,
			DisableWebPagePreview: true,
			//ReplyTo:   int64(update.Message.MessageID),
		}
		msg, _ := json.Marshal(data)

		w.Header().Add("Content-Type", "application/json")

		_, _ = fmt.Fprintf(w, string(msg))
	}
}

func QuoteReply(message *tgbotapi.Message) (replyMsg string) {
	if len(message.Text) < 2 {
		return
	}
	if !strings.HasPrefix(message.Text, "/") || (isASCII(message.Text[:2]) && !strings.HasPrefix(message.Text, "/$")) {
		if !strings.HasPrefix(message.Text, "\\") || (isASCII(message.Text[:2]) && !strings.HasPrefix(message.Text, "\\$")) {
			return
		}
	}

	keywords := strings.SplitN(mdV2escaper.Replace(strings.Replace(message.Text, "$", "", 1)[1:]), " ", 2)
	if len(keywords) == 0 {
		return
	}

	senderName := mdV2escaper.Replace(message.From.FirstName + " " + message.From.LastName)
	senderURI := fmt.Sprintf("tg://user?id=%d", message.From.ID)

	if message.SenderChat != nil {
		senderName = mdV2escaper.Replace(message.SenderChat.Title)
		senderURI = fmt.Sprintf("https://t.me/%s", message.SenderChat.UserName)
	}

	if message.ReplyToMessage != nil {
		replyToName := mdV2escaper.Replace(message.ReplyToMessage.From.FirstName + " " + message.ReplyToMessage.From.LastName)
		replyToURI := fmt.Sprintf("tg://user?id=%d", message.ReplyToMessage.From.ID)

		if message.ReplyToMessage.From.IsBot && len(message.ReplyToMessage.Entities) != 0 {
			if message.ReplyToMessage.Entities[0].Type == "text_mention" {
				replyToName = mdV2escaper.Replace(message.ReplyToMessage.Entities[0].User.FirstName + " " + message.ReplyToMessage.Entities[0].User.LastName)
				replyToURI = fmt.Sprintf("tg://user?id=%d", message.ReplyToMessage.Entities[0].User.ID)
			}
		}

		if message.ReplyToMessage.SenderChat != nil {
			replyToName = mdV2escaper.Replace(message.ReplyToMessage.SenderChat.Title)
			replyToURI = fmt.Sprintf("https://t.me/%s", message.ReplyToMessage.SenderChat.UserName)
		}

		if strings.HasPrefix(message.Text, "\\") {
			senderName, replyToName = replyToName, senderName
			senderURI, replyToURI = replyToURI, senderURI
		}
		if len(keywords) < 2 {
			return fmt.Sprintf("[%s](%s) %s了 [%s](%s)！", senderName, senderURI, keywords[0], replyToName, replyToURI)
		} else {
			return fmt.Sprintf("[%s](%s) %s [%s](%s) %s！", senderName, senderURI, keywords[0], replyToName, replyToURI, keywords[1])
		}
	} else {
		if len(keywords) < 2 {
			return fmt.Sprintf("[%s](%s) %s了 [自己](%s)！", senderName, senderURI, keywords[0], senderURI)
		} else {
			return fmt.Sprintf("[%s](%s) %s [自己](%s) %s！", senderName, senderURI, keywords[0], senderURI, keywords[1])
		}
	}
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}
