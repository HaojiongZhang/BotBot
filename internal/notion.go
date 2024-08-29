package util

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"strings"

	"github.com/jomei/notionapi"
)

var notionClient *notionapi.Client
var dbID string

func InitNotionClient() {
	apiKey := os.Getenv("NOTION_API_KEY")
	parentPageID := os.Getenv("NOTION_PARENT_PAGE_ID")

	if apiKey == "" || parentPageID == "" {
		log.Fatalf("API key or parent page ID is not set")
	}

	token := notionapi.Token(apiKey)
	notionClient = notionapi.NewClient(token)

	// Check if the database with the specified title exists under the parent page
	dbTitle := "Your Database Title" 
	var err error
	dbID, err = queryDatabase(dbTitle, parentPageID)
	if err != nil {
		log.Fatalf("Error checking database: %v", err)
	}
	fmt.Println(dbID)
	if dbID == "" {
		// Create the database if it doesn't exist
		dbID, err = createDatabase(dbTitle, parentPageID)
		if err != nil {
			log.Fatalf("Error creating database: %v", err)
		}
	}
	fmt.Printf("Database ID: %s\n", dbID)
}

func normalizeID(id string) string {
	return strings.ReplaceAll(id, "-", "")
}

func queryDatabase(dbTitle, parentID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Normalize the parent ID for comparison
	normalizedParentID := normalizeID(parentID)

	// Use Search API to find the database by title
	searchResponse, err := notionClient.Search.Do(ctx, &notionapi.SearchRequest{
		Query: dbTitle,
		Filter: notionapi.SearchFilter{
			Value:    "database",
			Property: "object",
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to search databases: %w", err)
	}

	for _, result := range searchResponse.Results {
		if database, ok := result.(*notionapi.Database); ok {
			
			if normalizeID(string(database.Parent.PageID)) == normalizedParentID {
				if len(database.Title) > 0 && database.Title[0].PlainText == dbTitle {
					return string(database.ID), nil // Database exists
				}
			}
		}
	}

	return "", nil // Database does not exist
}

func createDatabase(dbTitle, parentID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Request body to create a database with specified properties
	request := &notionapi.DatabaseCreateRequest{
		Parent: notionapi.Parent{
			Type:   notionapi.ParentTypePageID,
			PageID: notionapi.PageID(parentID),
		},
		Title: []notionapi.RichText{
			{
				Type: notionapi.ObjectTypeText,
				Text: &notionapi.Text{Content: dbTitle},
			},
		},
		Properties: notionapi.PropertyConfigs{
			"Name": notionapi.TitlePropertyConfig{
				Type: notionapi.PropertyConfigTypeTitle,
			},
			"Date Created": notionapi.DatePropertyConfig{
				Type: notionapi.PropertyConfigTypeDate,
			},
			"Label Tags": notionapi.MultiSelectPropertyConfig{
				Type: notionapi.PropertyConfigTypeMultiSelect,
				MultiSelect: notionapi.Select{
					Options: []notionapi.Option{
						{Name: "Tag1"},
						{Name: "Tag2"},
					},
				},
			},
			"URL Link": notionapi.URLPropertyConfig{
				Type: notionapi.PropertyConfigTypeURL,
			},
			"Summary": notionapi.RichTextPropertyConfig{
				Type: notionapi.PropertyConfigTypeRichText,
			},
		},
		IsInline: false,
	}

	// Call the Database.Create method
	newDatabase, err := notionClient.Database.Create(ctx, request)
	if err != nil {
		return "", fmt.Errorf("failed to create database: %w", err)
	}

	return string(newDatabase.ID), nil
}

func AddEntryToDatabase( name, dateCreated, labelTags, urlLink, summary string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var dateObject notionapi.Date
	if err := dateObject.UnmarshalText([]byte(dateCreated)); err != nil {
		return fmt.Errorf("failed to parse date: %w", err)
	}

	labels := strings.Split(labelTags, ",")
	multiSelectOptions := make([]notionapi.Option, 0, len(labels))
	for _, label := range labels {
		trimmedLabel := strings.TrimSpace(label)
		if trimmedLabel != "" {
			multiSelectOptions = append(multiSelectOptions, notionapi.Option{Name: trimmedLabel})
		}
	}

	// Prepare the properties for the new entry
	properties := notionapi.Properties{
		"Name": notionapi.TitleProperty{
			Title: []notionapi.RichText{
				{
					Text: &notionapi.Text{Content: name},
				},
			},
		},
		"Date Created": notionapi.DateProperty{
			Date: &notionapi.DateObject{
				Start: &dateObject, 
			},
		},
		"Label Tags": notionapi.MultiSelectProperty{
			MultiSelect: multiSelectOptions,
		},
		"URL Link": notionapi.URLProperty{
			URL: urlLink,
		},
		"Summary": notionapi.RichTextProperty{
			RichText: []notionapi.RichText{
				{
					Text: &notionapi.Text{Content: summary},
				},
			},
		},
	}

	// Create the new page (entry) in the specified database
	pageRequest := &notionapi.PageCreateRequest{
		Parent: notionapi.Parent{
			Type:       notionapi.ParentTypeDatabaseID,
			DatabaseID: notionapi.DatabaseID(dbID),
		},
		Properties: properties,
	}

	_, err := notionClient.Page.Create(ctx, pageRequest)
	if err != nil {
		return fmt.Errorf("failed to add entry to database: %w", err)
	}

	fmt.Println("Successfully added entry to database")
	return nil
}