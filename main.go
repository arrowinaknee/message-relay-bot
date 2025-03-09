package main

import (
	"fmt"
	"math/rand"
	"message-relay-bot/pkg/tgapi"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gorilla/mux"
)

func serveBot(tg tgapi.Api) {
	updateOffset := 0

	for {
		updates, err := tg.GetUpdates(updateOffset, 100, 100, nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to poll: %v\n", err)
			continue
		}

		for _, update := range updates {
			if update.Id >= updateOffset {
				updateOffset = update.Id + 1
			}
			switch v := update.Value.(type) {
			case *tgapi.Message:

				fmt.Printf("Message: %s says %q\n", v.Chat.FirstName, v.Text)

				sr := strings.NewReader(v.Text)
				var sb strings.Builder
				rand.New(rand.NewSource(time.Now().UnixNano()))
				for {
					ch, _, err := sr.ReadRune()
					if err != nil {
						break // ReadRune only returns EOF
					}
					if rand.Int()%2 == 0 {
						sb.WriteRune(unicode.ToLower(ch))
					} else {
						sb.WriteRune(unicode.ToUpper(ch))
					}
				}

				_, err = tg.SendMessage(&tgapi.NewMessage{
					ChatId: v.Chat.Id,
					Text:   sb.String(),
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to send message: %v\n", err)
				}
			}
		}
	}
}

func serveRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Message Relay Bot API")
}

func userMessagePage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// just check that id can be parsed
	_, err := strconv.ParseInt(vars["id"], 10, 0)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "User not found")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<form method="POST"><label for="message">Message:</label><textarea name="message"></textarea><br><button type="submit">Send</button></form>`)
}

// TODO: use api object for handler ctx
func userMessageSend(tg tgapi.Api) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		chatId, err := strconv.ParseInt(vars["id"], 10, 0)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "User not found")
			return
		}

		err = r.ParseForm()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Error parsing form: %v", err)
			return
		}

		text := r.FormValue("message")
		if text == "" {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, "Missing message text")
			return
		}

		_, err = tg.SendMessage(&tgapi.NewMessage{
			ChatId: int(chatId),
			Text:   text,
		})

		// TODO: different responses for different errors
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Error sending message: %v", err)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Message sent")
	}
}

func serveApi(addr string, api tgapi.Api) {
	r := mux.NewRouter()
	r.StrictSlash(true)
	r.HandleFunc("/", serveRoot).Methods("GET")
	r.HandleFunc("/u/{id}/message", userMessagePage).Methods("GET")
	r.HandleFunc("/u/{id}/message", userMessageSend(api)).Methods("POST")
	http.ListenAndServe(addr, r)
}

func main() {
	token := os.Getenv("TG_BOT_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Missing env var: TG_BOT_TOKEN")
		os.Exit(1)
	}
	tg := tgapi.New(token)

	fmt.Println("Starting bot service")
	go serveBot(tg)

	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	fmt.Printf("Starting api service on %s\n", addr)
	serveApi(addr, tg)
}
