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
)

func serveBot(api tgapi.Api) {
	updateOffset := 0

	for {
		updates, err := api.GetUpdates(updateOffset, 100, 100, nil)
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

				_, err = api.SendMessage(&tgapi.NewMessage{
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

func apiHandler(api tgapi.Api) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) == 1 && parts[0] == "" {
			if r.Method == "GET" {
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "Message Relay Bot API")
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Method not allowed")
			return
		}
		if parts[0] == "u" {
			if len(parts) != 3 {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, "This endpoint doesn't exist")
				return
			}
			if parts[2] != "message" {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, "This endpoint doesn't exist")
				return
			}
			if r.Method == "GET" {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, `<form method="POST"><label for="message">Message:</label><textarea name="message"></textarea><br><button type="submit">Send</button></form>`)
				return
			}
			if r.Method == "POST" {
				err := r.ParseForm()
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

				chatId, err := strconv.ParseInt(parts[1], 10, 0)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "Invalid chat id: %v", err)
					return
				}
				_, err = api.SendMessage(&tgapi.NewMessage{
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
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			fmt.Fprintf(w, "Method not allowed")
			return
		}
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "This endpoint doesn't exist")
	}
}

func main() {
	token := os.Getenv("TG_BOT_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Missing env var: TG_BOT_TOKEN")
		os.Exit(1)
	}

	api := tgapi.New(token)

	fmt.Println("Starting bot service")
	go serveBot(api)
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	fmt.Printf("Starting api service on %s\n", addr)
	http.HandleFunc("/", apiHandler(api))
	http.ListenAndServe(addr, nil)
}
