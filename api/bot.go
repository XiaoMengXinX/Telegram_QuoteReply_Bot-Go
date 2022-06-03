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
	Msg       string `json:"text"`
	ChatID    int64  `json:"chat_id"`
	ReplyTo   int64  `json:"reply_to_message_id"`
	ParseMode string `json:"parse_mode"`
	Method    string `json:"method"`
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
			Msg:       replyMsg,
			Method:    "sendMessage",
			ParseMode: "MarkdownV2",
			//ReplyTo:   int64(update.Message.MessageID),
			ChatID: update.Message.Chat.ID,
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
	senderID := message.From.ID

	if message.ReplyToMessage != nil {
		replyToName := mdV2escaper.Replace(message.ReplyToMessage.From.FirstName + " " + message.ReplyToMessage.From.LastName)
		replyToID := message.ReplyToMessage.From.ID
		if strings.HasPrefix(message.Text, "\\") {
			senderName, replyToName = replyToName, senderName
			senderID, replyToID = replyToID, senderID
		}
		if len(keywords) < 2 {
			return fmt.Sprintf("[%s](tg://user?id=%d) %s了 [%s](tg://user?id=%d)！", senderName, senderID, keywords[0], replyToName, replyToID)
		} else {
			return fmt.Sprintf("[%s](tg://user?id=%d) %s [%s](tg://user?id=%d) %s！", senderName, senderID, keywords[0], replyToName, replyToID, keywords[1])
		}
	} else {
		if len(keywords) < 2 {
			return fmt.Sprintf("[%s](tg://user?id=%d) %s了 [自己](tg://user?id=%d)！", senderName, senderID, keywords[0], senderID)
		} else {
			return fmt.Sprintf("[%s](tg://user?id=%d) %s [自己](tg://user?id=%d) %s！", senderName, senderID, keywords[0], senderID, keywords[1])
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
