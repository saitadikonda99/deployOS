// Package scheduler defines how DeployOS will schedule and track units of
// work (deployments, backups, health sweeps) across a fleet. It contains
// an interface and supporting types only - no scheduling logic exists yet.
package scheduler

import "context"

// JobID identifies a scheduled unit of work.
type JobID string

// JobStatus is the lifecycle state of a Job.
type JobStatus string

const (
	// JobStatusPending means the job has been accepted but not started.
	JobStatusPending JobStatus = "pending"
	// JobStatusRunning means the job is currently executing.
	JobStatusRunning JobStatus = "running"
	// JobStatusSucceeded means the job completed without error.
	JobStatusSucceeded JobStatus = "succeeded"
	// JobStatusFailed means the job completed with an error.
	JobStatusFailed JobStatus = "failed"
)

// Job is a single unit of work tracked by a Scheduler.
type Job struct {
	ID     JobID
	Name   string
	Status JobStatus
}

// Scheduler represents the future scheduling operations DeployOS will
// expose for coordinating work across agents. Implementations are
// expected to be added alongside the features that need them (deploys,
// backups, etc.), not here.
type Scheduler interface {
	// Schedule enqueues a new job and returns once it has been accepted.
	Schedule(ctx context.Context, job Job) error
	// Status returns the current state of a previously scheduled job.
	Status(ctx context.Context, id JobID) (Job, error)
	// Cancel requests cancellation of a pending or running job.
	Cancel(ctx context.Context, id JobID) error
}
