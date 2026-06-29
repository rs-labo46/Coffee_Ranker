package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"

	"github.com/labstack/echo/v4"
)

// Controller単体テスト用に、HTTP method、path、bodyからEcho Contextを作る。
// RouterやMiddlewareを通さず、Controller単体のBind、Validator、HTTP変換だけを確認する。
func newTestContext(method string, target string, body string) (*echo.Echo, echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()
	return e, e.NewContext(req, rec), rec
}

// テスト用Request DTOをJSON文字列へ変換する。
// Controllerのc.Bindが実際のJSON bodyとして読み取れる形にする。
func jsonBody(t *testing.T, value interface{}) string {
	t.Helper()
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(value); err != nil {
		t.Fatalf("encode json failed: %v", err)
	}
	return buf.String()
}

// ResponseRecorderのHTTP statusが期待値と一致することを確認する。
// Controllerがdomain errorを正しいHTTP statusへ変換できているかを見る。
func assertStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d, body = %s", rec.Code, want, rec.Body.String())
	}
}

// 認証済みUserとして扱うため、Controllerが参照するContext keyへuser_idを入れる。
// bodyのuser_idではなくContextのuser_idだけを信用する設計をテストする。
func setUser(c echo.Context, id uint64) {
	c.Set(ContextUserIDKey, id)
}

// Guest actorとして扱うため、Controllerが参照するContext keyへguest_session_idを入れる。
// bodyのguest_session_idを信用しない設計をテストする。
func setGuest(c echo.Context, id uint64) {
	c.Set(ContextGuestSessionIDKey, id)
}

// uint64値へのポインタを作る。
// modelやusecaseのUserID/GuestSessionIDなど任意IDを組み立てる時に使う。
func uint64Ptr(value uint64) *uint64 {
	return &value
}

type fakeMetricRepo struct{}

// ランキング指標の保存処理はControllerテストでは対象外なので成功扱いにする。
func (r *fakeMetricRepo) Upsert(ctx context.Context, metric *model.ContentMetric) error {
	return nil
}

// ランキング指標の一括保存処理はControllerテストでは対象外なので成功扱いにする。
func (r *fakeMetricRepo) BulkUpsert(ctx context.Context, metrics []*model.ContentMetric) error {
	return nil
}

// 指定RankTargetIDのContentMetric取得を成功扱いにする。
func (r *fakeMetricRepo) FindByRankTargetID(ctx context.Context, rankTargetID uint64) (*model.ContentMetric, error) {
	return sampleMetric(rankTargetID, entity.ContentTypeBean, rankTargetID), nil
}

// 複数RankTargetIDのContentMetric取得を成功扱いにする。
func (r *fakeMetricRepo) FindByRankTargetIDs(ctx context.Context, rankTargetIDs []uint64) ([]*model.ContentMetric, error) {
	metrics := make([]*model.ContentMetric, 0, len(rankTargetIDs))
	for _, id := range rankTargetIDs {
		metrics = append(metrics, sampleMetric(id, entity.ContentTypeBean, id))
	}
	return metrics, nil
}

// RankingUsecase.List/RecommendationUsecase.Listが使うランキング取得を成功扱いにする。
// RankTargetも埋めて返し、Usecase内のcontent_type振り分けで落ちないようにする。
func (r *fakeMetricRepo) ListRanking(ctx context.Context, contentType *entity.ContentType, limit int, offset int) ([]*model.ContentMetric, error) {
	ct := entity.ContentTypeBean
	if contentType != nil {
		ct = *contentType
	}
	return []*model.ContentMetric{sampleMetric(1, ct, 1)}, nil
}

// RankingUsecase.Topが使うTOPランキング取得を成功扱いにする。
func (r *fakeMetricRepo) ListTopByScore(ctx context.Context, limit int) ([]*model.ContentMetric, error) {
	return []*model.ContentMetric{sampleMetric(1, entity.ContentTypeBean, 1)}, nil
}

// 最終集計日時取得はこのControllerテストでは使わないため現在時刻を返す。
func (r *fakeMetricRepo) LatestCalculatedAt(ctx context.Context) (*time.Time, error) {
	now := time.Now()
	return &now, nil
}

// Usecaseが処理できるContentMetricを作る。
// RankTargetを埋めないとRankingUsecase.Listの分岐でErrRankTargetNotFoundになる。
func sampleMetric(rankTargetID uint64, contentType entity.ContentType, contentID uint64) *model.ContentMetric {
	return &model.ContentMetric{
		ID:           rankTargetID,
		RankTargetID: rankTargetID,
		Score:        100,
		RankTarget: model.RankTarget{
			ID:          rankTargetID,
			ContentType: contentType,
			ContentID:   contentID,
			IsActive:    true,
		},
	}
}

type fakeRankTargetRepo struct{}

// RankTarget作成はControllerテストでは対象外なので成功扱いにする。
func (r *fakeRankTargetRepo) Create(ctx context.Context, target *model.RankTarget) error {
	if target != nil && target.ID == 0 {
		target.ID = 1
	}
	return nil
}

// RankTargetIDで有効なランキング対象を返す。
func (r *fakeRankTargetRepo) FindByID(ctx context.Context, id uint64) (*model.RankTarget, error) {
	return &model.RankTarget{ID: id, ContentType: entity.ContentTypeBean, ContentID: id, IsActive: true}, nil
}

// content_type/content_idで有効なランキング対象を返す。
func (r *fakeRankTargetRepo) FindByContent(ctx context.Context, contentType entity.ContentType, contentID uint64) (*model.RankTarget, error) {
	return &model.RankTarget{ID: contentID, ContentType: contentType, ContentID: contentID, IsActive: true}, nil
}

// 公開処理系で使うRankTarget作成または取得を成功扱いにする。
func (r *fakeRankTargetRepo) FindOrCreate(ctx context.Context, contentType entity.ContentType, contentID uint64) (*model.RankTarget, error) {
	return &model.RankTarget{ID: contentID, ContentType: contentType, ContentID: contentID, IsActive: true}, nil
}

// 複数IDのRankTarget取得を成功扱いにする。
func (r *fakeRankTargetRepo) FindByIDs(ctx context.Context, ids []uint64) ([]*model.RankTarget, error) {
	targets := make([]*model.RankTarget, 0, len(ids))
	for _, id := range ids {
		targets = append(targets, &model.RankTarget{ID: id, ContentType: entity.ContentTypeBean, ContentID: id, IsActive: true})
	}
	return targets, nil
}

// content_type別の有効RankTarget一覧取得を成功扱いにする。
func (r *fakeRankTargetRepo) ListActiveByType(ctx context.Context, contentType entity.ContentType) ([]*model.RankTarget, error) {
	return []*model.RankTarget{{ID: 1, ContentType: contentType, ContentID: 1, IsActive: true}}, nil
}

// ModalUsecase.Showが候補の有効性を確認するため、trueを返す。
func (r *fakeRankTargetRepo) ExistsActiveByID(ctx context.Context, id uint64) (bool, error) {
	return id > 0, nil
}

// RankTarget有効状態の更新はControllerテストでは対象外なので成功扱いにする。
func (r *fakeRankTargetRepo) UpdateActive(ctx context.Context, id uint64, isActive bool) error {
	return nil
}

type fakeBeanRepo struct{}

// Bean作成はControllerテストでは対象外なのでIDだけ補完して成功扱いにする。
func (r *fakeBeanRepo) Create(ctx context.Context, bean *model.Bean) error {
	if bean != nil && bean.ID == 0 {
		bean.ID = 1
	}
	return nil
}

// Bean更新はControllerテストでは対象外なので成功扱いにする。
func (r *fakeBeanRepo) Update(ctx context.Context, bean *model.Bean) error {
	return nil
}

// 管理用Bean取得を成功扱いにする。
func (r *fakeBeanRepo) FindByID(ctx context.Context, id uint64) (*model.Bean, error) {
	return &model.Bean{ID: id, Name: "Bean", RoastLevel: entity.RoastLevelMedium, IsPublished: true}, nil
}

// 公開Bean詳細取得を成功扱いにする。
func (r *fakeBeanRepo) FindPublishedByID(ctx context.Context, id uint64) (*model.Bean, error) {
	return r.FindByID(ctx, id)
}

// 公開Bean一覧取得を成功扱いにする。
func (r *fakeBeanRepo) ListPublished(ctx context.Context, limit int, offset int) ([]*model.Bean, error) {
	return []*model.Bean{{ID: 1, Name: "Bean", RoastLevel: entity.RoastLevelMedium, IsPublished: true}}, nil
}

// Bean検索を成功扱いにする。
func (r *fakeBeanRepo) SearchPublished(ctx context.Context, filter repository.BeanSearchFilter) ([]*model.Bean, error) {
	return []*model.Bean{{ID: 1, Name: "Bean", RoastLevel: entity.RoastLevelMedium, IsPublished: true}}, nil
}

// RankingUsecase.ListがBean実体を取得できるようにする。
func (r *fakeBeanRepo) FindByIDs(ctx context.Context, ids []uint64) ([]*model.Bean, error) {
	beans := make([]*model.Bean, 0, len(ids))
	for _, id := range ids {
		beans = append(beans, &model.Bean{ID: id, Name: "Bean", RoastLevel: entity.RoastLevelMedium, IsPublished: true})
	}
	return beans, nil
}

// 管理操作用のBean存在確認を成功扱いにする。
func (r *fakeBeanRepo) ExistsByID(ctx context.Context, id uint64) (bool, error) {
	return id > 0, nil
}

// 公開Bean存在確認を成功扱いにする。
func (r *fakeBeanRepo) ExistsPublishedByID(ctx context.Context, id uint64) (bool, error) {
	return id > 0, nil
}

// Bean公開状態更新はControllerテストでは対象外なので成功扱いにする。
func (r *fakeBeanRepo) UpdatePublished(ctx context.Context, id uint64, isPublished bool) error {
	return nil
}

type fakeArticleRepo struct{}

// Article作成はControllerテストでは対象外なのでIDだけ補完して成功扱いにする。
func (r *fakeArticleRepo) Create(ctx context.Context, article *model.Article) error {
	if article != nil && article.ID == 0 {
		article.ID = 1
	}
	return nil
}

// Article更新はControllerテストでは対象外なので成功扱いにする。
func (r *fakeArticleRepo) Update(ctx context.Context, article *model.Article) error {
	return nil
}

// 管理用Article取得を成功扱いにする。
func (r *fakeArticleRepo) FindByID(ctx context.Context, id uint64) (*model.Article, error) {
	return &model.Article{ID: id, Title: "Article", Slug: "article", Summary: "summary", IsPublished: true}, nil
}

// 公開Article詳細取得を成功扱いにする。
func (r *fakeArticleRepo) FindPublishedByID(ctx context.Context, id uint64) (*model.Article, error) {
	return r.FindByID(ctx, id)
}

// slugでArticleを取得する処理を成功扱いにする。
func (r *fakeArticleRepo) FindBySlug(ctx context.Context, slug string) (*model.Article, error) {
	return &model.Article{ID: 1, Title: "Article", Slug: slug, Summary: "summary", IsPublished: true}, nil
}

// 公開slugでArticleを取得する処理を成功扱いにする。
func (r *fakeArticleRepo) FindPublishedBySlug(ctx context.Context, slug string) (*model.Article, error) {
	return r.FindBySlug(ctx, slug)
}

// 公開Article一覧取得を成功扱いにする。
func (r *fakeArticleRepo) ListPublished(ctx context.Context, limit int, offset int) ([]*model.Article, error) {
	return []*model.Article{{ID: 1, Title: "Article", Slug: "article", Summary: "summary", IsPublished: true}}, nil
}

// Article検索を成功扱いにする。
func (r *fakeArticleRepo) SearchPublished(ctx context.Context, filter repository.ArticleSearchFilter) ([]*model.Article, error) {
	return []*model.Article{{ID: 1, Title: "Article", Slug: "article", Summary: "summary", IsPublished: true}}, nil
}

// RankingUsecase.ListがArticle実体を取得できるようにする。
func (r *fakeArticleRepo) FindByIDs(ctx context.Context, ids []uint64) ([]*model.Article, error) {
	articles := make([]*model.Article, 0, len(ids))
	for _, id := range ids {
		articles = append(articles, &model.Article{ID: id, Title: "Article", Slug: "article", Summary: "summary", IsPublished: true})
	}
	return articles, nil
}

// 管理操作用のArticle存在確認を成功扱いにする。
func (r *fakeArticleRepo) ExistsByID(ctx context.Context, id uint64) (bool, error) {
	return id > 0, nil
}

// 公開Article存在確認を成功扱いにする。
func (r *fakeArticleRepo) ExistsPublishedByID(ctx context.Context, id uint64) (bool, error) {
	return id > 0, nil
}

// Article公開状態更新はControllerテストでは対象外なので成功扱いにする。
func (r *fakeArticleRepo) UpdatePublished(ctx context.Context, id uint64, isPublished bool) error {
	return nil
}

type fakeInterestRepo struct{}

// InterestProfile保存はControllerテストでは対象外なので成功扱いにする。
func (r *fakeInterestRepo) Upsert(ctx context.Context, profile *model.InterestProfile) error {
	return nil
}

// InterestProfile一括保存はControllerテストでは対象外なので成功扱いにする。
func (r *fakeInterestRepo) BulkUpsert(ctx context.Context, profiles []*model.InterestProfile) error {
	return nil
}

// Userの興味プロフィール取得を空配列成功扱いにする。
func (r *fakeInterestRepo) FindByUser(ctx context.Context, userID uint64) ([]*model.InterestProfile, error) {
	return []*model.InterestProfile{}, nil
}

// GuestSessionの興味プロフィール取得を空配列成功扱いにする。
func (r *fakeInterestRepo) FindByGuestSession(ctx context.Context, guestSessionID uint64) ([]*model.InterestProfile, error) {
	return []*model.InterestProfile{}, nil
}

// Userの上位興味プロフィール取得を空配列成功扱いにする。
func (r *fakeInterestRepo) ListTopByUser(ctx context.Context, userID uint64, limit int) ([]*model.InterestProfile, error) {
	return []*model.InterestProfile{}, nil
}

// Guestの上位興味プロフィール取得を空配列成功扱いにする。
func (r *fakeInterestRepo) ListTopByGuest(ctx context.Context, guestSessionID uint64, limit int) ([]*model.InterestProfile, error) {
	return []*model.InterestProfile{}, nil
}

// 期限切れInterestProfile削除はControllerテストでは対象外なので0件成功扱いにする。
func (r *fakeInterestRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}

type fakeSavedRepo struct{}

// 保存作成はControllerテストでは対象外なので成功扱いにする。
func (r *fakeSavedRepo) Create(ctx context.Context, item *model.SavedItem) error {
	return nil
}

// User/RankTarget別保存取得を未保存扱いで返す。
func (r *fakeSavedRepo) FindByUserAndTarget(ctx context.Context, userID uint64, rankTargetID uint64) (*model.SavedItem, error) {
	return nil, entity.ErrSavedItemNotFound
}

// 保存または再保存はControllerテストでは対象外なので保存済みmodelを返す。
func (r *fakeSavedRepo) SaveOrRestore(ctx context.Context, userID uint64, rankTargetID uint64, now time.Time) (*model.SavedItem, error) {
	return &model.SavedItem{ID: 1, UserID: userID, RankTargetID: rankTargetID}, nil
}

// 保存解除はControllerテストでは対象外なので成功扱いにする。
func (r *fakeSavedRepo) Remove(ctx context.Context, userID uint64, rankTargetID uint64, removedAt time.Time) error {
	return nil
}

// 保存一覧取得を空配列成功扱いにする。
func (r *fakeSavedRepo) ListActiveByUserID(ctx context.Context, userID uint64, limit int, offset int) ([]*model.SavedItem, error) {
	return []*model.SavedItem{}, nil
}

// 保存済み除外やモーダル表示確認では未保存扱いにする。
func (r *fakeSavedRepo) ExistsActive(ctx context.Context, userID uint64, rankTargetID uint64) (bool, error) {
	return false, nil
}

// 対象ごとの保存数取得は0件成功扱いにする。
func (r *fakeSavedRepo) CountActiveByTarget(ctx context.Context, rankTargetID uint64) (int64, error) {
	return 0, nil
}

type fakeModalDisplayRepo struct{}

// モーダル表示ログを作成し、IDが未設定なら補完する。
func (r *fakeModalDisplayRepo) Create(ctx context.Context, log *model.ModalDisplayLog) error {
	if log != nil && log.ID == 0 {
		log.ID = 1
	}
	return nil
}

// actor本人の表示ログ取得を成功扱いにする。
func (r *fakeModalDisplayRepo) FindByIDForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64) (*model.ModalDisplayLog, error) {
	return &model.ModalDisplayLog{ID: id, UserID: userID, GuestSessionID: guestSessionID, RankTargetID: 1, PagePath: "/beans/1"}, nil
}

// actor条件付きクリック更新を成功扱いにする。
func (r *fakeModalDisplayRepo) MarkClickedForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64, clickedAt time.Time) error {
	return nil
}

// actor条件付きクローズ更新を成功扱いにする。
func (r *fakeModalDisplayRepo) MarkClosedForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64, closedAt time.Time) error {
	return nil
}

// 同一セッションの表示回数を0として返し、表示上限に引っかからないようにする。
func (r *fakeModalDisplayRepo) CountShownInSession(ctx context.Context, userID *uint64, guestSessionID *uint64, since time.Time) (int64, error) {
	return 0, nil
}

// 同一ページの表示回数を0として返し、表示上限に引っかからないようにする。
func (r *fakeModalDisplayRepo) CountShownOnPage(ctx context.Context, userID *uint64, guestSessionID *uint64, pagePath string, since time.Time) (int64, error) {
	return 0, nil
}

// 同じ候補の直近表示はないものとして返す。
func (r *fakeModalDisplayRepo) WasTargetShownRecently(ctx context.Context, userID *uint64, guestSessionID *uint64, rankTargetID uint64, since time.Time) (bool, error) {
	return false, nil
}

// 同じ候補の直近クローズはないものとして返す。
func (r *fakeModalDisplayRepo) WasTargetClosedRecently(ctx context.Context, userID *uint64, guestSessionID *uint64, rankTargetID uint64, since time.Time) (bool, error) {
	return false, nil
}

type fakeModalBlockRepo struct{}

// モーダル非表示ログ作成はControllerテストでは対象外なので成功扱いにする。
func (r *fakeModalBlockRepo) Create(ctx context.Context, log *model.ModalBlockLog) error {
	return nil
}

// actor別非表示ログ一覧は空配列成功扱いにする。
func (r *fakeModalBlockRepo) ListByActor(ctx context.Context, userID *uint64, guestSessionID *uint64, limit int) ([]*model.ModalBlockLog, error) {
	return []*model.ModalBlockLog{}, nil
}

// 理由別非表示件数は0件成功扱いにする。
func (r *fakeModalBlockRepo) CountByReasonSince(ctx context.Context, reason entity.ModalBlockReason, since time.Time) (int64, error) {
	return 0, nil
}

// 候補別非表示ログ一覧は空配列成功扱いにする。
func (r *fakeModalBlockRepo) ListByCandidate(ctx context.Context, rankTargetID uint64, limit int) ([]*model.ModalBlockLog, error) {
	return []*model.ModalBlockLog{}, nil
}

type fakeEventRepo struct{}

// 行動ログ作成は補助記録として成功扱いにする。
func (r *fakeEventRepo) Create(ctx context.Context, event *model.ActionEvent) error {
	return nil
}

// 行動ログ一括作成はControllerテストでは対象外なので成功扱いにする。
func (r *fakeEventRepo) BulkCreate(ctx context.Context, events []*model.ActionEvent) error {
	return nil
}

// actorの最近イベント取得は空配列成功扱いにする。
func (r *fakeEventRepo) FindRecentByActor(ctx context.Context, userID *uint64, guestSessionID *uint64, limit int) ([]*model.ActionEvent, error) {
	return []*model.ActionEvent{}, nil
}

// 最後の検索hashは未存在扱いにする。
func (r *fakeEventRepo) FindLastSearchHash(ctx context.Context, userID *uint64, guestSessionID *uint64) (*string, error) {
	return nil, nil
}

// ランキング集計は空配列成功扱いにする。
func (r *fakeEventRepo) AggregateContentMetrics(ctx context.Context, periodStart time.Time, periodEnd time.Time) ([]repository.ContentMetricAggregate, error) {
	return []repository.ContentMetricAggregate{}, nil
}

// User興味集計は空配列成功扱いにする。
func (r *fakeEventRepo) AggregateUserInterest(ctx context.Context, userID uint64, periodStart time.Time, periodEnd time.Time) ([]repository.InterestAggregate, error) {
	return []repository.InterestAggregate{}, nil
}

// Guest興味集計は空配列成功扱いにする。
func (r *fakeEventRepo) AggregateGuestInterest(ctx context.Context, guestSessionID uint64, periodStart time.Time, periodEnd time.Time) ([]repository.InterestAggregate, error) {
	return []repository.InterestAggregate{}, nil
}

// actor/event_type別件数は0件成功扱いにする。
func (r *fakeEventRepo) CountByActorAndTypeSince(ctx context.Context, userID *uint64, guestSessionID *uint64, eventType entity.EventType, since time.Time) (int64, error) {
	return 0, nil
}

// target/event_type別件数は0件成功扱いにする。
func (r *fakeEventRepo) CountByTargetAndTypeSince(ctx context.Context, rankTargetID uint64, eventType entity.EventType, since time.Time) (int64, error) {
	return 0, nil
}

type fakeSuppressionRepo struct{}

// モーダル表示済み抑制の保存は成功扱いにする。
func (r *fakeSuppressionRepo) SetShown(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error {
	return nil
}

// 同じ候補はまだ表示されていないものとして返す。
func (r *fakeSuppressionRepo) WasShown(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error) {
	return false, nil
}

// ページ表示回数のRedis加算は1回目として返す。
func (r *fakeSuppressionRepo) IncrementPageCount(ctx context.Context, actorKey string, pagePath string, ttl time.Duration) (int64, error) {
	return 1, nil
}

// セッション表示回数のRedis加算は1回目として返す。
func (r *fakeSuppressionRepo) IncrementSessionCount(ctx context.Context, actorKey string, ttl time.Duration) (int64, error) {
	return 1, nil
}

// モーダルを閉じた候補の抑制保存は成功扱いにする。
func (r *fakeSuppressionRepo) SetClosed(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error {
	return nil
}

// 同じ候補は直近で閉じられていないものとして返す。
func (r *fakeSuppressionRepo) WasClosed(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error) {
	return false, nil
}
