package sync

import (
	"errors"
	"kizuna/server/internal/collaboration"
	"kizuna/shared/models"
	"testing"
)

func TestSyncEngine_PushPullAndConflicts(t *testing.T) {
	locker := collaboration.NewLocker()
	engine := NewEngine(locker)

	user1 := "user_dev1"
	user2 := "user_dev2"

	// 1. Initial Push: Create new project in cloud
	pushReq := PushRequest{
		ProjectID:     0,
		Title:         "Stairway Idea",
		BPM:           78,
		TimeSignature: "4/4",
		Tracks: []models.Track{
			{Name: "12-String Guitar", DataContent: ":4 5.3 7.3"},
		},
		AudioVersions: []models.AudioVersion{
			{VersionName: "Intro Take", FilePath: "/audio/intro.wav"},
		},
	}

	pushRes, err := engine.Push(user1, pushReq)
	if err != nil {
		t.Fatalf("failed initial push: %v", err)
	}
	if pushRes.ProjectID <= 0 || pushRes.NewVersion != 1 {
		t.Errorf("unexpected push response: %+v", pushRes)
	}

	projectID := pushRes.ProjectID

	// 2. Pull project from cloud
	cloudProject, err := engine.Pull(projectID)
	if err != nil {
		t.Fatalf("failed to pull project: %v", err)
	}
	if cloudProject.Title != "Stairway Idea" || len(cloudProject.Tracks) != 1 {
		t.Errorf("cloud project content mismatch: %+v", cloudProject)
	}

	// 3. Incremental update with correct base version (v1 -> v2)
	updateReq := PushRequest{
		ProjectID:     projectID,
		Title:         "Stairway to Heaven (Full)",
		BPM:           82,
		TimeSignature: "4/4",
		BaseVersion:   1,
		Tracks: []models.Track{
			{Name: "Acoustic", DataContent: ":4 5.3 7.3"},
			{Name: "Electric Solo", DataContent: ":4 12.2 15.2"},
		},
	}

	updateRes, err := engine.Push(user1, updateReq)
	if err != nil {
		t.Fatalf("failed update push: %v", err)
	}
	if updateRes.NewVersion != 2 {
		t.Errorf("expected version 2, got %d", updateRes.NewVersion)
	}

	// 4. Conflict Detection: Sending push based on outdated base version v1 returns ErrVersionConflict
	staleReq := PushRequest{
		ProjectID:   projectID,
		Title:       "Outdated Edit",
		BaseVersion: 1, // cloud is now at v2!
	}
	_, err = engine.Push(user1, staleReq)
	if !errors.Is(err, ErrVersionConflict) {
		t.Errorf("expected ErrVersionConflict, got %v", err)
	}

	// 5. Unauthorized push by user2 returns ErrUnauthorizedSync
	_, err = engine.Push(user2, updateReq)
	if !errors.Is(err, ErrUnauthorizedSync) {
		t.Errorf("expected ErrUnauthorizedSync for user2, got %v", err)
	}

	// 6. Test Diff
	diff, err := engine.GetDiff(projectID, 1)
	if err != nil {
		t.Fatalf("failed to get diff: %v", err)
	}
	if !diff.HasChanges || diff.ClientVersion != 1 || diff.CloudVersion != 2 {
		t.Errorf("diff mismatch: %+v", diff)
	}
}
