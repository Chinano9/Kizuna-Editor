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

	// 7. Share project with user2 and test Checkout Lock validation & Auto-Release on Push (Issues #16, #22)
	if err := engine.Share(projectID, user1, user2); err != nil {
		t.Fatalf("failed to share project with user2: %v", err)
	}

	// user2 acquires checkout lock
	_, err = locker.AcquireLock(projectID, user2, "Dev2", 0)
	if err != nil {
		t.Fatalf("failed for user2 to acquire lock: %v", err)
	}

	// user1 (owner) tries to push while user2 holds lock -> must return ErrProjectLocked (Issue #16)
	lockedPushReq := PushRequest{
		ProjectID:   projectID,
		Title:       "Owner Edit While Locked",
		BaseVersion: 2,
	}
	_, err = engine.Push(user1, lockedPushReq)
	if !errors.Is(err, collaboration.ErrProjectLocked) {
		t.Fatalf("expected ErrProjectLocked when pushing while locked by user2, got %v", err)
	}

	// user2 (lock holder & collaborator) pushes v2 -> v3 -> succeeds AND automatically releases lock (Issue #22)
	collabPushReq := PushRequest{
		ProjectID:   projectID,
		Title:       "Stairway to Heaven (Dev2 Bass & Drums)",
		BPM:         82,
		BaseVersion: 2,
	}
	collabRes, err := engine.Push(user2, collabPushReq)
	if err != nil {
		t.Fatalf("expected collaborator user2 holding lock to push successfully, got %v", err)
	}
	if collabRes.NewVersion != 3 {
		t.Errorf("expected version 3, got %d", collabRes.NewVersion)
	}
	if !collabRes.LockReleased {
		t.Errorf("expected LockReleased to be true after push by lock holder")
	}

	// Verify lock was automatically removed from Locker (Issue #22)
	if _, stillLocked := locker.GetLock(projectID); stillLocked {
		t.Errorf("expected lock to be released automatically after push, but project is still locked")
	}
}

func TestSyncEngine_OfflineLocalFirstReconciliation(t *testing.T) {
	locker := collaboration.NewLocker()
	engine := NewEngine(locker)

	owner := "user_owner"
	collab := "user_collab"

	// 1. Owner creates project v1 in cloud and shares with collaborator
	initialRes, err := engine.Push(owner, PushRequest{
		ProjectID:     0,
		Title:         "Local-First Ballad",
		BPM:           95,
		TimeSignature: "4/4",
		Tracks: []models.Track{
			{Name: "Piano", DataContent: ":4 c.4 e.4 g.4"},
		},
	})
	if err != nil {
		t.Fatalf("failed initial push: %v", err)
	}
	projectID := initialRes.ProjectID

	if err := engine.Share(projectID, owner, collab); err != nil {
		t.Fatalf("failed to share project: %v", err)
	}

	// 2. Collaborator edits offline based on v1 (without holding a cloud checkout lock)
	// and later reconnects when project is still at v1 and unlocked -> Push succeeds to v2
	offlineSyncRes, err := engine.Push(collab, PushRequest{
		ProjectID:     projectID,
		Title:         "Local-First Ballad (Offline Vocal Idea)",
		BPM:           95,
		TimeSignature: "4/4",
		BaseVersion:   1,
	})
	if err != nil {
		t.Fatalf("expected offline edits based on latest v1 to reconcile cleanly, got %v", err)
	}
	if offlineSyncRes.NewVersion != 2 {
		t.Errorf("expected version 2 after offline reconciliation, got %d", offlineSyncRes.NewVersion)
	}

	// 3. Owner was also editing offline based on stale v1. When Owner reconnects and attempts to push
	// with BaseVersion = 1, server detects version mismatch (cloud is at v2) and prevents overwrite!
	_, err = engine.Push(owner, PushRequest{
		ProjectID:   projectID,
		Title:       "Local-First Ballad (Conflicting Offline Edit)",
		BPM:         100,
		BaseVersion: 1,
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("expected ErrVersionConflict when pushing stale offline base version 1 against cloud v2, got %v", err)
	}

	// 4. Diff check reports the divergence so client can inspect changes before overwriting
	diff, err := engine.GetDiff(projectID, 1)
	if err != nil {
		t.Fatalf("failed to compute diff for offline reconciliation: %v", err)
	}
	if !diff.HasChanges || diff.ClientVersion != 1 || diff.CloudVersion != 2 {
		t.Errorf("expected diff to detect cloud v2 vs client v1, got %+v", diff)
	}
}

func TestSyncEngine_ContentHashAndDiff(t *testing.T) {
	locker := collaboration.NewLocker()
	engine := NewEngine(locker)

	owner := "user_hash_tester"

	// 1. Initial push generates deterministic ContentHash
	res1, err := engine.Push(owner, PushRequest{
		ProjectID:     0,
		Title:         "Checksum Song",
		BPM:           120,
		TimeSignature: "4/4",
		KeySignature:  "C",
		Tracks: []models.Track{
			{InstrumentID: 1, Name: "Guitar", DataContent: ":4 1.1 2.1"},
		},
	})
	if err != nil {
		t.Fatalf("failed initial push: %v", err)
	}
	if len(res1.ContentHash) != 16 {
		t.Fatalf("expected 16-char ContentHash in PushResponse, got %q", res1.ContentHash)
	}

	pulled1, err := engine.Pull(res1.ProjectID)
	if err != nil {
		t.Fatalf("failed to pull project: %v", err)
	}
	if pulled1.ContentHash != res1.ContentHash {
		t.Errorf("expected pulled project ContentHash %q to match PushResponse %q", pulled1.ContentHash, res1.ContentHash)
	}

	// 2. GetDiff with matching version and matching ContentHash -> HasChanges = false
	diffSame, err := engine.GetDiff(res1.ProjectID, 1, res1.ContentHash)
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}
	if diffSame.HasChanges {
		t.Errorf("expected HasChanges=false when version and hash match, got true")
	}
	if diffSame.CloudHash != res1.ContentHash {
		t.Errorf("expected CloudHash=%q, got %q", res1.ContentHash, diffSame.CloudHash)
	}

	// 3. GetDiff with same version but different local ContentHash (unpushed local changes) -> HasChanges = true
	diffLocalChanged, err := engine.GetDiff(res1.ProjectID, 1, "deadbeefcafebabe")
	if err != nil {
		t.Fatalf("GetDiff failed: %v", err)
	}
	if !diffLocalChanged.HasChanges {
		t.Errorf("expected HasChanges=true when client hash differs from cloud hash")
	}

	// 4. Push modified content -> ContentHash changes on server
	res2, err := engine.Push(owner, PushRequest{
		ProjectID:     res1.ProjectID,
		Title:         "Checksum Song",
		BPM:           120,
		TimeSignature: "4/4",
		KeySignature:  "C",
		BaseVersion:   1,
		Tracks: []models.Track{
			{InstrumentID: 1, Name: "Guitar", DataContent: ":4 1.1 2.1 5.1"},
		},
	})
	if err != nil {
		t.Fatalf("failed v2 push: %v", err)
	}
	if res2.ContentHash == res1.ContentHash {
		t.Errorf("expected ContentHash to change after modifying track content")
	}
}
