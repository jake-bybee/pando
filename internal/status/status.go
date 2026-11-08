package status

import (
	"sync"
	"time"
)

type State string

const (
	StateStarting        State = "starting"
	StateWaitingForPeers State = "waiting_for_peers"
	StateScanning        State = "scanning"
	StateDownloading     State = "downloading"
	StateWaitingForDisk  State = "waiting_for_disk"
	StateIdle            State = "idle"
	StateStopped         State = "stopped"
)

type Error struct {
	Message string
	At      time.Time
	Source  string
}

type Snapshot struct {
	State                  State
	StateSince             time.Time
	LastScanStarted        *time.Time
	LastScanCompleted      *time.Time
	LastSuccessfulDownload *time.Time
	LastError              *Error
	StartedAt              time.Time
	UptimeSeconds          int64
}

type Tracker struct {
	mu                     sync.RWMutex
	state                  State
	stateSince             time.Time
	previousState          State
	activeChunks           int
	lastScanStarted        *time.Time
	lastScanCompleted      *time.Time
	lastSuccessfulDownload *time.Time
	lastError              *Error
	startedAt              time.Time
}

func NewTracker() *Tracker {
	now := time.Now().UTC()
	return &Tracker{state: StateStarting, stateSince: now, startedAt: now}
}

func (t *Tracker) SetState(state State) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.state == state {
		return
	}
	now := time.Now().UTC()
	if t.activeChunks > 0 && t.state == StateDownloading {
		t.previousState = state
		return
	}
	t.state = state
	t.stateSince = now
}

func (t *Tracker) MarkScanStarted() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().UTC()
	t.lastScanStarted = &now
}

func (t *Tracker) MarkScanCompleted() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().UTC()
	t.lastScanCompleted = &now
}

func (t *Tracker) MarkDownloadSuccess() {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().UTC()
	t.lastSuccessfulDownload = &now
}

func (t *Tracker) RecordError(source string, err error) {
	if err == nil {
		return
	}
	message := err.Error()
	messageRunes := []rune(message)
	if len(messageRunes) > 500 {
		message = string(messageRunes[:500])
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	now := time.Now().UTC()
	t.lastError = &Error{Message: message, At: now, Source: source}
}

func (t *Tracker) ChunkStarted() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.activeChunks++
	if t.activeChunks == 1 {
		t.previousState = t.state
		t.state = StateDownloading
		t.stateSince = time.Now().UTC()
	}
}

func (t *Tracker) ChunkFinished() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.activeChunks == 0 {
		return
	}
	t.activeChunks--
	if t.activeChunks == 0 {
		state := t.previousState
		if state == "" || state == StateDownloading {
			state = StateScanning
		}
		t.state = state
		t.stateSince = time.Now().UTC()
	}
}

func (t *Tracker) Snapshot() Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()
	snapshot := Snapshot{
		State:                  t.state,
		StateSince:             t.stateSince,
		LastScanStarted:        cloneTime(t.lastScanStarted),
		LastScanCompleted:      cloneTime(t.lastScanCompleted),
		LastSuccessfulDownload: cloneTime(t.lastSuccessfulDownload),
		LastError:              cloneError(t.lastError),
		StartedAt:              t.startedAt,
		UptimeSeconds:          int64(time.Since(t.startedAt).Seconds()),
	}
	return snapshot
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneError(value *Error) *Error {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
