package main

import (
	"context"
	"log"
	"strconv"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/usecase"
)

func startDailyRankingBatch(ctx context.Context, ranking *usecase.RankingBatchUsecase, logger *log.Logger) {
	if ranking == nil {
		return
	}
	if logger == nil {
		logger = log.Default()
	}

	go func() {
		for {
			next := nextDailyBatchTime(time.Now(), entity.RankingBatchHour)
			timer := time.NewTimer(time.Until(next))

			select {
			case <-ctx.Done():
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				return
			case <-timer.C:
				runDailyRankingBatch(ctx, ranking, logger)
			}
		}
	}()
}

func nextDailyBatchTime(now time.Time, hour int) time.Time {
	if hour < 0 || hour > 23 {
		hour = entity.RankingBatchHour
	}

	next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func runDailyRankingBatch(parent context.Context, ranking *usecase.RankingBatchUsecase, logger *log.Logger) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Minute)
	defer cancel()

	owner := "system_daily_ranking_" + strconv.FormatInt(time.Now().UnixNano(), 10)
	run, err := ranking.Recalculate(ctx, usecase.BatchInput{
		JobName:     "ranking",
		Owner:       owner,
		TriggeredBy: entity.AuditActorSystem,
	})
	if err != nil {
		logger.Printf("daily ranking batch failed: %v", err)
		return
	}
	if run == nil {
		logger.Print("daily ranking batch finished without batch run result")
		return
	}

	logger.Printf("daily ranking batch finished: run_id=%d rows=%d", run.ID, run.RowsProcessed)
}
