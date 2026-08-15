package store

import (
	"database/sql"
	"fmt"
    "time"
	"math"
	"math/rand"
	"github.com/krishnabhardwaj25/flowithgo/internal/models"
)

type JobStore struct {
	db *sql.DB
}

func NewJobStore(db *sql.DB) *JobStore {
	return &JobStore{db: db}
}

func (s *JobStore) GetQueueStats() (models.QueueStats, error) {
	query := `
		SELECT status, COUNT(*)
		FROM jobs
		GROUP BY status
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return models.QueueStats{}, fmt.Errorf("failed to get queue stats: %w", err)
	}
	defer rows.Close()

	var stats models.QueueStats

	for rows.Next() {
		var status string
		var count int

		if err := rows.Scan(&status, &count); err != nil {
			return models.QueueStats{}, fmt.Errorf("failed to scan queue stats: %w", err)
		}

		switch status {
		case "queued":
			stats.Queued = count

		case "running":
			stats.Running = count

		case "done":
			stats.Done = count

		case "failed":
			stats.Failed = count
		}
	}

	if err := rows.Err(); err != nil {
		return models.QueueStats{}, fmt.Errorf("failed while reading queue stats: %w", err)
	}

	return stats, nil
}

func (s *JobStore) RequeueJob(id string) error {
	query := `
		UPDATE jobs
		SET status = 'queued',
		    attempts = 0,
		    last_error = NULL,
		    run_after = NOW(),
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to requeue job: %w", err)
	}

	return nil
}

func(s *JobStore) FailJob(id string, lastError string) error{
	var attempts, maxAttempts int

	err := s.db.QueryRow(`
		SELECT attempts, max_attempts
		FROM jobs
		WHERE id = $1
	`, id).Scan(&attempts, &maxAttempts)

	if err != nil {
		return fmt.Errorf("failed to get job attempts: %w", err)
	}

	// No more retries.
	if attempts >= maxAttempts {
		_, err := s.db.Exec(`
			UPDATE jobs
			SET status = 'failed',
			    last_error = $1,
			    updated_at = NOW()
			WHERE id = $2
		`, lastError, id)

		if err != nil {
			return fmt.Errorf("failed to mark job as failed: %w", err)
		}

		return nil
	}

	// Exponential backoff.
	backoff := time.Duration(
		math.Pow(2, float64(attempts-1)),
	) * 10 * time.Second

	// Random jitter: 0–4 seconds.
	jitter := time.Duration(rand.Intn(5)) * time.Second

	runAfter := time.Now().Add(backoff + jitter)

	_, err = s.db.Exec(`
		UPDATE jobs
		SET status = 'queued',
		    last_error = $1,
		    run_after = $2,
		    updated_at = NOW()
		WHERE id = $3
	`, lastError, runAfter, id)

	if err != nil {
		return fmt.Errorf("failed to reschedule job: %w", err)
	}

	return nil
}

func (s *JobStore) GetDeadJobs() ([]models.Job, error) {
	query := `
		SELECT id, type, payload, status, attempts, max_attempts, run_after, created_at, updated_at, last_error
		FROM jobs
		WHERE status = 'failed' AND attempts >= max_attempts
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get dead jobs: %w", err)
	}
	defer rows.Close()

	var jobs []models.Job
	for rows.Next() {
		var job models.Job
		err := rows.Scan(
			&job.ID,
			&job.Type,
			&job.Payload,
			&job.Status,
			&job.Attempts,
			&job.MaxAttempts,
			&job.RunAfter,
			&job.CreatedAt,
			&job.UpdatedAt,
			&job.LastError,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}
		jobs = append(jobs, job)
	}

	return jobs, nil
}


func (s *JobStore) UpdateJobStatus(id string, status string) error {
	query := `
		UPDATE jobs
		SET status = $1,
		    updated_at = NOW()
		WHERE id = $2
	`

	_, err := s.db.Exec(query, status, id)
	if err != nil {
		return fmt.Errorf("failed to update job status: %w", err)
	}

	return nil
}

func (s *JobStore) ClaimNextJob() (models.Job, error) {
	tx , err := s.db.Begin();
	if err!= nil{
		return models.Job{}, fmt.Errorf("failed to begin transaction : %w",err)
	}
	defer tx.Rollback()

	query := `UPDATE jobs
		SET status = 'running',
		    attempts = attempts + 1,
		    updated_at = NOW()
		WHERE id = (
			SELECT id
			FROM jobs
			WHERE status = 'queued'
			  AND run_after <= NOW()
			  AND attempts < max_attempts
			ORDER BY created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING id, type, payload, status, attempts, max_attempts,
		          run_after, created_at, updated_at,last_error`

	var job models.Job
	err = tx.QueryRow(query).Scan(
		&job.ID,
		&job.Type,
		&job.Payload,
		&job.Status,
		&job.Attempts,
		&job.MaxAttempts,
		&job.RunAfter,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.LastError,
	)

	if err == sql.ErrNoRows {
		return models.Job{}, fmt.Errorf("no jobs available")
	}

	if err != nil {
		return models.Job{}, fmt.Errorf("failed to claim job: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return models.Job{}, fmt.Errorf("failed to commit job claim: %w", err)
	}

	return job, nil
}


func (s *JobStore) InsertJob(job models.Job) (models.Job, error) {
	query := `
		INSERT INTO jobs (type, payload, status, max_attempts, run_after)
		VALUES ($1, $2, 'queued', $3, NOW())
		RETURNING id, type, payload, status, attempts, max_attempts, run_after, created_at, updated_at,last_error
	`

	var inserted models.Job
	err := s.db.QueryRow(query, job.Type, job.Payload, job.MaxAttempts).Scan(
		&inserted.ID,
		&inserted.Type,
		&inserted.Payload,
		&inserted.Status,
		&inserted.Attempts,
		&inserted.MaxAttempts,
		&inserted.RunAfter,
		&inserted.CreatedAt,
		&inserted.UpdatedAt,
		&inserted.LastError,
	)
	if err != nil {
		return models.Job{}, fmt.Errorf("failed to insert job: %w", err)
	}

	return inserted, nil
}

func (s *JobStore) GetJobByID(id string) (models.Job, error) {
	query := `
		SELECT id, type, payload, status, attempts, max_attempts, run_after, created_at, updated_at,last_error
		FROM jobs
		WHERE id = $1
	`

	var job models.Job
	err := s.db.QueryRow(query, id).Scan(
		&job.ID,
		&job.Type,
		&job.Payload,
		&job.Status,
		&job.Attempts,
		&job.MaxAttempts,
		&job.RunAfter,
		&job.CreatedAt,
		&job.UpdatedAt,
		&job.LastError,
	)
	if err == sql.ErrNoRows {
		return models.Job{}, fmt.Errorf("job not found: %s", id)
	}
	if err != nil {
		return models.Job{}, fmt.Errorf("failed to get job: %w", err)
	}

	return job, nil
}