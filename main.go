package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"
)

func serveBot(token string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s", token)

	updateOffset := 0

	for {
		pollurl := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=100", url, updateOffset)
		resp, err := http.DefaultClient.Get(pollurl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to poll: %v\n", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Failed to poll: %s, could not read error body: %v\n", resp.Status, err)
			}
			fmt.Fprintf(os.Stderr, "Failed to poll: %s\n%s\n", resp.Status, string(body))
			continue
		}

		type Updates struct {
			Ok     bool                         `json:"ok"`
			Result []map[string]json.RawMessage `json:"result"`
		}
		var updates Updates
		if err := json.NewDecoder(resp.Body).Decode(&updates); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing update array: %v\n", err)
			continue
		}

		for _, update := range updates.Result {
			for k, v := range update {
				switch k {
				case "update_id":
					var updateId int
					err = json.Unmarshal(v, &updateId)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error parsing update_id: %v\n", err)
						continue
					}
					if updateId >= updateOffset {
						updateOffset = updateId + 1
					}
				case "message":
					type message struct {
						Chat struct {
							Id        int    `json:"id"`
							FirstName string `json:"first_name"`
						} `json:"chat"`
						Text string `json:"text"`
					}
					var msg message
					err = json.Unmarshal(v, &msg)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error parsing message: %v\n", err)
						continue
					}

					fmt.Printf("Message: %s says \"%s\"\n", msg.Chat.FirstName, msg.Text)

					sr := strings.NewReader(msg.Text)
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

					type NewMessage struct {
						ChatId int    `json:"chat_id"`
						Text   string `json:"text"`
					}

					replyMsg := NewMessage{
						ChatId: msg.Chat.Id,
						Text:   sb.String(),
					}

					fmt.Printf("Reply: %s\n", replyMsg.Text)

					b, err := json.Marshal(&replyMsg)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error marshalling response message: %v\n", err)
					}
					resp, err := http.Post(fmt.Sprintf("%s/sendMessage", url), "application/json", bytes.NewReader(b))
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error sending response message: %v\n", err)
					}
					resp.Body.Close()
				}
			}
		}
	}
}
func apiHandler(token string) func(w http.ResponseWriter, r *http.Request) {
	apiUrl := fmt.Sprintf("https://api.telegram.org/bot%s", token)
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

				type NewMessage struct {
					ChatId string `json:"chat_id"`
					Text   string `json:"text"`
				}

				replyMsg := NewMessage{
					ChatId: parts[1],
					Text:   text,
				}

				b, err := json.Marshal(&replyMsg)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "Error sending message")
					fmt.Fprintf(os.Stderr, "Error marshalling response message: %v\n", err)
					return
				}

				resp, err := http.Post(fmt.Sprintf("%s/sendMessage", apiUrl), "application/json", bytes.NewReader(b))
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "Error sending message")
					fmt.Fprintf(os.Stderr, "Error sending response message: %v\n", err)
					return
				}

				if resp.StatusCode != http.StatusOK {
					w.WriteHeader(http.StatusInternalServerError)
					fmt.Fprintf(w, "Error sending message")

					body, err := io.ReadAll(resp.Body)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Error sending response message: %d, failed to read body: %v\n", resp.StatusCode, err)
						return
					}
					fmt.Fprintf(os.Stderr, "Error sending response message: %d, response: %q\n", resp.StatusCode, body)
					resp.Body.Close()
					return
				}

				resp.Body.Close()
				w.WriteHeader(http.StatusOK)
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

	fmt.Println("Starting bot service")
	go serveBot(token)
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	fmt.Printf("Starting api service on %s\n", addr)
	http.HandleFunc("/", apiHandler(token))
	http.ListenAndServe(addr, nil)
}
