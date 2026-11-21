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

	// 8. Test Share Project with Collaborator (Issue #15)
	t.Run("Share project with collaborator", func(t *testing.T) {
		shareURL := fmt.Sprintf("%s/api/v1/projects/%d/share", server.URL, projectID)
		body, _ := json.Marshal(map[string]string{
			"collaborator": "ritchie@deep-purple.com",
		})
		req, _ := http.NewRequest("POST", shareURL, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+authToken)
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("share request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
		}

		// Sharing the same collaborator again should return 409 Conflict
		req2, _ := http.NewRequest("POST", shareURL, bytes.NewReader(body))
		req2.Header.Set("Authorization", "Bearer "+authToken)
		req2.Header.Set("Content-Type", "application/json")
		resp2, err := client.Do(req2)
		if err != nil {
			t.Fatalf("duplicate share request failed: %v", err)
		}
		defer resp2.Body.Close()

		if resp2.StatusCode != http.StatusConflict {
			t.Errorf("expected 409 Conflict on duplicate share, got %d", resp2.StatusCode)
		}

		// Sharing a non-existent project should return 404
		nonExistentURL := fmt.Sprintf("%s/api/v1/projects/%d/share", server.URL, 99999)
		req3, _ := http.NewRequest("POST", nonExistentURL, bytes.NewReader(body))
		req3.Header.Set("Authorization", "Bearer "+authToken)
		req3.Header.Set("Content-Type", "application/json")
		resp3, err := client.Do(req3)
		if err != nil {
			t.Fatalf("non-existent share request failed: %v", err)
		}
		defer resp3.Body.Close()

		if resp3.StatusCode != http.StatusNotFound {
			t.Errorf("expected 404 Not Found, got %d", resp3.StatusCode)
		}
	})

	// 9. Test Collaborative Checkout Lock Rejection & Auto-Release on Push (Issues #16, #22)
	t.Run("Checkout conflict and auto-release on push", func(t *testing.T) {
		// Register second user (collaborator)
		regBody, _ := json.Marshal(map[string]string{
			"username": "ritchie_blackmore",
			"email":    "ritchie@deep-purple.com",
			"password": "guitarpassword123",
		})
		regResp, err := client.Post(server.URL+"/api/v1/auth/register", "application/json", bytes.NewReader(regBody))
		if err != nil {
			t.Fatalf("second user registration failed: %v", err)
		}
		defer regResp.Body.Close()

		var regEnv apiEnvelope
		_ = json.NewDecoder(regResp.Body).Decode(&regEnv)
		var authData2 struct {
			Token string `json:"token"`
		}
		_ = json.Unmarshal(regEnv.Data, &authData2)
		token2 := authData2.Token

		// User 1 acquires checkout lock
		checkoutURL := fmt.Sprintf("%s/api/v1/projects/%d/checkout", server.URL, projectID)
		reqLock1, _ := http.NewRequest("POST", checkoutURL, nil)
		reqLock1.Header.Set("Authorization", "Bearer "+authToken)
		respLock1, err := client.Do(reqLock1)
		if err != nil || respLock1.StatusCode != http.StatusOK {
			t.Fatalf("user 1 failed to acquire checkout lock: %v (status %d)", err, respLock1.StatusCode)
		}
		respLock1.Body.Close()

		// User 2 attempts to checkout while User 1 holds lock -> 409 Conflict (Issue #16)
		reqLock2, _ := http.NewRequest("POST", checkoutURL, nil)
		reqLock2.Header.Set("Authorization", "Bearer "+token2)
		respLock2, err := client.Do(reqLock2)
		if err != nil {
			t.Fatalf("user 2 checkout request failed: %v", err)
		}
		defer respLock2.Body.Close()
		if respLock2.StatusCode != http.StatusConflict {
			t.Errorf("expected 409 Conflict when user 2 checks out locked project, got %d", respLock2.StatusCode)
		}

		// User 2 attempts to push while User 1 holds lock -> 409 Conflict (Issue #16)
		pushBody2, _ := json.Marshal(sync.PushRequest{
			ProjectID:   projectID,
			Title:       "Highway Star (Unauthorized Overwrite)",
			BPM:         175,
			BaseVersion: 1,
		})
		reqPush2, _ := http.NewRequest("POST", server.URL+"/api/v1/sync/push", bytes.NewReader(pushBody2))
		reqPush2.Header.Set("Authorization", "Bearer "+token2)
		reqPush2.Header.Set("Content-Type", "application/json")
		respPush2, err := client.Do(reqPush2)
		if err != nil {
			t.Fatalf("user 2 push request failed: %v", err)
		}
		defer respPush2.Body.Close()
		if respPush2.StatusCode != http.StatusConflict {
			t.Errorf("expected 409 Conflict when user 2 pushes to locked project, got %d", respPush2.StatusCode)
		}

		// User 1 pushes update -> succeeds AND auto-releases the lock (Issue #22)
		pushBody1, _ := json.Marshal(sync.PushRequest{
			ProjectID:   projectID,
			Title:       "Highway Star (v2 Lead Added)",
			BPM:         175,
			BaseVersion: 1,
		})
		reqPush1, _ := http.NewRequest("POST", server.URL+"/api/v1/sync/push", bytes.NewReader(pushBody1))
		reqPush1.Header.Set("Authorization", "Bearer "+authToken)
		reqPush1.Header.Set("Content-Type", "application/json")
		respPush1, err := client.Do(reqPush1)
		if err != nil {
			t.Fatalf("user 1 push failed: %v", err)
		}
		defer respPush1.Body.Close()
		if respPush1.StatusCode != http.StatusOK {
			t.Fatalf("expected 200 OK on user 1 push, got %d", respPush1.StatusCode)
		}

		// Now that User 1's push auto-released the lock, User 2 can acquire checkout lock (Issue #22)
		reqLock2Retry, _ := http.NewRequest("POST", checkoutURL, nil)
		reqLock2Retry.Header.Set("Authorization", "Bearer "+token2)
		respLock2Retry, err := client.Do(reqLock2Retry)
		if err != nil {
			t.Fatalf("user 2 retry checkout failed: %v", err)
		}
		defer respLock2Retry.Body.Close()
		if respLock2Retry.StatusCode != http.StatusOK {
			t.Errorf("expected 200 OK for user 2 checkout after auto-release, got %d", respLock2Retry.StatusCode)
		}
	})
}
