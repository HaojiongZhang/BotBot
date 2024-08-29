package util

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

const (
	MaxLabels  = 3
	LabelsFile = "logs/labels.txt"
)

var (
	GlobalLabels mapset.Set[string]
	labelsMutex  sync.RWMutex
)

func InitLLM() {
	GlobalLabels = mapset.NewSet[string]()
	loadLabelsFromFile()
}

func CallOllama(input string, history []string) (string, error) {
	llm, err := ollama.New(ollama.WithModel("llama3.1"))
	if err != nil {
		log.Printf("Failed to initialize Ollama model: %v", err)
		return "", err
	}

	// First, use LLM to classify the input
	classificationPrompt := fmt.Sprintf(
		"Classify the following input as either 'URL' or 'QUERY'. If it's a URL, extract the URL and any user-provided labels that appear after the URL. Labels are any words or symbols following the URL. Return the result strictly in the format 'URL: https://example.com/page, label1, label2' for URLs with labels, or 'URL: https://example.com/page' for URLs without labels. If the input is not a URL, return 'QUERY'. Do not include any additional text. Input: %s",
		input)
	classification, err := classifyInput(llm, classificationPrompt)
	if err != nil {
		log.Printf("Failed to classify input: %v", err)
		return "", err
	}
	PrintDebug("classification result: " + classification)
	PrintDebug("Global Labels are: " + strings.Join(GlobalLabels.ToSlice(), ", "))

	// Process based on classification
	if strings.HasPrefix(classification, "URL:") {
		parts := strings.SplitN(classification, ":", 2)
		if len(parts) == 2 {
			urlAndLabels := strings.TrimSpace(parts[1])
			urlParts := strings.Fields(urlAndLabels)
			url := urlParts[0]
			userLabels := strings.Join(urlParts[1:], ", ")
		

			summary:= ""
			err = AddEntryToDatabase(url, time.Now().Format("2006-01-02"), userLabels, url, summary)
			if err != nil {
				log.Printf("Failed to add entry to Notion: %v", err)
			}
			// return processURL(llm, urlAndLabels)
			return "I have added url and label to Notion!", nil
		}
	}

	// If not a URL, process as a regular query
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

func classifyInput(llm llms.LLM, prompt string) (string, error) {
	ctx := context.Background()
	classification, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(classification), nil
}

func processURL(llm llms.LLM, urlAndLabels string) (string, error) {
	parts := strings.Fields(urlAndLabels)
	url := parts[0]
	userLabels := parts[1:]
	if len(userLabels) > 0 && userLabels[0] != "label1"{
		userLabels = []string{}
	}

	content, err := WebScraper(url)
	if err != nil {
		log.Printf("Failed to scrape URL: %v", err)
		return "", err
	}
	PrintDebug("User provided labels: " + strings.Join(userLabels," "))
	var prompt string
	if len(userLabels) > 0 {
		if len(userLabels) > MaxLabels {
			userLabels = userLabels[:MaxLabels]
		}
		prompt = fmt.Sprintf(`Given the following website content, please provide:
1. A short summary (max 3 sentences)


Labels: %s
Content: %s

Format your response as follows and do not include any additional text beyond the specified fields or add any markdown support:
Summary: [Your summary here]`, strings.Join(userLabels, ", "), content)
	} else {
		prompt = fmt.Sprintf(`Given the following website content, please provide:
1. A short summary (max 3 sentences)
2. Up to %d labels for this content (prioritize using existing labels if it makes sense from this list: %s. If necessary, suggest new meaningful labels)

Content: %s

Format your response as follows and do not include any additional text beyond the specified fields or add any markdown support:
Summary: [Your summary here]
Labels: [comma-separated list of labels]`, MaxLabels, strings.Join(GlobalLabels.ToSlice(), ", "), content)
	}

	ctx := context.Background()
	completion, err := llms.GenerateFromSinglePrompt(ctx, llm, prompt)
	if err != nil {
		log.Printf("Failed to generate response for URL analysis: %v", err)
		return "", err
	}

	if len(userLabels) == 0 {
		suggestedLabels := extractLabels(completion)
		updateGlobalLabels(suggestedLabels)
	}
	PrintDebug("Final Content Here: " + completion)
	return completion, nil
}

func extractLabels(completion string) []string {
	labelSection := strings.Split(completion, "Labels: ")
	if len(labelSection) > 1 {
		labels := strings.Split(labelSection[1], "\n")[0]
		return strings.Split(labels, ", ")
	}
	return []string{}
}

// WebScraper function (to be implemented)
func WebScraper(url string) (string, error) {

	fullURL := "https://r.jina.ai/" + url

	response, err := http.Get(fullURL)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", response.StatusCode)
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	PrintDebug("jina response: " + string(body))
	return string(body), nil
}

func updateGlobalLabels(newLabels []string) {
	labelsMutex.Lock()
	defer labelsMutex.Unlock()

	labelsAdded := false
	for _, label := range newLabels {
		if GlobalLabels.Add(label) {
			labelsAdded = true
		}
	}

	if labelsAdded {
		saveLabelsToFile()
	}
}

// =============================== Helpers functions ========================== \
func loadLabelsFromFile() {
	file, err := os.OpenFile(LabelsFile, os.O_RDONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("Error opening labels file: %v", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		label := strings.TrimSpace(scanner.Text())
		if label != "" {
			GlobalLabels.Add(label)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading labels file: %v", err)
	}
}

func saveLabelsToFile() {
	labelsMutex.RLock()
	defer labelsMutex.RUnlock()

	file, err := os.Create(LabelsFile)
	if err != nil {
		log.Printf("Error creating labels file: %v", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for label := range GlobalLabels.Iter() {
		_, err := writer.WriteString(label + "\n")
		if err != nil {
			log.Printf("Error writing label to file: %v", err)
			return
		}
	}

	if err := writer.Flush(); err != nil {
		log.Printf("Error flushing writer: %v", err)
	}
}
