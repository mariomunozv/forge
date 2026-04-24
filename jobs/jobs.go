package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// Job is the interface all background jobs must implement.
type Job interface {
	Perform(ctx context.Context) error
}

var (
	mu       sync.RWMutex
	registry = map[string]func() Job{}
)

// Register associates a job name with a factory that creates it.
// Call this in an init() function in each job file.
func Register(name string, factory func() Job) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = factory
}

// Queryer is satisfied by *sql.DB and *sql.Tx.
type Queryer interface {
	QueryRow(query string, args ...any) *sql.Row
}

// Enqueue inserts a job into background_jobs and returns its id.
func Enqueue(q Queryer, jobType string, payload any) (int64, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}
	var id int64
	err = q.QueryRow(
		`INSERT INTO background_jobs (type, payload, status, run_at, created_at, updated_at)
		 VALUES ($1, $2, 'pending', NOW(), NOW(), NOW()) RETURNING id`,
		jobType, string(b),
	).Scan(&id)
	return id, err
}

// Work starts workers workers that poll the DB and run jobs. Blocks until ctx is cancelled.
func Work(ctx context.Context, conn *sql.DB, workers int) error {
	if err := ensureTable(conn); err != nil {
		return err
	}
	log.Printf("jobs: starting %d workers", workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			runWorker(ctx, conn)
		}()
	}
	wg.Wait()
	return nil
}

const maxAttempts = 3

func runWorker(ctx context.Context, conn *sql.DB) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		processed, err := processOne(ctx, conn)
		if err != nil {
			log.Printf("jobs: worker error: %v", err)
			sleep(ctx, 2*time.Second)
			continue
		}
		if !processed {
			sleep(ctx, time.Second)
		}
	}
}

type jobRow struct {
	id      int64
	jobType string
	payload string
	attempt int
}

func processOne(ctx context.Context, conn *sql.DB) (bool, error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var row jobRow
	err = tx.QueryRowContext(ctx, `
		SELECT id, type, payload, attempts
		FROM background_jobs
		WHERE status = 'pending' AND run_at <= NOW()
		ORDER BY run_at
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`).Scan(&row.id, &row.jobType, &row.payload, &row.attempt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// Mark running
	if _, err = tx.ExecContext(ctx,
		`UPDATE background_jobs SET status='running', updated_at=NOW() WHERE id=$1`, row.id,
	); err != nil {
		return false, err
	}
	if err = tx.Commit(); err != nil {
		return false, err
	}

	jobErr := runJob(ctx, row)

	// Record result in a new transaction
	tx2, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return true, err
	}
	defer tx2.Rollback()

	if jobErr == nil {
		_, err = tx2.ExecContext(ctx,
			`UPDATE background_jobs SET status='done', updated_at=NOW() WHERE id=$1`, row.id)
	} else {
		attempt := row.attempt + 1
		if attempt >= maxAttempts {
			_, err = tx2.ExecContext(ctx,
				`UPDATE background_jobs SET status='failed', attempts=$2, error=$3, updated_at=NOW() WHERE id=$1`,
				row.id, attempt, jobErr.Error())
		} else {
			backoff := time.Duration(1<<attempt) * time.Minute
			_, err = tx2.ExecContext(ctx,
				`UPDATE background_jobs SET status='pending', attempts=$2, error=$3, run_at=NOW()+$4, updated_at=NOW() WHERE id=$1`,
				row.id, attempt, jobErr.Error(), backoff.String())
		}
	}
	if err != nil {
		return true, err
	}
	return true, tx2.Commit()
}

func runJob(ctx context.Context, row jobRow) (err error) {
	mu.RLock()
	factory, ok := registry[row.jobType]
	mu.RUnlock()
	if !ok {
		log.Printf("jobs: unknown job type %q (id=%d)", row.jobType, row.id)
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	job := factory()
	if err := json.Unmarshal([]byte(row.payload), job); err != nil {
		log.Printf("jobs: unmarshal error for job %d: %v", row.id, err)
	}
	return job.Perform(ctx)
}

func ensureTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS background_jobs (
			id          BIGSERIAL PRIMARY KEY,
			type        TEXT        NOT NULL,
			payload     TEXT        NOT NULL DEFAULT '{}',
			status      TEXT        NOT NULL DEFAULT 'pending',
			attempts    INT         NOT NULL DEFAULT 0,
			error       TEXT,
			run_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
