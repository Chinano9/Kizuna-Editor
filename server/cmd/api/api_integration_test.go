package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"kizuna/server/internal/sync"
	"kizuna/shared/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type apiEnvelope struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func TestKizunaCloudAPI_EndToEndFlow(t *testing.T) {
	cfg := Config{
		Port:      "8080",
		JWTSecret: "integration-test-secret-32-bytes-long",
		TokenTTL:  time.Hour,
	}

	container := NewAppContainer(cfg)
	router := SetupRouter(container)
	server := httptest.NewServer(router)
	defer server.Close()

	client := server.Client()

	// 1. Test Health Check
	t.Run("Health check", func(t *testing.T) {
		resp, err := client.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("health check request failed: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
	})

	var authToken string
	var userID string

	// 2. Test User Registration
	t.Run("User registration", func(t *testing.T) {
		body, _ := json.Marshal(map[string]string{
			"username": "lead_guitarist",
			"email":    "lead@kizuna.io",
			"password": "securepassword123",
		})
		resp, err := client.Post(server.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatalf("registration failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d", resp.StatusCode)
		}

		var env apiEnvelope
		_ = json.NewDecoder(resp.Body).Decode(&env)
		var authData struct {
			User struct {
				ID string `json:"id"`
			} `json:"user"`
			Token string `json:"token"`
		}
		_ = json.Unmarshal(env.Data, &authData)

		authToken = authData.Token
		userID = authData.User.ID
		if authToken == "" || userID == "" {
			t.Fatalf("expected token and user ID from registration")
		}
	})

	// 3. Test Auth Me
	t.Run("Auth me profile", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/auth/me", nil)
		req.Header.Set("Authorization", "Bearer "+authToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("auth me request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
	})

	var projectID int

	// 4. Test Sync Push (Create Project)
	t.Run("Sync push new project", func(t *testing.T) {
		pushPayload, _ := json.Marshal(sync.PushRequest{
			ProjectID:     0,
			Title:         "Highway Star",
			BPM:           175,
			TimeSignature: "4/4",
			KeySignature:  "G minor",
			Tracks: []models.Track{
				{Name: "Organ Riff", DataContent: ":4 3.4 5.4"},
			},
		})

		req, _ := http.NewRequest("POST", server.URL+"/api/v1/sync/push", bytes.NewReader(pushPayload))
		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("push failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		var env apiEnvelope
		_ = json.NewDecoder(resp.Body).Decode(&env)
		var pushRes sync.PushResponse
		_ = json.Unmarshal(env.Data, &pushRes)

		projectID = pushRes.ProjectID
		if projectID <= 0 || pushRes.NewVersion != 1 {
			t.Errorf("unexpected push result: %+v", pushRes)
		}
	})

	// 5. Test Project Checkout Lock
	t.Run("Checkout project lock", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%d/checkout", server.URL, projectID)
		req, _ := http.NewRequest("POST", url, nil)
		req.Header.Set("Authorization", "Bearer "+authToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("checkout failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
	})

	// 6. Test Project Release Lock
	t.Run("Release project lock", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/projects/%d/release", server.URL, projectID)
		req, _ := http.NewRequest("POST", url, nil)
		req.Header.Set("Authorization", "Bearer "+authToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("release lock failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
	})

	// 7. Test Pull Project
	t.Run("Pull project", func(t *testing.T) {
		url := fmt.Sprintf("%s/api/v1/sync/pull?project_id=%d", server.URL, projectID)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("Authorization", "Bearer "+authToken)

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("pull failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		var env apiEnvelope
		_ = json.NewDecoder(resp.Body).Decode(&env)
		var pulled sync.CloudProject
		_ = json.Unmarshal(env.Data, &pulled)

		if pulled.Title != "Highway Star" || len(pulled.Tracks) != 1 {
			t.Errorf("pulled project mismatch: %+v", pulled)
		}
	})
}
