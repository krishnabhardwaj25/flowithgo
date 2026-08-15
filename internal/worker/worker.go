package worker

import (
	"context"
	"sync"
	"time"

	"github.com/krishnabhardwaj25/flowithgo/internal/events"
	"github.com/krishnabhardwaj25/flowithgo/internal/handlers"
	"github.com/krishnabhardwaj25/flowithgo/internal/logger"
	"github.com/krishnabhardwaj25/flowithgo/internal/store"
)

type Broadcaster interface {
	Publish(event events.Event)
}

type Worker struct {
	id          int
	jobStore    *store.JobStore
	handlers    map[string]handlers.HandlerFunc
	pool        *Pool
	broadcaster Broadcaster
}

type Pool struct {
	workers     []*Worker
	jobStore    *store.JobStore
	size        int
	wg          sync.WaitGroup
	broadcaster Broadcaster
}

func NewPool(size int, jobStore *store.JobStore, broadcaster Broadcaster) *Pool {
	return &Pool{
		size:        size,
		jobStore:    jobStore,
		broadcaster: broadcaster,
	}
}

func NewWorker(id int, jobStore *store.JobStore, pool *Pool) *Worker {
	w := &Worker{
		id:          id,
		jobStore:    jobStore,
		handlers:    make(map[string]handlers.HandlerFunc),
		pool:        pool,
		broadcaster: pool.broadcaster,
	}
	w.handlers["send_email"] = handlers.SendEmail
	w.handlers["process_file"] = handlers.ProcessFile
	return w
}

func (w *Worker) Start(ctx context.Context) {
	logger.L.Info("worker started", "worker_id", w.id)
	defer w.pool.wg.Done()

	for {
		select {
		case <-ctx.Done():
			logger.L.Info("worker shutting down", "worker_id", w.id)
			return
		default:
		}

		job, err := w.jobStore.ClaimNextJob()
		if err != nil {
			if err.Error() != "no jobs available" {
				logger.L.Error("error claiming job", "worker_id", w.id, "error", err.Error())
			}
			time.Sleep(2 * time.Second)
			continue
		}

		logger.L.Info("claimed job", "worker_id", w.id, "job_id", job.ID, "type", job.Type)
		w.broadcaster.Publish(events.Event{
			Type: "job_claimed",
			Payload: map[string]any{
				"job_id":    job.ID,
				"type":      job.Type,
				"worker_id": w.id,
			},
		})

		handler, exists := w.handlers[job.Type]
		if !exists {
			logger.L.Error("no handler for job type", "worker_id", w.id, "type", job.Type)
			w.jobStore.FailJob(job.ID, "no handler registered for job type: "+job.Type)
			w.broadcaster.Publish(events.Event{
				Type: "job_failed",
				Payload: map[string]any{
					"job_id": job.ID,
					"type":   job.Type,
					"error":  "no handler registered",
				},
			})
			continue
		}

		err = handler(job.Payload)
		if err != nil {
			logger.L.Error("job failed", "worker_id", w.id, "job_id", job.ID, "error", err.Error())
			w.jobStore.FailJob(job.ID, err.Error())
			w.broadcaster.Publish(events.Event{
				Type: "job_failed",
				Payload: map[string]any{
					"job_id": job.ID,
					"type":   job.Type,
					"error":  err.Error(),
				},
			})
			continue
		}

		logger.L.Info("job completed", "worker_id", w.id, "job_id", job.ID)
		w.jobStore.UpdateJobStatus(job.ID, "done")
		w.broadcaster.Publish(events.Event{
			Type: "job_completed",
			Payload: map[string]any{
				"job_id": job.ID,
				"type":   job.Type,
			},
		})
	}
}

func (p *Pool) Start(ctx context.Context) {
	logger.L.Info("starting worker pool", "size", p.size)

	for i := 1; i <= p.size; i++ {
		w := NewWorker(i, p.jobStore, p)
		p.workers = append(p.workers, w)
		p.wg.Add(1)
		go w.Start(ctx)
	}
}

func (p *Pool) Wait() {
	p.wg.Wait()
}