package main

import (
	"fmt"
	"math/rand"
	"message-relay-bot/pkg/api"
	"message-relay-bot/pkg/tgapi"
	"os"
	"strings"
	"time"
	"unicode"
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

func main() {
	token := os.Getenv("TG_BOT_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "Missing env var: TG_BOT_TOKEN")
		os.Exit(1)
	}
	tg := tgapi.New(token)

	fmt.Println("Starting bot service")
	go serveBot(tg)

	api := api.New(tg)
	addr := os.Getenv("API_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	fmt.Printf("Starting api service on %s\n", addr)
	api.ServeAddr(addr)
}
