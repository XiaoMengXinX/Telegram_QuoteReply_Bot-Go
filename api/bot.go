package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Response struct {
	Msg                   string `json:"text"`
	ChatID                int64  `json:"chat_id"`
	ReplyTo               int64  `json:"reply_to_message_id"`
	MessageThreadID       int64  `json:"message_thread_id"`
	ParseMode             string `json:"parse_mode"`
	Method                string `json:"method"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview"`
}

func BotHandler(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	body, _ := io.ReadAll(r.Body)

	var update tgbotapi.Update

	err := json.Unmarshal(body, &update)
	if err != nil {
		log.Println(err)
		return
	}

	if update.Message != nil {
		bot := &tgbotapi.BotAPI{
			Token:  strings.ReplaceAll(r.URL.Path, "/", ""),
			Client: &http.Client{},
			Buffer: 100,
		}
		bot.SetAPIEndpoint(tgbotapi.APIEndpoint)

		replyMsg := QuoteReply(bot, update.Message)
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
		if update.Message.IsTopicMessage {
			data.MessageThreadID = int64(update.Message.MessageThreadID)
		}
		msg, _ := json.Marshal(data)

		w.Header().Add("Content-Type", "application/json")

		_, _ = fmt.Fprintf(w, string(msg))
	}
}

func QuoteReply(bot *tgbotapi.BotAPI, message *tgbotapi.Message) (replyMsg string) {
	if len(message.Text) < 2 {
		return
	}
	if !strings.HasPrefix(message.Text, "/") || (isASCII(message.Text[:2]) && !strings.HasPrefix(message.Text, "/$")) {
		if !strings.HasPrefix(message.Text, "\\") || (isASCII(message.Text[:2]) && !strings.HasPrefix(message.Text, "\\$")) {
			return
		}
	}

	keywords := strings.SplitN(tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, strings.Replace(message.Text, "$", "", 1)[1:]), " ", 2)
	if len(keywords) == 0 {
		return
	}

	senderName := tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, message.From.FirstName+" "+message.From.LastName)
	senderURI := fmt.Sprintf("tg://user?id=%d", message.From.ID)
	replyToName := ""
	replyToURI := ""

	if message.SenderChat != nil {
		senderName = tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, message.SenderChat.Title)
		senderURI = fmt.Sprintf("tg://user?id=%s", message.SenderChat.UserName)
	}

	if message.ReplyToMessage != nil && message.IsTopicMessage {
		if message.ReplyToMessage.MessageID == message.MessageThreadID {
			message.ReplyToMessage = nil
		}
	}

	if message.ReplyToMessage != nil {
		replyToName = tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, message.ReplyToMessage.From.FirstName+" "+message.ReplyToMessage.From.LastName)
		replyToURI = fmt.Sprintf("tg://user?id=%d", message.ReplyToMessage.From.ID)

		if message.ReplyToMessage.From.IsBot && len(message.ReplyToMessage.Entities) != 0 {
			if message.ReplyToMessage.Entities[0].Type == "text_mention" {
				replyToName = tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, message.ReplyToMessage.Entities[0].User.FirstName+" "+message.ReplyToMessage.Entities[0].User.LastName)
				replyToURI = fmt.Sprintf("tg://user?id=%d", message.ReplyToMessage.Entities[0].User.ID)
			}
		}

		if message.ReplyToMessage.SenderChat != nil {
			replyToName = tgbotapi.EscapeText(tgbotapi.ModeMarkdownV2, message.ReplyToMessage.SenderChat.Title)
			replyToURI = fmt.Sprintf("tg://user?id=%s", message.ReplyToMessage.SenderChat.UserName)
		}

		if strings.HasPrefix(message.Text, "\\") {
			senderName, replyToName = replyToName, senderName
			senderURI, replyToURI = replyToURI, senderURI
		}
	} else {
		textNoCommand := strings.TrimPrefix(strings.TrimPrefix(message.Text, "/"), "$")
		if text := strings.Split(textNoCommand, "@"); len(text) > 1 {
			if name := getUserByUsername(text[1]); name != "" {
				replyToName = name
				replyToURI = fmt.Sprintf("tg://user?id=%s", text[1])
			}
		}
		if replyToName == "" {
			replyToName = "自己"
			replyToURI = senderURI
		}
	}
	if len(keywords) < 2 {
		return fmt.Sprintf("[%s](%s) %s了 [%s](%s)！", senderName, senderURI, keywords[0], replyToName, replyToURI)
	} else {
		return fmt.Sprintf("[%s](%s) %s [%s](%s) %s！", senderName, senderURI, keywords[0], replyToName, replyToURI, keywords[1])
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

func getUserByUsername(username string) (name string) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", fmt.Sprintf("https://t.me/%s", username), nil)
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Println(string(body))
	if len(body) == 0 {
		return
	}

	reName := regexp.MustCompile(`<meta property="og:title" content="([^"]*)"`)
	match := reName.FindStringSubmatch(string(body))
	if len(match) > 1 {
		name = match[1]
	}
	pageTitle := ""
	reTitle1 := regexp.MustCompile(`<title>`)
	reTitle2 := regexp.MustCompile(`</title>`)
	start := reTitle1.FindStringIndex(string(body))
	end := reTitle2.FindStringIndex(string(body))
	if start != nil && end != nil {
		pageTitle = string(body)[start[1]:end[0]]
	}

	if pageTitle == name { // 用户不存在
		name = ""
	}
	return
}
