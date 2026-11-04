package collaboration

import (
	"errors"
	"testing"
	"time"
)

func TestProjectLocker(t *testing.T) {
	locker := NewLocker()

	projectID := 42
	userAlice := "user_alice"
	userBob := "user_bob"

	// 1. Acquire lock by Alice
	lock, err := locker.AcquireLock(projectID, userAlice, "Alice", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to acquire lock: %v", err)
	}
	if lock.ProjectID != projectID || lock.UserID != userAlice {
		t.Errorf("unexpected lock details: %+v", lock)
	}

	// 2. Bob tries to acquire locked project -> returns ErrProjectLocked
	_, err = locker.AcquireLock(projectID, userBob, "Bob", time.Minute)
	if !errors.Is(err, ErrProjectLocked) {
		t.Fatalf("expected ErrProjectLocked for Bob, got %v", err)
	}

	// 3. Alice renews her own lock -> succeeds
	renewed, err := locker.AcquireLock(projectID, userAlice, "Alice", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("expected Alice to renew her lock: %v", err)
	}
	if renewed.ExpiresAt.Before(lock.ExpiresAt) {
		t.Errorf("expected renewed expiration to be later")
	}

	// 4. Bob tries to release Alice's lock -> returns ErrLockNotHeld
	err = locker.ReleaseLock(projectID, userBob)
	if !errors.Is(err, ErrLockNotHeld) {
		t.Fatalf("expected ErrLockNotHeld, got %v", err)
	}

	// 5. Alice releases lock -> succeeds
	if err := locker.ReleaseLock(projectID, userAlice); err != nil {
		t.Fatalf("failed to release lock: %v", err)
	}

	// 6. Project is now free, Bob can acquire it
	bobLock, err := locker.AcquireLock(projectID, userBob, "Bob", 50*time.Millisecond)
	if err != nil {
		t.Fatalf("failed for Bob to acquire free lock: %v", err)
	}
	if bobLock.UserID != userBob {
		t.Errorf("expected Bob to hold lock")
	}

	// 7. Wait for Bob's lock to expire
	time.Sleep(70 * time.Millisecond)

	// Alice acquires after Bob's lock expired automatically
	aliceNewLock, err := locker.AcquireLock(projectID, userAlice, "Alice", time.Minute)
	if err != nil {
		t.Fatalf("expected Alice to acquire expired lock: %v", err)
	}
	if aliceNewLock.UserID != userAlice {
		t.Errorf("expected Alice to hold lock after expiration")
	}
}
