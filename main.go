package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/drpepper/slack-standup/internal/bot"
	"github.com/drpepper/slack-standup/internal/session"
	"github.com/joho/godotenv"
	"github.com/slack-go/slack"
)

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[standup] ")

	// Load .env file if present (not an error if missing)
	godotenv.Load()

	botToken := os.Getenv("SLACK_BOT_TOKEN")
	appToken := os.Getenv("SLACK_APP_TOKEN")
	signingSecret := os.Getenv("SLACK_SIGNING_SECRET")
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	if botToken == "" {
		log.Fatal("SLACK_BOT_TOKEN is required. Copy .env.example to .env and fill in your tokens.")
	}
	if appToken == "" && signingSecret == "" {
		log.Fatal("Either SLACK_APP_TOKEN (Socket Mode) or SLACK_SIGNING_SECRET (HTTP mode) is required.")
	}

	socketMode := appToken != ""

	log.Println("Starting standup bot…")
	log.Printf("Mode: %s", modeStr(socketMode))
	log.Printf("SLACK_BOT_TOKEN set: %v", botToken != "")
	log.Printf("SLACK_APP_TOKEN set: %v", appToken != "")
	log.Printf("SLACK_SIGNING_SECRET set: %v", signingSecret != "")
	if !socketMode {
		log.Printf("Port: %s", port)
	}

	slackLogger := bot.NewRedactingLogger(botToken, appToken, signingSecret)
	opts := []slack.Option{
		slack.OptionLog(slackLogger),
	}
	if socketMode {
		opts = append(opts, slack.OptionAppLevelToken(appToken))
	}
	api := slack.New(botToken, opts...)
	store := session.NewStore()
	b := bot.New(api, store, slackLogger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	var err error
	if socketMode {
		err = b.RunSocketMode(ctx)
	} else {
		err = b.RunHTTP(ctx, signingSecret, port)
	}
	if err != nil {
		log.Fatalf("Failed to start: %v", err)
	}
}

func modeStr(socketMode bool) string {
	if socketMode {
		return "Socket Mode"
	}
	return "HTTP"
}
