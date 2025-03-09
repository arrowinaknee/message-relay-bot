package tgapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Api interface {
	GetUpdates(offset int, timeout int, limit int, allowedUpdates []string) ([]Update, error)
	SendMessage(message *NewMessage) (*Message, error)
}

type apiImpl struct {
	url string
}

func New(token string) Api {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/", token)
	return &apiImpl{url: url}
}

func (api *apiImpl) GetUpdates(offset int, timeout int, limit int, allowedUpdates []string) ([]Update, error) {
	if allowedUpdates == nil {
		allowedUpdates = []string{}
	}
	params := map[string]interface{}{
		"offset":          offset,
		"limit":           limit,
		"timeout":         timeout,
		"allowed_updates": allowedUpdates,
	}
	var rawUpdates []map[string]json.RawMessage
	err := api.get("getUpdates", params, &rawUpdates)
	if err != nil {
		return nil, fmt.Errorf("tgapi: GetUpdates: %w", err)
	}
	updates := make([]Update, 0, len(rawUpdates))
	for _, raw := range rawUpdates {
		var update Update
		for k, v := range raw {
			switch k {
			case "update_id":
				err = json.Unmarshal(v, &update.Id)
				if err != nil {
					return nil, fmt.Errorf("tgapi: GetUpdates: %w", err)
				}
			case "message":
				msg := new(Message)
				err = json.Unmarshal(v, msg)
				if err != nil {
					return nil, fmt.Errorf("tgapi: GetUpdates: %w", err)
				}
				update.Value = msg
			default:
				continue
			}
		}
		updates = append(updates, update)
	}
	return updates, nil
}

func (api *apiImpl) SendMessage(message *NewMessage) (*Message, error) {
	var resp Message
	err := api.post("sendMessage", message, &resp)
	if err != nil {
		return nil, fmt.Errorf("tgapi: SendMessage: %w", err)
	}
	return &resp, nil
}

type apiResponse struct {
	Ok          bool            `json:"ok"`
	Description string          `json:"description"`
	ErrorCode   int             `json:"error_code"`
	Result      json.RawMessage `json:"result"`
}

func (api *apiImpl) get(endpoint string, request map[string]interface{}, response interface{}) error {
	var query strings.Builder
	for k, v := range request {
		if query.Len() > 0 {
			query.WriteString("&")
		}
		query.WriteString(k)
		query.WriteString("=")
		enc, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to encode request: %w", err)
		}
		query.WriteString(url.QueryEscape(string(enc)))
	}

	resp, err := http.Get(api.url + endpoint + "?" + query.String())
	if err != nil {
		return fmt.Errorf("failed to get %q: %w", endpoint, err)
	}
	defer resp.Body.Close()

	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return fmt.Errorf("failed to get %q: status %d, decode response: %w", endpoint, resp.StatusCode, err)
	}
	if !ar.Ok {
		return fmt.Errorf("failed to get %q: status %d, description: %s", endpoint, ar.ErrorCode, ar.Description)
	}
	if err := json.Unmarshal(ar.Result, response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return nil
}

func (api *apiImpl) post(endpoint string, request interface{}, response interface{}) error {
	body, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}
	resp, err := http.Post(api.url+endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to post %q: %w", endpoint, err)
	}
	defer resp.Body.Close()

	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return fmt.Errorf("failed to post %q: status %d, decode response: %w", endpoint, resp.StatusCode, err)
	}
	if !ar.Ok {
		return fmt.Errorf("failed to post %q: status %d, description: %s", endpoint, ar.ErrorCode, ar.Description)
	}
	if err := json.Unmarshal(ar.Result, response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}
	return nil
}
