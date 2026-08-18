package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/KawaiiSh1zuku/SimpleBlog/internal/repository"
	"github.com/google/uuid"
)

const (
	defaultManagedPageSize = 50
	maxManagedPageSize     = 100
	managedCursorVersion   = 1
)

type managedCursorPayload struct {
	Version int    `json:"v"`
	Kind    string `json:"kind"`
	At      string `json:"at"`
	ID      string `json:"id"`
}

func parseManagedPage(limitValue, cursorValue, kind string) (int, *repository.TimeCursor, error) {
	limit := defaultManagedPageSize
	if value := strings.TrimSpace(limitValue); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > maxManagedPageSize {
			return 0, nil, ErrInvalidInput
		}
		limit = parsed
	}
	cursorValue = strings.TrimSpace(cursorValue)
	if cursorValue == "" {
		return limit, nil, nil
	}
	if len(cursorValue) > 512 {
		return 0, nil, ErrInvalidInput
	}
	encoded, err := base64.RawURLEncoding.DecodeString(cursorValue)
	if err != nil {
		return 0, nil, ErrInvalidInput
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	var payload managedCursorPayload
	if err := decoder.Decode(&payload); err != nil {
		return 0, nil, ErrInvalidInput
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return 0, nil, ErrInvalidInput
	}
	if payload.Version != managedCursorVersion || payload.Kind != kind {
		return 0, nil, ErrInvalidInput
	}
	at, err := time.Parse(time.RFC3339Nano, payload.At)
	if err != nil {
		return 0, nil, ErrInvalidInput
	}
	id, err := uuid.Parse(payload.ID)
	if err != nil || id == uuid.Nil {
		return 0, nil, ErrInvalidInput
	}
	return limit, &repository.TimeCursor{At: at, ID: id}, nil
}

func encodeManagedCursor(kind string, cursor repository.TimeCursor) (string, error) {
	payload, err := json.Marshal(managedCursorPayload{
		Version: managedCursorVersion, Kind: kind,
		At: cursor.At.UTC().Format(time.RFC3339Nano), ID: cursor.ID.String(),
	})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}
