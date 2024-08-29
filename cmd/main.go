package main

import (
	"log"
	"flag"

	"github.com/HaojiongZhang/BotBot/internal"
	"github.com/joho/godotenv"

)


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

	util.InitNotionClient()

	util.InitLLM()
	
	// Initialize Slack client and Socket Mode
	if err := util.InitializeSlackClient(); err != nil {
		log.Fatalf("Failed to initialize Slack client: %v", err)
	}

	// Run the Slack server
	if err := util.RunSlackServer(); err != nil {
		log.Fatalf("Error running Slack server: %v", err)
	}
}