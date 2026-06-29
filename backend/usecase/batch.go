package usecase

import (
	"context"
	"strconv"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// ランキング用の集計バッチ。
// action_eventsを集計し、content_metricsにランキング指標として保存。
type RankingBatchUsecase struct {
	events  repository.IActionEventRepository
	metrics repository.IContentMetricRepository
	runs    repository.IBatchRunRepository
	locks   repository.IBatchLockRepository
	audits  repository.IAuditLogRepository
	tx      repository.TxManager
}

// 興味プロフィール用の集計バッチ。
// User/Guestの行動ログを集計し、interest_profilesを更新。
type InterestBatchUsecase struct {
	events    repository.IActionEventRepository
	profiles  repository.IInterestProfileRepository
	runs      repository.IBatchRunRepository
	locks     repository.IBatchLockRepository
	audits    repository.IAuditLogRepository
	expiresIn time.Duration
}

// バッチ実行時に外から渡す共通情報。
// 未指定の値はnormalizeBatchInputで安全な初期値に。
type BatchInput struct {
	JobName         string
	Owner           string
	LockTTL         time.Duration
	TriggeredBy     entity.AuditActorType
	TriggeredUserID *uint64
	Meta            AuditMeta
}

// 興味プロフィールを再計算する対象。
// UserとGuestSessionのどちらも指定できる。
type InterestBatchInput struct {
	BatchInput
	UserIDs         []uint64
	GuestSessionIDs []uint64
}

// RankingBatchUsecase。
func NewRankingBatchUsecase(
	events repository.IActionEventRepository,
	metrics repository.IContentMetricRepository,
	runs repository.IBatchRunRepository,
	locks repository.IBatchLockRepository,
	audits repository.IAuditLogRepository,
	tx repository.TxManager,
) *RankingBatchUsecase {
	return &RankingBatchUsecase{
		events:  events,
		metrics: metrics,
		runs:    runs,
		locks:   locks,
		audits:  audits,
		tx:      tx,
	}
}

// InterestBatchUsecase
// expiresInは、Guestなど一時的な興味プロフィールに有効期限を付けるために使う。
func NewInterestBatchUsecase(
	events repository.IActionEventRepository,
	profiles repository.IInterestProfileRepository,
	runs repository.IBatchRunRepository,
	locks repository.IBatchLockRepository,
	audits repository.IAuditLogRepository,
	expiresIn time.Duration,
) *InterestBatchUsecase {
	return &InterestBatchUsecase{
		events:    events,
		profiles:  profiles,
		runs:      runs,
		locks:     locks,
		audits:    audits,
		expiresIn: expiresIn,
	}
}

// ランキング指標を再計算する。
// 同じバッチが同時に走ると集計結果が競合するため、最初にRedis lockを取る。
func (u *RankingBatchUsecase) Recalculate(ctx context.Context, input BatchInput) (*model.BatchRun, error) {
	input = normalizeBatchInput(input, "ranking")

	lockKey := "batch:" + input.JobName

	// 二重実行を防ぐため、job単位でlockを取る。
	locked, err := u.locks.Acquire(ctx, lockKey, input.Owner, input.LockTTL)
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, entity.ErrBatchAlreadyRunning
	}
	defer func() {
		// 処理終了後はlockを解放する。
		// ctxがキャンセル済みでも解放を試すため、Backgroundを使う。
		_ = u.locks.Release(context.Background(), lockKey, input.Owner)
	}()

	now := time.Now()

	// バッチ実行履歴をrunning状態で作成。
	// 失敗時・成功時にこのrunを更新
	run := &model.BatchRun{
		JobName:         input.JobName,
		Status:          entity.BatchStatusRunning,
		StartedAt:       now,
		TriggeredBy:     input.TriggeredBy,
		TriggeredUserID: input.TriggeredUserID,
	}
	if err := u.runs.Create(ctx, run); err != nil {
		return nil, err
	}

	// 直近RankingWindowDays日分の行動ログをランキング集計対象にする。
	periodEnd := now
	periodStart := now.AddDate(0, 0, -entity.RankingWindowDays)

	// action_eventsをrank_target単位で集計する。
	aggregates, err := u.events.AggregateContentMetrics(ctx, periodStart, periodEnd)
	if err != nil {
		_ = u.runs.MarkFailed(ctx, run.ID, time.Now(), 0, err.Error())
		return nil, err
	}

	// 集計結果をcontent_metricsに保存できる形へ変換する。
	metrics := make([]*model.ContentMetric, 0, len(aggregates))
	for _, aggregate := range aggregates {
		metrics = append(metrics, contentMetricFromAggregate(aggregate, periodStart, periodEnd, now))
	}

	finishedAt := time.Now()

	// 指標更新とBatchRun成功更新は同じTxで扱う。
	// 片方だけ成功すると「指標は更新済みなのにrunはrunningのまま」などの不整合が起きる。
	err = u.tx.WithinTx(ctx, func(ctx context.Context, tx repository.ITxRepos) error {
		if err := tx.ContentMetric().BulkUpsert(ctx, metrics); err != nil {
			return err
		}
		return tx.BatchRun().MarkSuccess(ctx, run.ID, finishedAt, int64(len(metrics)))
	})
	if err != nil {
		_ = u.runs.MarkFailed(ctx, run.ID, time.Now(), int64(len(metrics)), err.Error())
		return nil, err
	}

	// DB更新後、返却するrunにも成功状態を反映。
	run.Status = entity.BatchStatusSuccess
	run.FinishedAt = &finishedAt
	run.RowsProcessed = int64(len(metrics))

	// 監査ログは補助記録。
	// 失敗してもランキング再計算自体は成功扱いにする。
	targetType := "batch_run"
	safeAudit(ctx, u.audits, auditLog(
		input.TriggeredBy,
		input.TriggeredUserID,
		entity.AuditActionRankingBatchRun,
		&targetType,
		&run.ID,
		nil,
		input.Meta,
	))

	return run, nil
}

// User/Guestの興味プロフィールを再計算する。
// 行動ログを集計し、originやroast_levelなどの興味スコアを保存。
func (u *InterestBatchUsecase) Recalculate(ctx context.Context, input InterestBatchInput) (*model.BatchRun, error) {
	input.BatchInput = normalizeBatchInput(input.BatchInput, "interest")

	lockKey := "batch:" + input.JobName

	// interest batchも二重実行すると同じプロフィールを同時更新するため、lockを取る。
	locked, err := u.locks.Acquire(ctx, lockKey, input.Owner, input.LockTTL)
	if err != nil {
		return nil, err
	}
	if !locked {
		return nil, entity.ErrBatchAlreadyRunning
	}
	defer func() {
		_ = u.locks.Release(context.Background(), lockKey, input.Owner)
	}()

	now := time.Now()

	// 実行履歴をrunningで作成。
	run := &model.BatchRun{
		JobName:         input.JobName,
		Status:          entity.BatchStatusRunning,
		StartedAt:       now,
		TriggeredBy:     input.TriggeredBy,
		TriggeredUserID: input.TriggeredUserID,
	}
	if err := u.runs.Create(ctx, run); err != nil {
		return nil, err
	}

	// ランキングと同じ集計窓を使う。
	periodEnd := now
	periodStart := now.AddDate(0, 0, -entity.RankingWindowDays)

	profiles := make([]*model.InterestProfile, 0)

	// 指定されたUserごとに興味スコアを集計する。
	for _, userID := range input.UserIDs {
		if userID == 0 {
			continue
		}

		aggregates, err := u.events.AggregateUserInterest(ctx, userID, periodStart, periodEnd)
		if err != nil {
			_ = u.runs.MarkFailed(ctx, run.ID, time.Now(), int64(len(profiles)), err.Error())
			return nil, err
		}

		profiles = append(profiles, profilesFromAggregates(aggregates, u.expiresIn, now)...)
	}

	// 指定されたGuestSessionごとに興味スコアを集計する。
	for _, guestSessionID := range input.GuestSessionIDs {
		if guestSessionID == 0 {
			continue
		}

		aggregates, err := u.events.AggregateGuestInterest(ctx, guestSessionID, periodStart, periodEnd)
		if err != nil {
			_ = u.runs.MarkFailed(ctx, run.ID, time.Now(), int64(len(profiles)), err.Error())
			return nil, err
		}

		profiles = append(profiles, profilesFromAggregates(aggregates, u.expiresIn, now)...)
	}

	// 集計した興味プロフィールをまとめて保存する。
	if err := u.profiles.BulkUpsert(ctx, profiles); err != nil {
		_ = u.runs.MarkFailed(ctx, run.ID, time.Now(), int64(len(profiles)), err.Error())
		return nil, err
	}

	finishedAt := time.Now()

	// 保存できた件数をBatchRunに記録し、成功扱いにする。
	if err := u.runs.MarkSuccess(ctx, run.ID, finishedAt, int64(len(profiles))); err != nil {
		_ = u.runs.MarkFailed(ctx, run.ID, time.Now(), int64(len(profiles)), err.Error())
		return nil, err
	}

	// 返却用のrunにも成功状態を反映する。
	run.Status = entity.BatchStatusSuccess
	run.FinishedAt = &finishedAt
	run.RowsProcessed = int64(len(profiles))

	targetType := "batch_run"
	detail := "interest"

	// interest専用のAuditActionがないため、manual_batch_runとして記録。
	// detailにinterestを入れて、ランキングバッチと区別できるようにする。
	safeAudit(ctx, u.audits, auditLog(
		input.TriggeredBy,
		input.TriggeredUserID,
		entity.AuditActionManualBatchRun,
		&targetType,
		&run.ID,
		&detail,
		input.Meta,
	))

	return run, nil
}

// バッチ入力の不足値を補完する。
// job名、owner、lock TTL、実行者種別が空でも安全に動くように。
func normalizeBatchInput(input BatchInput, defaultJobName string) BatchInput {
	if input.JobName == "" {
		input.JobName = defaultJobName
	}

	if input.Owner == "" {
		input.Owner = defaultJobName + ":" + strconv.FormatInt(time.Now().UnixNano(), 10)
	}

	if input.LockTTL <= 0 {
		input.LockTTL = 10 * time.Minute
	}

	if input.TriggeredBy == "" {
		input.TriggeredBy = entity.AuditActorSystem
	}

	return input
}

// rank_target単位の集計結果をcontent_metrics用のmodelに変換。
// ここでランキングに使う各種rateとscoreを計算する。
func contentMetricFromAggregate(aggregate repository.ContentMetricAggregate, periodStart time.Time, periodEnd time.Time, calculatedAt time.Time) *model.ContentMetric {
	clickRate := rate(aggregate.ClickCount, aggregate.ImpressionCount)
	saveRate := rate(aggregate.SaveCount, aggregate.ContentViewCount)
	modalClickRate := rate(aggregate.ModalClickCount, aggregate.ModalImpressionCount)
	modalCloseRate := rate(aggregate.ModalCloseCount, aggregate.ModalImpressionCount)
	goodRate := rate(aggregate.GoodCount, aggregate.RatingCount)
	badRate := rate(aggregate.BadCount, aggregate.RatingCount)

	// Goodを+1、Badを-1として平均化する。
	// 例: Good 8件、Bad 2件なら (8 - 2) / 10 = 0.6。
	ratingAvg := ratingAverage(aggregate.GoodCount, aggregate.BadCount, aggregate.RatingCount)

	// クリック・保存・Good評価・モーダルクリックを加点し、Bad評価・モーダルcloseを減点する。
	score := clickRate*30 +
		saveRate*25 +
		goodRate*20 +
		modalClickRate*10 +
		float64(aggregate.StayTotalMs)/1000*0.01 -
		badRate*15 -
		modalCloseRate*10

	return &model.ContentMetric{
		RankTargetID:         aggregate.RankTargetID,
		Score:                score,
		ImpressionCount:      aggregate.ImpressionCount,
		ContentViewCount:     aggregate.ContentViewCount,
		ClickCount:           aggregate.ClickCount,
		StayTotalMs:          aggregate.StayTotalMs,
		SaveCount:            aggregate.SaveCount,
		RatingCount:          aggregate.RatingCount,
		GoodCount:            aggregate.GoodCount,
		BadCount:             aggregate.BadCount,
		RatingAvg:            ratingAvg,
		GoodRate:             goodRate,
		BadRate:              badRate,
		ModalImpressionCount: aggregate.ModalImpressionCount,
		ModalClickCount:      aggregate.ModalClickCount,
		ModalCloseCount:      aggregate.ModalCloseCount,
		ClickRate:            clickRate,
		SaveRate:             saveRate,
		ModalClickRate:       modalClickRate,
		ModalCloseRate:       modalCloseRate,
		PeriodStart:          periodStart,
		PeriodEnd:            periodEnd,
		CalculatedAt:         calculatedAt,
	}
}

// 興味集計結果をinterest_profiles用のmodelに変換する。
// expiresInが指定されている場合は、有効期限付きのプロフィールとして保存する。
func profilesFromAggregates(aggregates []repository.InterestAggregate, expiresIn time.Duration, now time.Time) []*model.InterestProfile {
	profiles := make([]*model.InterestProfile, 0, len(aggregates))

	for _, aggregate := range aggregates {
		// dimensionやvalueが空だと、何に対する興味か分からないため保存しない。
		if aggregate.Dimension == "" || aggregate.Value == "" {
			continue
		}

		var expiresAt *time.Time
		if expiresIn > 0 {
			expires := now.Add(expiresIn)
			expiresAt = &expires
		}

		profiles = append(profiles, &model.InterestProfile{
			UserID:         aggregate.UserID,
			GuestSessionID: aggregate.GuestSessionID,
			Dimension:      aggregate.Dimension,
			Value:          aggregate.Value,
			Score:          aggregate.ScoreDelta,
			LastEventAt:    aggregate.LastEventAt,
			ExpiresAt:      expiresAt,
		})
	}

	return profiles
}

// 割合を計算する。
// 分母が0以下なら、0除算を避けるため0を返す。
func rate(numerator int64, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

// Good/Bad評価を-1〜1の平均値に変換する。
// Goodが多いほどプラス、Badが多いほどマイナスになる。
func ratingAverage(goodCount int64, badCount int64, ratingCount int64) float64 {
	if ratingCount <= 0 {
		return 0
	}
	return float64(goodCount-badCount) / float64(ratingCount)
}
