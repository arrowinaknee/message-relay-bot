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

func main() {
	token := os.Getenv("TG_BOT_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Missing env var: TG_BOT_TOKEN")
		os.Exit(1)
	}
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
				fmt.Fprintf(os.Stderr, "Failed to poll: %s, error on body read: %v\n", resp.Status, err)
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
