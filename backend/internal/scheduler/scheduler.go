// Package scheduler wires gocron jobs for the RSS fetcher and the digest.
package scheduler

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-co-op/gocron/v2"
)

type Scheduler struct {
	s   gocron.Scheduler
	log *slog.Logger
}

func New(tz string, log *slog.Logger) (*Scheduler, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		log.Warn("invalid TZ, falling back to UTC", "tz", tz, "err", err)
		loc = time.UTC
	}
	s, err := gocron.NewScheduler(gocron.WithLocation(loc))
	if err != nil {
		return nil, fmt.Errorf("scheduler new: %w", err)
	}
	return &Scheduler{s: s, log: log}, nil
}

// AddCron registers a cron-style job. The standard 5-field form (no seconds).
func (sc *Scheduler) AddCron(name, expr string, fn func(ctx context.Context)) error {
	job, err := sc.s.NewJob(
		gocron.CronJob(expr, false /* withSeconds */),
		gocron.NewTask(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
			defer cancel()
			fn(ctx)
		}),
		gocron.WithName(name),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)
	if err != nil {
		return fmt.Errorf("add cron %q: %w", name, err)
	}
	sc.log.Info("scheduled", "job", name, "expr", expr, "id", job.ID().String())
	return nil
}

func (sc *Scheduler) Start() { sc.s.Start() }

func (sc *Scheduler) Stop() error {
	return sc.s.Shutdown()
}
