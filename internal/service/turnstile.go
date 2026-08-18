package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const turnstileVerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"

type TurnstileService struct {
	Secret string
	Client *http.Client
}

type turnstileResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
	Hostname   string   `json:"hostname"`
	Action     string   `json:"action"`
}

func NewTurnstileService(secret string) *TurnstileService {
	return &TurnstileService{
		Secret: secret,
		Client: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *TurnstileService) Verify(
	ctx context.Context,
	token string,
	remoteIP string,
) error {
	if token == "" {
		return fmt.Errorf("turnstile token is empty")
	}

	data := url.Values{}
	data.Set("secret", s.Secret)
	data.Set("response", token)

	if remoteIP != "" {
		data.Set("remoteip", remoteIP)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		turnstileVerifyURL,
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return err
	}

	req.Header.Set(
		"Content-Type",
		"application/x-www-form-urlencoded",
	)

	resp, err := s.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("turnstile returned HTTP %d", resp.StatusCode)
	}

	var result turnstileResponse

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}

	if !result.Success {
		return fmt.Errorf(
			"turnstile verification failed: %v",
			result.ErrorCodes,
		)
	}

	return nil
}
