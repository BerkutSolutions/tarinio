package services

import "waf/control-plane/internal/jobs"

type JobStore interface {
	Create(job jobs.Job) (jobs.Job, error)
	MarkRunning(jobID string) (jobs.Job, error)
	MarkSucceeded(jobID string, result string) (jobs.Job, error)
	MarkFailed(jobID string, result string) (jobs.Job, error)
	Get(jobID string) (jobs.Job, bool, error)
	List() ([]jobs.Job, error)
}

type JobService struct {
	store JobStore
}

func NewJobService(store JobStore) *JobService {
	return &JobService{store: store}
}

func (s *JobService) Get(jobID string) (jobs.Job, bool, error) {
	return s.store.Get(jobID)
}

func (s *JobService) List() ([]jobs.Job, error) {
	return s.store.List()
}
