package runtime

import (
	"context"
	"log"
	"time"

	"github.com/larb26656/my-feed/backend/internal/fetch/service"
)

// DefaultExecutorPoll is how often the in-process executor looks for claimable jobs.
const DefaultExecutorPoll = 2 * time.Second

// Loops runs the fixed-rate schedule round and the claim/execute poller until ctx is cancelled.
type Loops struct {
	svc           *service.Service
	executorPoll  time.Duration
}

func NewLoops(svc *service.Service, executorPoll time.Duration) *Loops {
	if executorPoll <= 0 {
		executorPoll = DefaultExecutorPoll
	}
	return &Loops{svc: svc, executorPoll: executorPoll}
}

// Start launches scheduler and executor goroutines. They stop when ctx is cancelled.
func (l *Loops) Start(ctx context.Context) {
	go l.runScheduler(ctx)
	go l.runExecutor(ctx)
}

func (l *Loops) runScheduler(ctx context.Context) {
	interval := l.svc.ScheduleInterval()
	// Run once on start so a fresh API process does not wait a full interval.
	l.scheduleOnce(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.scheduleOnce(ctx)
		}
	}
}

func (l *Loops) scheduleOnce(ctx context.Context) {
	if _, err := l.svc.ScheduleRound(ctx); err != nil && ctx.Err() == nil {
		log.Printf("fetch schedule round: %v", err)
	}
}

func (l *Loops) runExecutor(ctx context.Context) {
	ticker := time.NewTicker(l.executorPoll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := l.svc.ExecuteReadyFetchJobs(ctx); err != nil && ctx.Err() == nil {
				log.Printf("fetch executor: %v", err)
			}
		}
	}
}
