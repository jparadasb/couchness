package jobs

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestStartRunsTaskAndCollectsLogs(t *testing.T) {
	runner := NewRunner(5)

	job, err := runner.Start("scan", func(log Logger) error {
		log("line one")
		log("line two")
		return nil
	})
	if err != nil {
		t.Fatalf("Start returned an error: %v", err)
	}
	if job.ID() == "" {
		t.Fatal("expected the job to have an id")
	}

	if !runner.Wait(job, 2*time.Second) {
		t.Fatal("job did not finish before the timeout")
	}

	snap := job.Snapshot()
	if !snap.Done {
		t.Error("expected snapshot Done to be true")
	}
	if snap.Error != "" {
		t.Errorf("expected no error, got %q", snap.Error)
	}
	if snap.Name != "scan" {
		t.Errorf("expected name %q, got %q", "scan", snap.Name)
	}
	if want := []string{"line one", "line two"}; !reflect.DeepEqual(snap.Lines, want) {
		t.Errorf("expected lines %v, got %v", want, snap.Lines)
	}
	if snap.Status() != "done" {
		t.Errorf("expected status done, got %q", snap.Status())
	}
	if snap.FinishedAt.Before(snap.StartedAt) {
		t.Error("expected FinishedAt to be after StartedAt")
	}
}

func TestStartWhileBusyReturnsErrBusy(t *testing.T) {
	runner := NewRunner(5)
	release := make(chan struct{})
	started := make(chan struct{})

	first, err := runner.Start("blocking", func(log Logger) error {
		close(started)
		<-release
		return nil
	})
	if err != nil {
		t.Fatalf("first Start returned an error: %v", err)
	}
	<-started

	_, err = runner.Start("second", func(log Logger) error { return nil })
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("expected ErrBusy, got %v", err)
	}
	if runner.Running() != first {
		t.Error("expected the first job to still be the running one")
	}

	close(release)
	if !runner.Wait(first, 2*time.Second) {
		t.Fatal("first job did not finish before the timeout")
	}
}

func TestSnapshotRecordsFailure(t *testing.T) {
	runner := NewRunner(5)

	job, err := runner.Start("failing", func(log Logger) error {
		log("about to fail")
		return errors.New("boom")
	})
	if err != nil {
		t.Fatalf("Start returned an error: %v", err)
	}

	if !runner.Wait(job, 2*time.Second) {
		t.Fatal("job did not finish before the timeout")
	}

	snap := job.Snapshot()
	if !snap.Done {
		t.Error("expected snapshot Done to be true")
	}
	if snap.Error != "boom" {
		t.Errorf("expected error %q, got %q", "boom", snap.Error)
	}
	if snap.Status() != "failed" {
		t.Errorf("expected status failed, got %q", snap.Status())
	}
	if len(snap.Lines) != 1 || snap.Lines[0] != "about to fail" {
		t.Errorf("expected the logged line to be kept, got %v", snap.Lines)
	}
}

func TestListReturnsNewestFirst(t *testing.T) {
	runner := NewRunner(5)

	var last *Job
	for _, name := range []string{"one", "two", "three"} {
		job, err := runner.Start(name, func(log Logger) error { return nil })
		if err != nil {
			t.Fatalf("Start returned an error: %v", err)
		}
		if !runner.Wait(job, 2*time.Second) {
			t.Fatalf("job %s did not finish", name)
		}
		last = job
	}

	snaps := runner.List()
	if len(snaps) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(snaps))
	}
	for i, want := range []string{"3", "2", "1"} {
		if snaps[i].ID != want {
			t.Errorf("expected position %d to be job %s, got %s", i, want, snaps[i].ID)
		}
	}
	if _, found := runner.Get(last.ID()); !found {
		t.Errorf("expected to find job %s with Get", last.ID())
	}
}

func TestTrimKeepsOnlyKeepJobs(t *testing.T) {
	runner := NewRunner(2)

	for _, name := range []string{"one", "two", "three"} {
		job, err := runner.Start(name, func(log Logger) error { return nil })
		if err != nil {
			t.Fatalf("Start returned an error: %v", err)
		}
		if !runner.Wait(job, 2*time.Second) {
			t.Fatalf("job %s did not finish", name)
		}
	}

	snaps := runner.List()
	if len(snaps) != 2 {
		t.Fatalf("expected the trim to keep 2 jobs, got %d", len(snaps))
	}
	if snaps[0].ID != "3" || snaps[1].ID != "2" {
		t.Errorf("expected the newest jobs 3 and 2 to be kept, got %s and %s", snaps[0].ID, snaps[1].ID)
	}
	if _, found := runner.Get("1"); found {
		t.Error("expected the oldest job to have been trimmed")
	}
}
