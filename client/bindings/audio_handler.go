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

package bindings

import (
	"context"
	"fmt"
	"kizuna/client/internal/audio"
	"kizuna/client/internal/service"
	"kizuna/shared/models"
	"log"
)

// AudioHandler coordinates audio capture, playback, and versioning for Wails.
type AudioHandler struct {
	audioRecorder *audio.AudioRecorder
	audioPlayer   *audio.AudioPlayer
	audioService  *service.AudioService
}

// NewAudioHandler instantiates an AudioHandler.
func NewAudioHandler(
	recorder *audio.AudioRecorder,
	player *audio.AudioPlayer,
	audioService *service.AudioService,
) *AudioHandler {
	return &AudioHandler{
		audioRecorder: recorder,
		audioPlayer:   player,
		audioService:  audioService,
	}
}

// StartRecording starts audio capture on the specified input device.
func (h *AudioHandler) StartRecording(songID int, trackName string, deviceID string) error {
	log.Printf("[AudioHandler] StartRecording: Song ID %d, Track '%s', Device: %s", songID, trackName, deviceID)
	if h.audioRecorder == nil {
		return fmt.Errorf("audio recorder is not initialized")
	}
	return h.audioRecorder.StartRecording(songID, trackName, deviceID)
}

// StopRecording halts capture, writes WAV file to disk, registers the take, and returns the AudioVersion.
func (h *AudioHandler) StopRecording(trackID int) (*models.AudioVersion, error) {
	log.Printf("[AudioHandler] StopRecording: Target Track ID: %d", trackID)
	if h.audioRecorder == nil {
		return nil, fmt.Errorf("audio recorder is not initialized")
	}

	songID := h.audioRecorder.SongID
	trackName := h.audioRecorder.TrackName

	pcmData, sampleRate, channels, bitsPerSample, err := h.audioRecorder.StopRecording()
	if err != nil {
		return nil, err
	}

	wavPath, err := audio.ResolveProjectAudioPath(songID, trackName, trackID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve audio path: %w", err)
	}

	log.Printf("[AudioHandler] Writing WAV file to: %s", wavPath)
	if err := audio.WriteWavFile(wavPath, pcmData, sampleRate, channels, bitsPerSample); err != nil {
		return nil, fmt.Errorf("failed to save WAV file: %w", err)
	}

	var dbTrackID *int
	var versionName string
	if trackID > 0 {
		dbTrackID = &trackID
		versionName = fmt.Sprintf("Take: %s", trackName)
	} else {
		dbTrackID = nil
		versionName = fmt.Sprintf("Global take: %s", trackName)
	}

	log.Printf("[AudioHandler] Registering AudioVersion '%s' in database...", versionName)
	av, err := h.audioService.RegisterAudioVersion(context.Background(), songID, dbTrackID, versionName, wavPath, "Native Go recorded take")
	if err != nil {
		return nil, fmt.Errorf("failed to register audio version: %w", err)
	}

	log.Printf("[AudioHandler] Audio version registered: ID %d", av.ID)
	return av, nil
}

// GetAudioLevels returns the current normalized peak level [0.0, 1.0].
func (h *AudioHandler) GetAudioLevels() float64 {
	if h.audioRecorder == nil {
		return 0.0
	}
	return h.audioRecorder.GetAudioLevels()
}

// GetInputDevices enumerates all available physical capture devices.
func (h *AudioHandler) GetInputDevices() ([]audio.AudioDevice, error) {
	return audio.GetInputDevices()
}

// GetAvailableDrivers returns audio driver backend options for the current OS.
func (h *AudioHandler) GetAvailableDrivers() []string {
	return audio.GetAvailableDrivers()
}

// GetCurrentDriver returns the active audio driver backend name.
func (h *AudioHandler) GetCurrentDriver() string {
	return audio.GetCurrentDriver()
}

// SetAudioDriver switches the active audio backend.
func (h *AudioHandler) SetAudioDriver(driverName string) error {
	log.Printf("[AudioHandler] Transitioning to driver '%s'", driverName)
	return audio.SetAudioDriver(driverName)
}

// PlayAudio streams a WAV file from disk.
func (h *AudioHandler) PlayAudio(filePath string) error {
	log.Printf("[AudioHandler] PlayAudio: %s", filePath)
	if h.audioPlayer == nil {
		return fmt.Errorf("audio player is not initialized")
	}
	return h.audioPlayer.Play(filePath)
}

// PauseAudio pauses or resumes the active playback stream.
func (h *AudioHandler) PauseAudio() error {
	if h.audioPlayer == nil {
		return fmt.Errorf("audio player is not initialized")
	}
	return h.audioPlayer.Pause()
}

// SeekAudio repositions playback progress to the specified second mark.
func (h *AudioHandler) SeekAudio(seconds float64) error {
	if h.audioPlayer == nil {
		return fmt.Errorf("audio player is not initialized")
	}
	return h.audioPlayer.Seek(seconds)
}

// SetAudioVolume updates the output volume multiplier [0.0, 1.0].
func (h *AudioHandler) SetAudioVolume(volume float64) error {
	if h.audioPlayer == nil {
		return fmt.Errorf("audio player is not initialized")
	}
	return h.audioPlayer.SetVolume(volume)
}

// GetPlaybackPosition returns the current playback progress in seconds.
func (h *AudioHandler) GetPlaybackPosition() float64 {
	if h.audioPlayer == nil {
		return 0.0
	}
	return h.audioPlayer.GetPosition()
}

// GetPlaybackDuration returns the total length of the loaded audio in seconds.
func (h *AudioHandler) GetPlaybackDuration() float64 {
	if h.audioPlayer == nil {
		return 0.0
	}
	return h.audioPlayer.GetDuration()
}

// GetAudioVersionsForTrack retrieves all takes for a track.
func (h *AudioHandler) GetAudioVersionsForTrack(trackID int) ([]models.AudioVersion, error) {
	return h.audioService.GetAudioVersionsForTrack(context.Background(), trackID)
}

// GetAudioVersionsForSong retrieves all takes for a song.
func (h *AudioHandler) GetAudioVersionsForSong(songID int) ([]models.AudioVersion, error) {
	return h.audioService.GetAudioVersionsForSong(context.Background(), songID)
}
