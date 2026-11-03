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

package collaboration

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrProjectLocked       = errors.New("project is currently checked out by another user")
	ErrLockNotHeld         = errors.New("user does not hold lock for this project")
	ErrLockExpired         = errors.New("project checkout lock has expired")
	DefaultLockTTL         = 30 * time.Minute
)

// ProjectLock represents an exclusive checkout lease on a song project (Issues #16, #22).
type ProjectLock struct {
	ProjectID  int       `json:"project_id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	AcquiredAt time.Time `json:"acquired_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// IsExpired checks if the checkout lock has surpassed its time-to-live.
func (l *ProjectLock) IsExpired() bool {
	return time.Now().UTC().After(l.ExpiresAt)
}

// Locker manages exclusive project checkouts and lock release for collaborative editing.
type Locker struct {
	mu    sync.RWMutex
	locks map[int]*ProjectLock // keyed by project/song ID
}

// NewLocker instantiates a thread-safe Locker.
func NewLocker() *Locker {
	return &Locker{
		locks: make(map[int]*ProjectLock),
	}
}

// AcquireLock attempts to obtain an exclusive checkout lease on a project (Issue #16).
// If a lock is held by another user and has not expired, ErrProjectLocked is returned.
// If the same user already holds the lock, the expiration time is renewed.
func (l *Locker) AcquireLock(projectID int, userID, username string, ttl time.Duration) (*ProjectLock, error) {
	if ttl <= 0 {
		ttl = DefaultLockTTL
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()
	current, exists := l.locks[projectID]

	if exists && !current.IsExpired() {
		if current.UserID != userID {
			return nil, fmt.Errorf("%w: user '%s' holds the lock until %s", ErrProjectLocked, current.Username, current.ExpiresAt.Format(time.RFC3339))
		}
		// Same user: renew lease
		current.ExpiresAt = now.Add(ttl)
		return current, nil
	}

	// New lock or expired lock
	newLock := &ProjectLock{
		ProjectID:  projectID,
		UserID:     userID,
		Username:   username,
		AcquiredAt: now,
		ExpiresAt:  now.Add(ttl),
	}

	l.locks[projectID] = newLock
	return newLock, nil
}

// ReleaseLock releases an active project checkout lease held by a user (Issue #22).
func (l *Locker) ReleaseLock(projectID int, userID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	current, exists := l.locks[projectID]
	if !exists {
		return nil // Already free
	}

	if current.UserID != userID && !current.IsExpired() {
		return ErrLockNotHeld
	}

	delete(l.locks, projectID)
	return nil
}

// ForceReleaseLock forcefully removes a lock regardless of ownership (e.g. admin override).
func (l *Locker) ForceReleaseLock(projectID int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.locks, projectID)
}

// GetLock returns the current lock status for a project.
func (l *Locker) GetLock(projectID int) (*ProjectLock, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	current, exists := l.locks[projectID]
	if !exists || current.IsExpired() {
		return nil, false
	}

	return current, true
}
