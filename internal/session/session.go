package session

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Session struct {
	Branch      string   `json:"branch"`
	Description string   `json:"description"`
	Plan        []string `json:"plan"`
	Ticket      string   `json:"ticket,omitempty"`
}

const sessionPath = ".sg/session.json"

func Save(s Session) error {
	dir := filepath.Dir(sessionPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(sessionPath, data, 0644)
}

func Load() (*Session, error) {
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, err
	}

	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

func Exists() bool {
	_, err := os.Stat(sessionPath)
	return err == nil
}
