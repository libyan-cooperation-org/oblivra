package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/websocket"
)

type Client struct {
	BaseURL string
	Token   string
}

func loadToken() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	configPath := filepath.Join(home, ".oblivrashell", "cli.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	var config struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &config); err != nil {
		return ""
	}
	return config.Token
}

func saveToken(token string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".oblivrashell")
	os.MkdirAll(dir, 0700)

	configPath := filepath.Join(dir, "cli.json")
	data, _ := json.MarshalIndent(map[string]string{"token": token}, "", "  ")
	return os.WriteFile(configPath, data, 0600)
}

func (c *Client) Search(query string, limit int) error {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"query": query,
		"filters": map[string]interface{}{
			"limit": limit,
		},
	})

	req, err := http.NewRequest("POST", c.BaseURL+"/siem/search", bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	// Pretty print JSON response
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, body, "", "  "); err != nil {
		fmt.Println(string(body))
	} else {
		fmt.Println(prettyJSON.String())
	}
	return nil
}

func (c *Client) StreamEvents() error {
	wsURL := "ws" + c.BaseURL[4:] + "/events" // replace http with ws

	header := http.Header{}
	if c.Token != "" {
		header.Set("Authorization", "Bearer "+c.Token)
	}

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer conn.Close()

	fmt.Printf("Connected to event stream at %s\n", wsURL)

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}

		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, message, "", "  "); err != nil {
			fmt.Printf("%s\n", message)
		} else {
			fmt.Printf("%s\n", prettyJSON.String())
		}
	}
}
