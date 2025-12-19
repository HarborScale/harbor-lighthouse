package transport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type CargoPayload struct {
	Time    string      `json:"time"`
	ShipID  string      `json:"ship_id"`
	CargoID string      `json:"cargo_id"`
	Value   interface{} `json:"value"`
}

func Send(url, apiKey string, payload interface{}) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil { return err }

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil { return err }

	req.Header.Set("Content-Type", "application/json")
	// Only add API key if it's a real value
	if apiKey != "" && apiKey != "undefined" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("API Error %s", resp.Status)
	}
	return nil
}
