package util

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

func CallOllama(input string, history []string) (string, error) {
	llm, err := ollama.New(ollama.WithModel("llama3.1"))
	if err != nil {
		log.Printf("Failed to initialize Ollama model: %v", err)
		return "", err
	}

	prompt := strings.Join(history, "\n") + fmt.Sprintf("\nUser: %s\nBot:", input)

	ctx := context.Background()
	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		log.Printf("Failed to generate response from Ollama: %v", err)
		return "", err
	}

	history = append(history, fmt.Sprintf("User: %s", input), fmt.Sprintf("Bot: %s", completion))

	return completion, nil
}