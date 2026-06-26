package gormrepo

import (
	"coffee-ranker/entity"
	"coffee-ranker/repository/gorm/model"
)

// entity.Userをmodel.Userに変換。
// Create, UpdateなどDBに保存する前に使う。
func toUserModel(user *entity.User) *model.User {
	if user == nil {
		return nil
	}
	return &model.User{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         string(user.Role),
		Status:       string(user.Status),
		TokenVersion: user.TokenVersion,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

// model.Userをentity.Userに変換。
// FindByID, FindByEmailなどDBから取得した後に使う。
func toUserEntity(user *model.User) *entity.User {
	if user == nil {
		return nil
	}
	return &entity.User{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		Role:         entity.UserRole(user.Role),
		Status:       entity.UserStatus(user.Status),
		TokenVersion: user.TokenVersion,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

// entity.RefreshTokenをmodel.RefreshTokenに変換。
// RefreshTokenをDBに保存する前に使う。
func toRefreshTokenModel(token *entity.RefreshToken) *model.RefreshToken {
	if token == nil {
		return nil
	}
	return &model.RefreshToken{
		ID:                token.ID,
		UserID:            token.UserID,
		TokenHash:         token.TokenHash,
		FamilyID:          token.FamilyID,
		UsedAt:            token.UsedAt,
		ReplacedByTokenID: token.ReplacedByTokenID,
		RevokedAt:         token.RevokedAt,
		ExpiresAt:         token.ExpiresAt,
		CreatedAt:         token.CreatedAt,
	}
}

// model.RefreshTokenをentity.RefreshTokenに変換。
// RefreshやLogout時にDBから取得した後に使う。
func toRefreshTokenEntity(token *model.RefreshToken) *entity.RefreshToken {
	if token == nil {
		return nil
	}
	return &entity.RefreshToken{
		ID:                token.ID,
		UserID:            token.UserID,
		TokenHash:         token.TokenHash,
		FamilyID:          token.FamilyID,
		UsedAt:            token.UsedAt,
		ReplacedByTokenID: token.ReplacedByTokenID,
		RevokedAt:         token.RevokedAt,
		ExpiresAt:         token.ExpiresAt,
		CreatedAt:         token.CreatedAt,
	}
}

// entity.GuestSessionをmodel.GuestSessionに変換。
// GuestSessionをDBに保存する前に使う。
func toGuestSessionModel(session *entity.GuestSession) *model.GuestSession {
	if session == nil {
		return nil
	}
	return &model.GuestSession{
		ID:             session.ID,
		SessionKeyHash: session.SessionKeyHash,
		FirstSeenAt:    session.FirstSeenAt,
		LastSeenAt:     session.LastSeenAt,
		ExpiresAt:      session.ExpiresAt,
	}
}

// model.GuestSessionをentity.GuestSessionに変換。
// GuestSessionをDBから取得した後に使う。
func toGuestSessionEntity(session *model.GuestSession) *entity.GuestSession {
	if session == nil {
		return nil
	}
	return &entity.GuestSession{
		ID:             session.ID,
		SessionKeyHash: session.SessionKeyHash,
		FirstSeenAt:    session.FirstSeenAt,
		LastSeenAt:     session.LastSeenAt,
		ExpiresAt:      session.ExpiresAt,
	}
}

// entity.Beanをmodel.Beanに変換。
// BeanをDBに保存する前に使う。
func toBeanModel(bean *entity.Bean) *model.Bean {
	if bean == nil {
		return nil
	}
	return &model.Bean{
		ID:          bean.ID,
		Name:        bean.Name,
		Roaster:     bean.Roaster,
		Origin:      bean.Origin,
		Region:      bean.Region,
		Farm:        bean.Farm,
		Variety:     bean.Variety,
		RoastLevel:  string(bean.RoastLevel),
		Acidity:     bean.Acidity,
		Bitterness:  bean.Bitterness,
		Flavor:      bean.Flavor,
		Aroma:       bean.Aroma,
		Body:        bean.Body,
		FlavorNote:  bean.FlavorNote,
		Description: bean.Description,
		ImageURL:    bean.ImageURL,
		IsPublished: bean.IsPublished,
		CreatedAt:   bean.CreatedAt,
		UpdatedAt:   bean.UpdatedAt,
	}
}

// model.Beanをentity.Beanに変換。
// BeanをDBから取得した後に使う。
func toBeanEntity(bean *model.Bean) *entity.Bean {
	if bean == nil {
		return nil
	}
	return &entity.Bean{
		ID:          bean.ID,
		Name:        bean.Name,
		Roaster:     bean.Roaster,
		Origin:      bean.Origin,
		Region:      bean.Region,
		Farm:        bean.Farm,
		Variety:     bean.Variety,
		RoastLevel:  entity.RoastLevel(bean.RoastLevel),
		Acidity:     bean.Acidity,
		Bitterness:  bean.Bitterness,
		Flavor:      bean.Flavor,
		Aroma:       bean.Aroma,
		Body:        bean.Body,
		FlavorNote:  bean.FlavorNote,
		Description: bean.Description,
		ImageURL:    bean.ImageURL,
		IsPublished: bean.IsPublished,
		CreatedAt:   bean.CreatedAt,
		UpdatedAt:   bean.UpdatedAt,
	}
}

// entity.Articleをmodel.Articleに変換。
// ArticleをDBに保存する前に使う。
func toArticleModel(article *entity.Article) *model.Article {
	if article == nil {
		return nil
	}
	return &model.Article{
		ID:          article.ID,
		Title:       article.Title,
		Slug:        article.Slug,
		Summary:     article.Summary,
		Body:        article.Body,
		Category:    article.Category,
		SourceName:  article.SourceName,
		SourceURL:   article.SourceURL,
		ImageURL:    article.ImageURL,
		IsPublished: article.IsPublished,
		PublishedAt: article.PublishedAt,
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
	}
}

// model.Articleをentity.Articleに変換。
// ArticleをDBから取得した後に使う。
func toArticleEntity(article *model.Article) *entity.Article {
	if article == nil {
		return nil
	}
	return &entity.Article{
		ID:          article.ID,
		Title:       article.Title,
		Slug:        article.Slug,
		Summary:     article.Summary,
		Body:        article.Body,
		Category:    article.Category,
		SourceName:  article.SourceName,
		SourceURL:   article.SourceURL,
		ImageURL:    article.ImageURL,
		IsPublished: article.IsPublished,
		PublishedAt: article.PublishedAt,
		CreatedAt:   article.CreatedAt,
		UpdatedAt:   article.UpdatedAt,
	}
}

// entity.BeanArticleをmodel.BeanArticleに変換。
// BeanとArticleの関連をDBに保存する前に使う。
func toBeanArticleModel(relation *entity.BeanArticle) *model.BeanArticle {
	if relation == nil {
		return nil
	}
	return &model.BeanArticle{
		ID:           relation.ID,
		BeanID:       relation.BeanID,
		ArticleID:    relation.ArticleID,
		DisplayOrder: relation.DisplayOrder,
		CreatedAt:    relation.CreatedAt,
	}
}

// model.BeanArticleをentity.BeanArticleに変換。
// BeanとArticleの関連をDBから取得した後に使う。
func toBeanArticleEntity(relation *model.BeanArticle) *entity.BeanArticle {
	if relation == nil {
		return nil
	}
	return &entity.BeanArticle{
		ID:           relation.ID,
		BeanID:       relation.BeanID,
		ArticleID:    relation.ArticleID,
		DisplayOrder: relation.DisplayOrder,
		CreatedAt:    relation.CreatedAt,
	}
}

// entity.RankTargetをmodel.RankTargetに変換。
// RankTargetをDBに保存する前に使う。
func toRankTargetModel(target *entity.RankTarget) *model.RankTarget {
	if target == nil {
		return nil
	}
	return &model.RankTarget{
		ID:          target.ID,
		ContentType: string(target.ContentType),
		ContentID:   target.ContentID,
		IsActive:    target.IsActive,
		CreatedAt:   target.CreatedAt,
		UpdatedAt:   target.UpdatedAt,
	}
}

// model.RankTargetをentity.RankTargetに変換。
// RankTargetをDBから取得した後に使う。
func toRankTargetEntity(target *model.RankTarget) *entity.RankTarget {
	if target == nil {
		return nil
	}
	return &entity.RankTarget{
		ID:          target.ID,
		ContentType: entity.ContentType(target.ContentType),
		ContentID:   target.ContentID,
		IsActive:    target.IsActive,
		CreatedAt:   target.CreatedAt,
		UpdatedAt:   target.UpdatedAt,
	}
}

// entity.ActionEventをmodel.ActionEventに変換。
// 行動イベントをDBに保存する前に使う。
func toActionEventModel(event *entity.ActionEvent) *model.ActionEvent {
	if event == nil {
		return nil
	}
	var ratingScore *int
	if event.RatingScore != nil {
		value := int(*event.RatingScore)
		ratingScore = &value
	}
	var roastLevel *string
	if event.SearchRoastLevel != nil {
		value := string(*event.SearchRoastLevel)
		roastLevel = &value
	}
	return &model.ActionEvent{
		ID:                    event.ID,
		UserID:                event.UserID,
		GuestSessionID:        event.GuestSessionID,
		EventType:             string(event.EventType),
		RankTargetID:          event.RankTargetID,
		Placement:             string(event.Placement),
		DwellMs:               event.DwellMs,
		RatingScore:           ratingScore,
		SearchConditionHash:   event.SearchConditionHash,
		PreviousConditionHash: event.PreviousConditionHash,
		SearchKeyword:         event.SearchKeyword,
		SearchOrigin:          event.SearchOrigin,
		SearchRoastLevel:      roastLevel,
		SearchAcidity:         event.SearchAcidity,
		SearchBitterness:      event.SearchBitterness,
		SearchAroma:           event.SearchAroma,
		SearchFlavor:          event.SearchFlavor,
		SearchBody:            event.SearchBody,
		SearchCategory:        event.SearchCategory,
		ModalDisplayLogID:     event.ModalDisplayLogID,
		PagePath:              event.PagePath,
		ReferrerPath:          event.ReferrerPath,
		UserAgent:             event.UserAgent,
		IPAddressHash:         event.IPAddressHash,
		RequestID:             event.RequestID,
		OccurredAt:            event.OccurredAt,
	}
}

// model.ActionEventをentity.ActionEventに変換。
// 行動イベントをDBから取得した後に使う。
func toActionEventEntity(event *model.ActionEvent) *entity.ActionEvent {
	if event == nil {
		return nil
	}
	var ratingScore *entity.RatingScore
	if event.RatingScore != nil {
		value := entity.RatingScore(*event.RatingScore)
		ratingScore = &value
	}
	var roastLevel *entity.RoastLevel
	if event.SearchRoastLevel != nil {
		value := entity.RoastLevel(*event.SearchRoastLevel)
		roastLevel = &value
	}
	return &entity.ActionEvent{
		ID:                    event.ID,
		UserID:                event.UserID,
		GuestSessionID:        event.GuestSessionID,
		EventType:             entity.EventType(event.EventType),
		RankTargetID:          event.RankTargetID,
		Placement:             entity.Placement(event.Placement),
		DwellMs:               event.DwellMs,
		RatingScore:           ratingScore,
		SearchConditionHash:   event.SearchConditionHash,
		PreviousConditionHash: event.PreviousConditionHash,
		SearchKeyword:         event.SearchKeyword,
		SearchOrigin:          event.SearchOrigin,
		SearchRoastLevel:      roastLevel,
		SearchAcidity:         event.SearchAcidity,
		SearchBitterness:      event.SearchBitterness,
		SearchAroma:           event.SearchAroma,
		SearchFlavor:          event.SearchFlavor,
		SearchBody:            event.SearchBody,
		SearchCategory:        event.SearchCategory,
		ModalDisplayLogID:     event.ModalDisplayLogID,
		PagePath:              event.PagePath,
		ReferrerPath:          event.ReferrerPath,
		UserAgent:             event.UserAgent,
		IPAddressHash:         event.IPAddressHash,
		RequestID:             event.RequestID,
		OccurredAt:            event.OccurredAt,
	}
}

// entity.ModalDisplayLogをmodel.ModalDisplayLogに変換。
// モーダル表示履歴をDBに保存する前に使う。
func toModalDisplayLogModel(log *entity.ModalDisplayLog) *model.ModalDisplayLog {
	if log == nil {
		return nil
	}
	return &model.ModalDisplayLog{
		ID:             log.ID,
		UserID:         log.UserID,
		GuestSessionID: log.GuestSessionID,
		RankTargetID:   log.RankTargetID,
		Trigger:        string(log.Trigger),
		PagePath:       log.PagePath,
		ShownAt:        log.ShownAt,
		ClickedAt:      log.ClickedAt,
		ClosedAt:       log.ClosedAt,
		CreatedAt:      log.CreatedAt,
	}
}

// model.ModalDisplayLogをentity.ModalDisplayLogに変換。
// モーダル表示履歴をDBから取得した後に使う。
func toModalDisplayLogEntity(log *model.ModalDisplayLog) *entity.ModalDisplayLog {
	if log == nil {
		return nil
	}
	return &entity.ModalDisplayLog{
		ID:             log.ID,
		UserID:         log.UserID,
		GuestSessionID: log.GuestSessionID,
		RankTargetID:   log.RankTargetID,
		Trigger:        entity.ModalTrigger(log.Trigger),
		PagePath:       log.PagePath,
		ShownAt:        log.ShownAt,
		ClickedAt:      log.ClickedAt,
		ClosedAt:       log.ClosedAt,
		CreatedAt:      log.CreatedAt,
	}
}

// entity.ModalBlockLogをmodel.ModalBlockLogに変換。
// モーダルを表示しなかった理由をDBに保存する前に使う。
func toModalBlockLogModel(log *entity.ModalBlockLog) *model.ModalBlockLog {
	if log == nil {
		return nil
	}
	return &model.ModalBlockLog{
		ID:                    log.ID,
		UserID:                log.UserID,
		GuestSessionID:        log.GuestSessionID,
		CandidateRankTargetID: log.CandidateRankTargetID,
		Reason:                string(log.Reason),
		PagePath:              log.PagePath,
		BlockedAt:             log.BlockedAt,
	}
}

// model.ModalBlockLogをentity.ModalBlockLogに変換。
// モーダルを表示しなかった理由をDBから取得した後に使う。
func toModalBlockLogEntity(log *model.ModalBlockLog) *entity.ModalBlockLog {
	if log == nil {
		return nil
	}
	return &entity.ModalBlockLog{
		ID:                    log.ID,
		UserID:                log.UserID,
		GuestSessionID:        log.GuestSessionID,
		CandidateRankTargetID: log.CandidateRankTargetID,
		Reason:                entity.ModalBlockReason(log.Reason),
		PagePath:              log.PagePath,
		BlockedAt:             log.BlockedAt,
	}
}

// entity.SavedItemをmodel.SavedItemに変換。
// 保存状態をDBに保存する前に使う。
func toSavedItemModel(item *entity.SavedItem) *model.SavedItem {
	if item == nil {
		return nil
	}
	return &model.SavedItem{
		ID:           item.ID,
		UserID:       item.UserID,
		RankTargetID: item.RankTargetID,
		RemovedAt:    item.RemovedAt,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

// model.SavedItemをentity.SavedItemに変換。
// 保存状態をDBから取得した後に使う。
func toSavedItemEntity(item *model.SavedItem) *entity.SavedItem {
	if item == nil {
		return nil
	}
	return &entity.SavedItem{
		ID:           item.ID,
		UserID:       item.UserID,
		RankTargetID: item.RankTargetID,
		RemovedAt:    item.RemovedAt,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}

// entity.Ratingをmodel.Ratingに変換。
// Good/Bad評価をDBに保存する前に使う。
func toRatingModel(rating *entity.Rating) *model.Rating {
	if rating == nil {
		return nil
	}
	return &model.Rating{
		ID:           rating.ID,
		UserID:       rating.UserID,
		RankTargetID: rating.RankTargetID,
		Score:        int(rating.Score),
		CreatedAt:    rating.CreatedAt,
		UpdatedAt:    rating.UpdatedAt,
	}
}

// model.Ratingをentity.Ratingに変換。
// Good/Bad評価をDBから取得した後に使う。
func toRatingEntity(rating *model.Rating) *entity.Rating {
	if rating == nil {
		return nil
	}
	return &entity.Rating{
		ID:           rating.ID,
		UserID:       rating.UserID,
		RankTargetID: rating.RankTargetID,
		Score:        entity.RatingScore(rating.Score),
		CreatedAt:    rating.CreatedAt,
		UpdatedAt:    rating.UpdatedAt,
	}
}

// entity.ContentMetricをmodel.ContentMetricに変換。
// ランキング指標をDBに保存する前に使う。
func toContentMetricModel(metric *entity.ContentMetric) *model.ContentMetric {
	if metric == nil {
		return nil
	}
	return &model.ContentMetric{
		ID:                   metric.ID,
		RankTargetID:         metric.RankTargetID,
		Score:                metric.Score,
		ImpressionCount:      metric.ImpressionCount,
		ContentViewCount:     metric.ContentViewCount,
		ClickCount:           metric.ClickCount,
		StayTotalMs:          metric.StayTotalMs,
		SaveCount:            metric.SaveCount,
		RatingCount:          metric.RatingCount,
		GoodCount:            metric.GoodCount,
		BadCount:             metric.BadCount,
		RatingAvg:            metric.RatingAvg,
		GoodRate:             metric.GoodRate,
		BadRate:              metric.BadRate,
		ModalImpressionCount: metric.ModalImpressionCount,
		ModalClickCount:      metric.ModalClickCount,
		ModalCloseCount:      metric.ModalCloseCount,
		ClickRate:            metric.ClickRate,
		SaveRate:             metric.SaveRate,
		ModalClickRate:       metric.ModalClickRate,
		ModalCloseRate:       metric.ModalCloseRate,
		PeriodStart:          metric.PeriodStart,
		PeriodEnd:            metric.PeriodEnd,
		CalculatedAt:         metric.CalculatedAt,
		UpdatedAt:            metric.UpdatedAt,
	}
}

// model.ContentMetricをentity.ContentMetricに変換。
// ランキング指標をDBから取得した後に使う。
func toContentMetricEntity(metric *model.ContentMetric) *entity.ContentMetric {
	if metric == nil {
		return nil
	}
	return &entity.ContentMetric{
		ID:                   metric.ID,
		RankTargetID:         metric.RankTargetID,
		Score:                metric.Score,
		ImpressionCount:      metric.ImpressionCount,
		ContentViewCount:     metric.ContentViewCount,
		ClickCount:           metric.ClickCount,
		StayTotalMs:          metric.StayTotalMs,
		SaveCount:            metric.SaveCount,
		RatingCount:          metric.RatingCount,
		GoodCount:            metric.GoodCount,
		BadCount:             metric.BadCount,
		RatingAvg:            metric.RatingAvg,
		GoodRate:             metric.GoodRate,
		BadRate:              metric.BadRate,
		ModalImpressionCount: metric.ModalImpressionCount,
		ModalClickCount:      metric.ModalClickCount,
		ModalCloseCount:      metric.ModalCloseCount,
		ClickRate:            metric.ClickRate,
		SaveRate:             metric.SaveRate,
		ModalClickRate:       metric.ModalClickRate,
		ModalCloseRate:       metric.ModalCloseRate,
		PeriodStart:          metric.PeriodStart,
		PeriodEnd:            metric.PeriodEnd,
		CalculatedAt:         metric.CalculatedAt,
		UpdatedAt:            metric.UpdatedAt,
	}
}

// entity.InterestProfileをmodel.InterestProfileに変換。
// 興味プロフィールをDBに保存する前に使う。
func toInterestProfileModel(profile *entity.InterestProfile) *model.InterestProfile {
	if profile == nil {
		return nil
	}
	return &model.InterestProfile{
		ID:             profile.ID,
		UserID:         profile.UserID,
		GuestSessionID: profile.GuestSessionID,
		Dimension:      string(profile.Dimension),
		Value:          profile.Value,
		Score:          profile.Score,
		LastEventAt:    profile.LastEventAt,
		ExpiresAt:      profile.ExpiresAt,
		CreatedAt:      profile.CreatedAt,
		UpdatedAt:      profile.UpdatedAt,
	}
}

// model.InterestProfileをentity.InterestProfileに変換。
// 興味プロフィールをDBから取得した後に使う。
func toInterestProfileEntity(profile *model.InterestProfile) *entity.InterestProfile {
	if profile == nil {
		return nil
	}
	return &entity.InterestProfile{
		ID:             profile.ID,
		UserID:         profile.UserID,
		GuestSessionID: profile.GuestSessionID,
		Dimension:      entity.InterestDimension(profile.Dimension),
		Value:          profile.Value,
		Score:          profile.Score,
		LastEventAt:    profile.LastEventAt,
		ExpiresAt:      profile.ExpiresAt,
		CreatedAt:      profile.CreatedAt,
		UpdatedAt:      profile.UpdatedAt,
	}
}

// entity.BatchRunをmodel.BatchRunに変換。
// バッチ実行履歴をDBに保存する前に使う。
func toBatchRunModel(run *entity.BatchRun) *model.BatchRun {
	if run == nil {
		return nil
	}
	return &model.BatchRun{
		ID:              run.ID,
		JobName:         run.JobName,
		Status:          string(run.Status),
		StartedAt:       run.StartedAt,
		FinishedAt:      run.FinishedAt,
		RowsProcessed:   run.RowsProcessed,
		ErrorMessage:    run.ErrorMessage,
		TriggeredBy:     string(run.TriggeredBy),
		TriggeredUserID: run.TriggeredUserID,
	}
}

// model.BatchRunをentity.BatchRunに変換。
// バッチ実行履歴をDBから取得した後に使う。
func toBatchRunEntity(run *model.BatchRun) *entity.BatchRun {
	if run == nil {
		return nil
	}
	return &entity.BatchRun{
		ID:              run.ID,
		JobName:         run.JobName,
		Status:          entity.BatchStatus(run.Status),
		StartedAt:       run.StartedAt,
		FinishedAt:      run.FinishedAt,
		RowsProcessed:   run.RowsProcessed,
		ErrorMessage:    run.ErrorMessage,
		TriggeredBy:     entity.AuditActorType(run.TriggeredBy),
		TriggeredUserID: run.TriggeredUserID,
	}
}

// entity.AuditLogをmodel.AuditLogに変換。
// 監査ログをDBに保存する前に使う。
func toAuditLogModel(log *entity.AuditLog) *model.AuditLog {
	if log == nil {
		return nil
	}
	return &model.AuditLog{
		ID:            log.ID,
		ActorType:     string(log.ActorType),
		ActorUserID:   log.ActorUserID,
		Action:        string(log.Action),
		TargetType:    log.TargetType,
		TargetID:      log.TargetID,
		Detail:        log.Detail,
		IPAddressHash: log.IPAddressHash,
		RequestID:     log.RequestID,
		CreatedAt:     log.CreatedAt,
	}
}

// model.AuditLogをentity.AuditLogに変換。
// 監査ログをDBから取得した後に使う。
func toAuditLogEntity(log *model.AuditLog) *entity.AuditLog {
	if log == nil {
		return nil
	}
	return &entity.AuditLog{
		ID:            log.ID,
		ActorType:     entity.AuditActorType(log.ActorType),
		ActorUserID:   log.ActorUserID,
		Action:        entity.AuditAction(log.Action),
		TargetType:    log.TargetType,
		TargetID:      log.TargetID,
		Detail:        log.Detail,
		IPAddressHash: log.IPAddressHash,
		RequestID:     log.RequestID,
		CreatedAt:     log.CreatedAt,
	}
}
