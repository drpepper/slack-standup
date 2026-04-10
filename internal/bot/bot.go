package bot

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/drpepper/slack-standup/internal/session"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/socketmode"
)

// Bot holds the Slack clients and shared state for the standup bot.
type Bot struct {
	api    *slack.Client
	store  *session.Store
	loop   *EventLoop
	logger *RedactingLogger

	// messageTs tracks the Slack message timestamp per channel for in-place updates.
	// Only accessed from the event loop goroutine.
	messageTs map[string]string
}

// New creates a new Bot instance.
func New(api *slack.Client, store *session.Store, logger *RedactingLogger) *Bot {
	return &Bot{
		api:       api,
		store:     store,
		loop:      NewEventLoop(),
		logger:    logger,
		messageTs: make(map[string]string),
	}
}

// RunSocketMode starts the bot in Socket Mode.
func (b *Bot) RunSocketMode(ctx context.Context) error {
	smOpts := []socketmode.Option{}
	if b.logger != nil {
		smOpts = append(smOpts, socketmode.OptionDebug(true), socketmode.OptionLog(b.logger))
	}
	smClient := socketmode.New(b.api, smOpts...)
	handler := socketmode.NewSocketmodeHandler(smClient)

	handler.HandleSlashCommand("/standup", b.handleSlashCommand)
	handler.HandleInteractionBlockAction("standup_next", b.handleNextAction)
	handler.HandleInteractionBlockAction("standup_end", b.handleEndAction)

	go b.loop.Run(ctx)

	log.Println("Bot ready (Socket Mode)")
	return handler.RunEventLoopContext(ctx)
}

// RunHTTP starts the bot in HTTP mode.
func (b *Bot) RunHTTP(ctx context.Context, signingSecret string, port string) error {
	go b.loop.Run(ctx)

	http.HandleFunc("/slack/events", b.makeSlashHTTPHandler(signingSecret))
	http.HandleFunc("/slack/actions", b.makeActionsHTTPHandler(signingSecret))

	log.Printf("Bot ready on port %s", port)
	server := &http.Server{Addr: ":" + port}
	go func() {
		<-ctx.Done()
		server.Close()
	}()
	return server.ListenAndServe()
}

// makeSlashHTTPHandler returns an HTTP handler for slash commands.
func (b *Bot) makeSlashHTTPHandler(signingSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			http.Error(w, "verification failed", http.StatusUnauthorized)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		body, err := readAndVerify(r, &verifier)
		if err != nil {
			http.Error(w, "verification failed", http.StatusUnauthorized)
			return
		}
		_ = body

		cmd, err := slack.SlashCommandParse(r)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
		go b.handleSlashCommandDirect(cmd)
	}
}

// makeActionsHTTPHandler returns an HTTP handler for interactive actions.
func (b *Bot) makeActionsHTTPHandler(signingSecret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		verifier, err := slack.NewSecretsVerifier(r.Header, signingSecret)
		if err != nil {
			http.Error(w, "verification failed", http.StatusUnauthorized)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		body, err := readAndVerify(r, &verifier)
		if err != nil {
			http.Error(w, "verification failed", http.StatusUnauthorized)
			return
		}
		_ = body

		var callback slack.InteractionCallback
		payload := r.FormValue("payload")
		if err := callback.UnmarshalJSON([]byte(payload)); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)

		for _, action := range callback.ActionCallback.BlockActions {
			switch action.ActionID {
			case "standup_next":
				go b.handleNextDirect(callback.Channel.ID, callback.Message.Timestamp, callback.User.ID)
			case "standup_end":
				go b.handleEndDirect(callback.Channel.ID, callback.Message.Timestamp, callback.User.ID)
			}
		}
	}
}

func readAndVerify(r *http.Request, verifier *slack.SecretsVerifier) ([]byte, error) {
	body := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := r.Body.Read(buf)
		if n > 0 {
			body = append(body, buf[:n]...)
			verifier.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	if err := verifier.Ensure(); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}
	return body, nil
}
