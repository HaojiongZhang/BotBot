package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/HaojiongZhang/BotBot/util"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
	"github.com/zeromicro/go-zero/core/logx"
)

var conversationHistory = make(map[string][]string)
var botID string
func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Initialize Slack client and Socket Mode
	slackAppToken := os.Getenv("SLACK_APP_TOKEN")
	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")
	client := slack.New(slackBotToken, slack.OptionDebug(true), slack.OptionAppLevelToken(slackAppToken))
	socketClient := socketmode.New(client, socketmode.OptionDebug(false))

	authTest, err := client.AuthTest()
    if err != nil {
        log.Fatalf("Failed to get bot's user ID: %v", err)
    }
    botID = authTest.UserID

	// Start handling events
	go func() {
		for evt := range socketClient.Events {
			switch evt.Type {
			case socketmode.EventTypeEventsAPI:
				eventsAPIEvent, _ := evt.Data.(slackevents.EventsAPIEvent)
				socketClient.Ack(*evt.Request)

				if eventsAPIEvent.Type == slackevents.CallbackEvent {
					innerEvent := eventsAPIEvent.InnerEvent
					switch ev := innerEvent.Data.(type) {
					case *slackevents.AppMentionEvent:
						handleAppMentionEvent(client, ev)
					}
				}
			}
		}
	}()

	// Start the Socket Mode client
	err = socketClient.Run()
	if err != nil {
		log.Fatalf("Error running socketmode: %v", err)
	}
}

// handleAppMentionEvent processes the AppMentionEvent and generates a response using Ollama via LangChainGo.
func handleAppMentionEvent(client *slack.Client, event *slackevents.AppMentionEvent) {
	userID := event.User
	channelID := event.Channel
	
	text := strings.TrimSpace(strings.Replace(event.Text, fmt.Sprintf("<@%s>", botID), "", -1))
	logx.Debug("\n_______________\n")
	logx.Debug(text)
	logx.Debug("\n_______________\n")

	// Check if the user sent "ping"
	if strings.ToLower(text) == "ping" {
		response := fmt.Sprintf("Hello <@%s>! Pong!", userID)
		_, _, err := client.PostMessage(channelID, slack.MsgOptionText(response, false))
		if err != nil {
			log.Printf("Failed to post message: %v", err)
		}
		return
	}

	// Retrieve or initialize conversation history for the user
	history := conversationHistory[userID]

	// Call Ollama using the util package with the current history
	response, err := util.CallOllama(text, history)
	if err != nil {
		response = "Sorry, I couldn't process that."
	} else {
		conversationHistory[userID] = append(conversationHistory[userID], fmt.Sprintf("User: %s", text), fmt.Sprintf("Bot: %s", response))
	}

	_, _, err = client.PostMessage(channelID, slack.MsgOptionText(response, false))
	if err != nil {
		log.Printf("Failed to post message: %v", err)
	}
}