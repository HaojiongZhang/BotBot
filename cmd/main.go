package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"flag"

	"github.com/HaojiongZhang/BotBot/internal"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

var conversationHistory = make(map[string][]string)
var thinkingEmoji = "one-sec-cooking"
var botID string
func main() {
	// Load environment variables
	var verboseFlag bool
	flag.BoolVar(&verboseFlag, "v", false, "Enable verbose logging")
	flag.Parse()

	// Set verbose flag in the utility package
	util.SetVerbose(verboseFlag)

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Initialize Slack client and Socket Mode
	slackAppToken := os.Getenv("SLACK_APP_TOKEN")
	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")
	util.InitNotionClient()

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
	messageTimestamp := event.TimeStamp
	
	text := strings.TrimSpace(strings.Replace(event.Text, fmt.Sprintf("<@%s>", botID), "", -1))
	

	err := client.AddReaction(thinkingEmoji, slack.ItemRef{
		Channel:   channelID,
		Timestamp: messageTimestamp,
	})

	// Check if the user sent "ping"
	if strings.ToLower(text) == "ping" {
		response := fmt.Sprintf("Hello <@%s>! Pong!", userID)
		_, _, err := client.PostMessage(channelID, slack.MsgOptionText(response, false))
		if err != nil {
			log.Printf("Failed to post message: %v", err)
		}
		return
	}

	history := conversationHistory[userID]

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

	err = client.RemoveReaction(thinkingEmoji, slack.ItemRef{
		Channel:   channelID,
		Timestamp: messageTimestamp,
	})
	if err != nil {
		log.Printf("Failed to remove reaction: %v", err)
	}
}