package api

import (
	"fmt"
	"message-relay-bot/pkg/tgapi"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Api struct {
	tg tgapi.Api
}

func New(tg tgapi.Api) *Api {
	return &Api{tg: tg}
}

func (api *Api) ServeAddr(addr string) {
	r := mux.NewRouter()
	r.StrictSlash(true)
	r.HandleFunc("/", api.serveRoot).Methods("GET")
	r.HandleFunc("/u/{id}/message", api.userMessagePage).Methods("GET")
	r.HandleFunc("/u/{id}/message", api.userMessageSend).Methods("POST")
	http.ListenAndServe(addr, r)
}

func (Api) serveRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Message Relay Bot API")
}

func (Api) userMessagePage(w http.ResponseWriter, r *http.Request) {
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

func (api *Api) userMessageSend(w http.ResponseWriter, r *http.Request) {
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

	_, err = api.tg.SendMessage(&tgapi.NewMessage{
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
