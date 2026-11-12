/*
Kizuna Editor - A local-first songwriting environment.
Copyright (C) 2025 Fernando Ponce Solis (@Chinano9)

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"kizuna/client/internal/repository"
	"kizuna/shared/models"
	"net/http"
	"strings"
	"time"
)

// SyncClientService coordinates local project packaging and cloud synchronization (Issue #13).
type SyncClientService struct {
	songRepo  repository.SongRepository
	trackRepo repository.TrackRepository
	audioRepo repository.AudioRepository
	client    *http.Client
}

// NewSyncClientService instantiates a SyncClientService with HTTP client timeout.
func NewSyncClientService(
	songRepo repository.SongRepository,
	trackRepo repository.TrackRepository,
	audioRepo repository.AudioRepository,
) *SyncClientService {
	return &SyncClientService{
		songRepo:  songRepo,
		trackRepo: trackRepo,
		audioRepo: audioRepo,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PackageProject gathers song metadata, tracks, and audio versions into a sync payload.
func (s *SyncClientService) PackageProject(ctx context.Context, songID int, baseVersion int) (*models.CloudPushPayload, error) {
	if songID <= 0 {
		return nil, fmt.Errorf("invalid song ID: %d", songID)
	}

	song, err := s.songRepo.GetByID(ctx, songID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve song: %w", err)
	}

	tracks, err := s.trackRepo.GetBySongID(ctx, songID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve tracks for song %d: %w", songID, err)
	}

	audioVersions, err := s.audioRepo.GetBySongID(ctx, songID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve audio versions for song %d: %w", songID, err)
	}

	return &models.CloudPushPayload{
		ProjectID:     song.ID,
		Title:         song.Title,
		BPM:           song.BPM,
		TimeSignature: song.TimeSignature,
		KeySignature:  song.KeySignature,
		BaseVersion:   baseVersion,
		Tracks:        tracks,
		AudioVersions: audioVersions,
	}, nil
}

// PushProjectToCloud packages the local song and pushes it to the Kizuna Cloud server.
func (s *SyncClientService) PushProjectToCloud(
	ctx context.Context,
	cloudURL string,
	token string,
	songID int,
	baseVersion int,
) (*models.CloudPushResult, error) {
	if strings.TrimSpace(cloudURL) == "" {
		cloudURL = "http://localhost:8080"
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("authentication token is required to sync with Kizuna Cloud")
	}

	payload, err := s.PackageProject(ctx, songID, baseVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to package project: %w", err)
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize sync payload: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/v1/sync/push", strings.TrimRight(cloudURL, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create sync request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Kizuna Cloud at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Success bool                    `json:"success"`
		Data    *models.CloudPushResult `json:"data,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details string `json:"details"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("invalid server response (HTTP %d): %w", resp.StatusCode, err)
	}

	if !apiResp.Success || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("sync push rejected (HTTP %d)", resp.StatusCode)
		if apiResp.Error != nil && apiResp.Error.Message != "" {
			errMsg = apiResp.Error.Message
		}
		return nil, fmt.Errorf("%s", errMsg)
	}

	if apiResp.Data == nil {
		return nil, fmt.Errorf("server returned empty data in response")
	}

	return apiResp.Data, nil
}

// ImportCloudProject imports a pulled cloud project into local SQLite tables (Issue #14).
func (s *SyncClientService) ImportCloudProject(ctx context.Context, cp *models.CloudProject) (int, error) {
	if cp == nil {
		return 0, fmt.Errorf("cloud project cannot be nil")
	}

	title := strings.TrimSpace(cp.Title)
	if title == "" {
		title = "Imported Cloud Project"
	}

	song := &models.Song{
		Title:         title,
		BPM:           cp.BPM,
		TimeSignature: cp.TimeSignature,
		KeySignature:  cp.KeySignature,
	}
	if song.BPM <= 0 {
		song.BPM = 120
	}
	if song.TimeSignature == "" {
		song.TimeSignature = "4/4"
	}

	songID, err := s.songRepo.Create(ctx, nil, song)
	if err != nil {
		return 0, fmt.Errorf("failed to create local song: %w", err)
	}

	for _, tr := range cp.Tracks {
		instID := tr.InstrumentID
		if instID <= 0 {
			instID = 1
		}
		mode := tr.DisplayMode
		if mode == "" {
			mode = models.DisplayModeBoth
		}

		newTrack := &models.Track{
			SongID:       songID,
			InstrumentID: instID,
			Name:         tr.Name,
			DataContent:  tr.DataContent,
			DisplayMode:  mode,
			IsMuted:      tr.IsMuted,
		}
		_, err := s.trackRepo.Create(ctx, nil, newTrack)
		if err != nil {
			return 0, fmt.Errorf("failed to create local track '%s': %w", tr.Name, err)
		}
	}

	for _, av := range cp.AudioVersions {
		newAV := &models.AudioVersion{
			SongID:      songID,
			VersionName: av.VersionName,
			FilePath:    av.FilePath,
			Notes:       av.Notes,
		}
		_, _ = s.audioRepo.Create(ctx, newAV)
	}

	return songID, nil
}

// PullProjectFromCloud downloads a project from Kizuna Cloud and unpacks it into the local database (Issue #14).
func (s *SyncClientService) PullProjectFromCloud(
	ctx context.Context,
	cloudURL string,
	token string,
	projectID int,
) (*models.CloudProject, int, error) {
	if strings.TrimSpace(cloudURL) == "" {
		cloudURL = "http://localhost:8080"
	}
	if strings.TrimSpace(token) == "" {
		return nil, 0, fmt.Errorf("authentication token is required to pull from Kizuna Cloud")
	}
	if projectID <= 0 {
		return nil, 0, fmt.Errorf("invalid cloud project ID: %d", projectID)
	}

	endpoint := fmt.Sprintf("%s/api/v1/sync/pull?project_id=%d", strings.TrimRight(cloudURL, "/"), projectID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create pull request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to connect to Kizuna Cloud at %s: %w", cloudURL, err)
	}
	defer resp.Body.Close()

	var apiResp struct {
		Success bool                 `json:"success"`
		Data    *models.CloudProject `json:"data,omitempty"`
		Error   *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Details string `json:"details"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, 0, fmt.Errorf("invalid server response (HTTP %d): %w", resp.StatusCode, err)
	}

	if !apiResp.Success || resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errMsg := fmt.Sprintf("sync pull rejected (HTTP %d)", resp.StatusCode)
		if apiResp.Error != nil && apiResp.Error.Message != "" {
			errMsg = apiResp.Error.Message
		}
		return nil, 0, fmt.Errorf("%s", errMsg)
	}

	if apiResp.Data == nil {
		return nil, 0, fmt.Errorf("server returned empty data for project %d", projectID)
	}

	localSongID, err := s.ImportCloudProject(ctx, apiResp.Data)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to import cloud project into local database: %w", err)
	}

	return apiResp.Data, localSongID, nil
}
