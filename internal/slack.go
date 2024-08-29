// util/slack_util.go
package util

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/socketmode"
)

var (
	client            *slack.Client
	socketClient      *socketmode.Client
	conversationHistory = make(map[string][]string)
	thinkingEmoji     = "one-sec-cooking"
	botID             string
	once              sync.Once
)




// InitializeSlackClient initializes the Slack client and Socket Mode client.
func InitializeSlackClient() error {
	slackAppToken := os.Getenv("SLACK_APP_TOKEN")
	slackBotToken := os.Getenv("SLACK_BOT_TOKEN")

	client = slack.New(slackBotToken, slack.OptionDebug(verbose), slack.OptionAppLevelToken(slackAppToken))
	socketClient = socketmode.New(client, socketmode.OptionDebug(verbose))

	authTest, err := client.AuthTest()
	if err != nil {
		return err
	}
	SetBotID(authTest.UserID)

	return nil
}

// SetBotID sets the bot's ID once during initialization.
func SetBotID(id string) {
	once.Do(func() {
		botID = id
	})
}

// RunSlackServer starts handling Slack events via Socket Mode.
func RunSlackServer() error {
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
						HandleAppMentionEvent(client, ev)
					}
				}
			}
		}
	}()

	return socketClient.Run()
}

// HandleAppMentionEvent processes the AppMentionEvent and generates a response.
func HandleAppMentionEvent(client *slack.Client, event *slackevents.AppMentionEvent) {
	userID := event.User
	channelID := event.Channel
	messageTimestamp := event.TimeStamp

	text := strings.TrimSpace(strings.Replace(event.Text, fmt.Sprintf("<@%s>", botID), "", -1))

	err := client.AddReaction(thinkingEmoji, slack.ItemRef{
		Channel:   channelID,
		Timestamp: messageTimestamp,
	})
	if err != nil {
		log.Printf("Failed to add reaction: %v", err)
	}

	defer func() {
		err = client.RemoveReaction(thinkingEmoji, slack.ItemRef{
			Channel:   channelID,
			Timestamp: messageTimestamp,
		})
		if err != nil {
			log.Printf("Failed to remove reaction: %v", err)
		}
	}()

    text = strings.ToLower(text) 
	// Check if the user sent "ping"
	if text == "ping" {
		response := fmt.Sprintf("Hello <@%s>! Pong!", userID)
		_, _, err := client.PostMessage(channelID, slack.MsgOptionText(response, false))
		if err != nil {
			log.Printf("Failed to post message: %v", err)
		}
		return
	}

	if strings.ToLower(text) == "-h" || strings.ToLower(text) == "-help" {
		helpMessage := "To add a link to notion follow the format: `@BotBot YOUR-URL-LINK-HERE LABEL1 LABEL2 ...`"
		_, _, err := client.PostMessage(channelID, slack.MsgOptionText(helpMessage, false))
		if err != nil {
			log.Printf("Failed to post message: %v", err)
		}
		return
	}

	systemMessage := "You are a helpful, funny, and sarcastic Slack bot called BotBot that can answer questions and add links to Notion. When adding links to Notion, users should provide the URL and optional labels.Users should call bot via the -h flag"

	history := conversationHistory[userID]
	if len(history) == 0 {
		history = []string{fmt.Sprintf("System: %s", systemMessage)}
	}
	
	response, err := CallOllama(text, history)
	if err != nil {
		response = "Sorry, I couldn't process that."
	} else {
		if len(conversationHistory[userID]) == 0 {
			conversationHistory[userID] = append([]string{fmt.Sprintf("System: %s", systemMessage)}, fmt.Sprintf("User: %s", text), fmt.Sprintf("Bot: %s", response))
		} else {
			conversationHistory[userID] = append(conversationHistory[userID], fmt.Sprintf("User: %s", text), fmt.Sprintf("Bot: %s", response))
		}
	}

	_, _, err = client.PostMessage(channelID, slack.MsgOptionText(response, false))
	if err != nil {
		log.Printf("Failed to post message: %v", err)
	}

	
}