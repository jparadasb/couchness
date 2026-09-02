// Package jobs runs background tasks (scan, downloads) started from the web UI.
package jobs

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
)

// Logger receives progress lines from a running job.
type Logger func(line string)

// Task is the work executed by a job.
type Task func(log Logger) error

// Job is a background task started from the web UI.
type Job struct {
	id         string
	name       string
	startedAt  time.Time
	finishedAt time.Time
	done       bool
	err        string
	lines      []string
	mu         sync.Mutex
}

// ID returns the job identifier.
func (j *Job) ID() string { return j.id }

// Log appends a progress line.
func (j *Job) Log(line string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.lines = append(j.lines, line)
}

// Snapshot is the immutable view of a job rendered by templates.
type Snapshot struct {
	ID         string
	Name       string
	StartedAt  time.Time
	FinishedAt time.Time
	Done       bool
	Error      string
	Lines      []string
}

// Snapshot captures the job state for rendering.
func (j *Job) Snapshot() Snapshot {
	j.mu.Lock()
	defer j.mu.Unlock()
	return Snapshot{
		ID:         j.id,
		Name:       j.name,
		StartedAt:  j.startedAt,
		FinishedAt: j.finishedAt,
		Done:       j.done,
		Error:      j.err,
		Lines:      append([]string(nil), j.lines...),
	}
}

// Duration reports how long the job ran.
func (s Snapshot) Duration() time.Duration {
	end := s.FinishedAt
	if !s.Done {
		end = time.Now()
	}
	return end.Sub(s.StartedAt).Round(time.Second)
}

// Status is a short human readable state.
func (s Snapshot) Status() string {
	switch {
	case !s.Done:
		return "running"
	case s.Error != "":
		return "failed"
	default:
		return "done"
	}
}

// ErrBusy is returned when another job is still running.
var ErrBusy = errors.New("another job is still running, wait for it to finish")

// Runner executes one job at a time and remembers the last ones.
type Runner struct {
	mu      sync.Mutex
	jobs    map[string]*Job
	running *Job
	next    int
	keep    int
}

// NewRunner creates a runner keeping at most keep finished jobs.
func NewRunner(keep int) *Runner {
	if keep <= 0 {
		keep = 20
	}
	return &Runner{jobs: map[string]*Job{}, keep: keep}
}

// Start begins task in the background. Only one job runs at a time.
func (r *Runner) Start(name string, task Task) (*Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.running != nil {
		return nil, ErrBusy
	}
	r.next++
	job := &Job{id: strconv.Itoa(r.next), name: name, startedAt: time.Now()}
	r.jobs[job.id] = job
	r.running = job
	r.trim()

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				r.finish(job, fmt.Errorf("panic: %v", recovered))
			}
		}()
		r.finish(job, task(job.Log))
	}()

	return job, nil
}

func (r *Runner) finish(job *Job, err error) {
	job.mu.Lock()
	if job.done {
		job.mu.Unlock()
		return
	}
	job.done = true
	job.finishedAt = time.Now()
	if err != nil {
		job.err = err.Error()
	}
	job.mu.Unlock()

	r.mu.Lock()
	if r.running == job {
		r.running = nil
	}
	r.mu.Unlock()
}

// Get returns a job by ID.
func (r *Runner) Get(id string) (*Job, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	job, found := r.jobs[id]
	return job, found
}

// Running returns the job currently in progress, if any.
func (r *Runner) Running() *Job {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running
}

// Wait blocks until the job finishes or the timeout expires.
func (r *Runner) Wait(job *Job, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if job.Snapshot().Done {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return job.Snapshot().Done
}

// List returns every remembered job, newest first.
func (r *Runner) List() []Snapshot {
	r.mu.Lock()
	all := make([]*Job, 0, len(r.jobs))
	for _, job := range r.jobs {
		all = append(all, job)
	}
	r.mu.Unlock()

	snapshots := make([]Snapshot, 0, len(all))
	for _, job := range all {
		snapshots = append(snapshots, job.Snapshot())
	}
	sort.Slice(snapshots, func(i, j int) bool {
		a, _ := strconv.Atoi(snapshots[i].ID)
		b, _ := strconv.Atoi(snapshots[j].ID)
		return a > b
	})
	return snapshots
}

// trim drops the oldest finished jobs beyond the keep limit. Caller holds r.mu.
func (r *Runner) trim() {
	if len(r.jobs) <= r.keep {
		return
	}
	ids := make([]string, 0, len(r.jobs))
	for id, job := range r.jobs {
		if job != r.running {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool {
		a, _ := strconv.Atoi(ids[i])
		b, _ := strconv.Atoi(ids[j])
		return a < b
	})
	for _, id := range ids {
		if len(r.jobs) <= r.keep {
			break
		}
		delete(r.jobs, id)
	}
}
