package handlers

import (
	"encoding/json"
	"fmt"
	"log"
)

type HandlerFunc func(payload []byte) error

func SendEmail(payload []byte) error {
	var data struct {
		To      string `json:"to"`
		Subject string `json:"subject"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	if data.To == "" {
		return fmt.Errorf("missing required field: to")
	}

	log.Printf("[send_email] sending email to %s, subject: %s", data.To, data.Subject)
	return nil
}

func ProcessFile(payload []byte) error {
	var data struct {
		Filename string `json:"filename"`
	}

	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("invalid payload: %w", err)
	}

	if data.Filename == "" {
		return fmt.Errorf("missing required field: filename")
	}

	log.Printf("[process_file] processing file: %s", data.Filename)
	return nil
}