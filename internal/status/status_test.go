package status

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestTrackerStateChanges(t *testing.T) {
	tracker := NewTracker()
	tracker.SetState(StateScanning)
	snapshot := tracker.Snapshot()
	if snapshot.State != StateScanning {
		t.Fatalf("state = %q, want %q", snapshot.State, StateScanning)
	}
	if snapshot.StateSince.IsZero() {
		t.Fatal("stateSince is zero")
	}
}

func TestTrackerActiveChunks(t *testing.T) {
	tracker := NewTracker()
	tracker.SetState(StateScanning)
	const workers = 20
	var started sync.WaitGroup
	started.Add(workers)
	var finished sync.WaitGroup
	finished.Add(workers)
	release := make(chan struct{})
	for range workers {
		go func() {
			defer finished.Done()
			tracker.ChunkStarted()
			started.Done()
			<-release
			tracker.ChunkFinished()
		}()
	}
	started.Wait()
	close(release)
	finished.Wait()
	if snapshot := tracker.Snapshot(); snapshot.State != StateScanning {
		t.Fatalf("state = %q, want %q", snapshot.State, StateScanning)
	}
}

func TestTrackerErrorTruncation(t *testing.T) {
	tracker := NewTracker()
	tracker.RecordError("download", errors.New(strings.Repeat("x", 600)))
	snapshot := tracker.Snapshot()
	if snapshot.LastError == nil || len(snapshot.LastError.Message) != 500 {
		t.Fatalf("last error = %+v, want 500-character message", snapshot.LastError)
	}
	if snapshot.LastError.Source != "download" {
		t.Fatalf("source = %q, want download", snapshot.LastError.Source)
	}
}
