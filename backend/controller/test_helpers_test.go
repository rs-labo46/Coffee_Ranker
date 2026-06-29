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

type fakeEventRepo struct {
	created *model.ActionEvent
}

// 行動ログ作成は補助記録として成功扱いにする。
func (r *fakeEventRepo) Create(ctx context.Context, event *model.ActionEvent) error {
	if event != nil && event.ID == 0 {
		event.ID = 1
	}
	r.created = event
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

type fakeUserRepo struct {
	created *model.User
	byID    *model.User
	byEmail *model.User
	exists  bool
}

// Signup時のUser作成をDBなしで成功扱いにし、作成内容を後で検証できるよう保持する。
func (r *fakeUserRepo) Create(ctx context.Context, user *model.User) error {
	if user.ID == 0 {
		user.ID = 1
	}
	r.created = user
	return nil
}

// Meや認証確認で使うUserID検索を成功扱いにする。
func (r *fakeUserRepo) FindByID(ctx context.Context, id uint64) (*model.User, error) {
	if r.byID != nil {
		return r.byID, nil
	}
	return &model.User{ID: id, Name: "User", Email: "user@example.com", PasswordHash: "hashed", Role: entity.UserRoleUser, Status: entity.UserStatusActive}, nil
}

// Login時のemail検索を成功扱いにする。
func (r *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	if r.byEmail != nil {
		return r.byEmail, nil
	}
	return &model.User{ID: 1, Name: "User", Email: email, PasswordHash: "hashed", Role: entity.UserRoleUser, Status: entity.UserStatusActive}, nil
}

// Signup時のemail重複確認結果を返す。
func (r *fakeUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return r.exists, nil
}

// token_version更新はController単体テストでは対象外なので成功扱いにする。
func (r *fakeUserRepo) UpdateTokenVersion(ctx context.Context, userID uint64, tokenVersion int) error {
	return nil
}

// token_version加算はController単体テストでは対象外なので成功扱いにする。
func (r *fakeUserRepo) IncrementTokenVersion(ctx context.Context, userID uint64) error {
	return nil
}

// User状態更新はController単体テストでは対象外なので成功扱いにする。
func (r *fakeUserRepo) UpdateStatus(ctx context.Context, userID uint64, status entity.UserStatus) error {
	return nil
}

// User一覧取得はController単体テストでは対象外なので空配列を返す。
func (r *fakeUserRepo) List(ctx context.Context, limit int, offset int) ([]*model.User, error) {
	return []*model.User{}, nil
}

type fakeRefreshTokenRepo struct {
	created *model.RefreshToken
}

// token_hash検索はController単体テストでは必要最小限の有効Tokenを返す。
func (r *fakeRefreshTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	return &model.RefreshToken{ID: 1, UserID: 1, TokenHash: tokenHash, FamilyID: "family-1", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

// User付きRefreshToken検索を成功扱いにする。
func (r *fakeRefreshTokenRepo) FindByTokenHashWithUser(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	token, err := r.FindByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, err
	}
	token.User = model.User{ID: token.UserID, Name: "User", Email: "user@example.com", Role: entity.UserRoleUser, Status: entity.UserStatusActive}
	return token, nil
}

// Refresh rotation用の行ロック取得を成功扱いにする。
func (r *fakeRefreshTokenRepo) FindByTokenHashWithUserForUpdate(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	return r.FindByTokenHashWithUser(ctx, tokenHash)
}

// 古いRefreshToken使用済み化はController単体テストでは対象外なので成功扱いにする。
func (r *fakeRefreshTokenRepo) MarkUsed(ctx context.Context, id uint64, usedAt time.Time, replacedByTokenID uint64) error {
	return nil
}

// Login/Refresh時のRefreshToken作成を成功扱いにし、IDを補完する。
func (r *fakeRefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	if token.ID == 0 {
		token.ID = 1
	}
	r.created = token
	return nil
}

// RefreshToken単体失効はController単体テストでは対象外なので成功扱いにする。
func (r *fakeRefreshTokenRepo) Revoke(ctx context.Context, id uint64, revokedAt time.Time) error {
	return nil
}

// RefreshToken family失効はController単体テストでは対象外なので成功扱いにする。
func (r *fakeRefreshTokenRepo) RevokeByFamilyID(ctx context.Context, familyID string, revokedAt time.Time) error {
	return nil
}

// User全RefreshToken失効はController単体テストでは対象外なので成功扱いにする。
func (r *fakeRefreshTokenRepo) RevokeByUserID(ctx context.Context, userID uint64, revokedAt time.Time) error {
	return nil
}

// 期限切れRefreshToken削除はController単体テストでは対象外なので0件成功扱いにする。
func (r *fakeRefreshTokenRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}

type fakeAuditRepo struct {
	created *model.AuditLog
}

// 監査ログ作成は補助記録なので成功扱いにし、必要なら内容を検証できるよう保持する。
func (r *fakeAuditRepo) Create(ctx context.Context, log *model.AuditLog) error {
	r.created = log
	return nil
}

// 監査ログ詳細取得を成功扱いにする。
func (r *fakeAuditRepo) FindByID(ctx context.Context, id uint64) (*model.AuditLog, error) {
	return &model.AuditLog{ID: id, ActorType: entity.AuditActorAdmin, Action: entity.AuditActionLogin}, nil
}

// 監査ログ一覧取得を空配列成功扱いにする。
func (r *fakeAuditRepo) List(ctx context.Context, filter repository.AuditLogFilter) ([]*model.AuditLog, error) {
	return []*model.AuditLog{}, nil
}

// actor別監査ログ取得を空配列成功扱いにする。
func (r *fakeAuditRepo) ListByActor(ctx context.Context, actorType entity.AuditActorType, actorUserID *uint64, limit int) ([]*model.AuditLog, error) {
	return []*model.AuditLog{}, nil
}

// target別監査ログ取得を空配列成功扱いにする。
func (r *fakeAuditRepo) ListByTarget(ctx context.Context, targetType string, targetID uint64, limit int) ([]*model.AuditLog, error) {
	return []*model.AuditLog{}, nil
}

// request_id別監査ログ取得を空配列成功扱いにする。
func (r *fakeAuditRepo) ListByRequestID(ctx context.Context, requestID string) ([]*model.AuditLog, error) {
	return []*model.AuditLog{}, nil
}

type fakePasswordHasher struct{}

// Signup時のpassword hash化を固定文字列で成功扱いにする。
func (h *fakePasswordHasher) Hash(ctx context.Context, password string) (string, error) {
	return "hashed", nil
}

// Login時のpassword照合を成功扱いにする。
func (h *fakePasswordHasher) Compare(ctx context.Context, password string, passwordHash string) error {
	return nil
}

type fakeTokenManager struct {
	refreshPlain string
	refreshHash  string
	familyID     string
	accessToken  string
}

// RefreshTokenの生値とDB保存用hashを固定値で発行する。
func (m *fakeTokenManager) NewRefreshToken(ctx context.Context) (string, string, error) {
	plain := m.refreshPlain
	if plain == "" {
		plain = "refresh-plain"
	}
	hash := m.refreshHash
	if hash == "" {
		hash = "refresh-hash"
	}
	return plain, hash, nil
}

// RefreshToken family_idを固定値で発行する。
func (m *fakeTokenManager) NewFamilyID(ctx context.Context) (string, error) {
	if m.familyID != "" {
		return m.familyID, nil
	}
	return "family-1", nil
}

// Cookieから受けたRefreshTokenをDB照合用hashへ変換する。
func (m *fakeTokenManager) HashRefreshToken(ctx context.Context, token string) (string, error) {
	if token == "" {
		return "", entity.ErrInvalidToken
	}
	return "hash:" + token, nil
}

// AccessTokenを固定値で発行する。
func (m *fakeTokenManager) IssueAccessToken(ctx context.Context, user *model.User, now time.Time) (string, error) {
	if m.accessToken != "" {
		return m.accessToken, nil
	}
	return "access-token", nil
}

type fakeGuestSessionRepo struct {
	created *model.GuestSession
}

// GuestSession作成を成功扱いにし、IDを補完する。
func (r *fakeGuestSessionRepo) Create(ctx context.Context, session *model.GuestSession) error {
	if session.ID == 0 {
		session.ID = 1
	}
	r.created = session
	return nil
}

// GuestSession ID検索を有効期限内のSessionとして返す。
func (r *fakeGuestSessionRepo) FindByID(ctx context.Context, id uint64) (*model.GuestSession, error) {
	return &model.GuestSession{ID: id, SessionKeyHash: "guest-hash", FirstSeenAt: time.Now(), LastSeenAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)}, nil
}

// session_key_hash検索は未存在扱いにし、新規作成フローへ進ませる。
func (r *fakeGuestSessionRepo) FindBySessionKeyHash(ctx context.Context, hash string) (*model.GuestSession, error) {
	return nil, entity.ErrNotFound
}

// GuestSessionの最終アクセス更新は成功扱いにする。
func (r *fakeGuestSessionRepo) Touch(ctx context.Context, id uint64, lastSeenAt time.Time, expiresAt time.Time) error {
	return nil
}

// 期限切れGuestSession削除はController単体テストでは対象外なので0件成功扱いにする。
func (r *fakeGuestSessionRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return 0, nil
}

type fakeGuestKeys struct{}

// 新しいGuestSession keyの生値とDB保存用hashを固定値で発行する。
func (k *fakeGuestKeys) NewGuestSessionKey(ctx context.Context) (string, string, error) {
	return "guest-plain", "guest-hash", nil
}

// 既存GuestSession keyをDB照合用hashへ変換する。
func (k *fakeGuestKeys) HashGuestSessionKey(ctx context.Context, key string) (string, error) {
	if key == "" {
		return "", entity.ErrInvalidInput
	}
	return "hash:" + key, nil
}

type fakeBeanArticleRepo struct{}

// BeanとArticleの関連作成はController単体テストでは対象外なので成功扱いにする。
func (r *fakeBeanArticleRepo) Create(ctx context.Context, relation *model.BeanArticle) error {
	return nil
}

// BeanとArticleの関連削除はController単体テストでは対象外なので成功扱いにする。
func (r *fakeBeanArticleRepo) Delete(ctx context.Context, beanID uint64, articleID uint64) error {
	return nil
}

// 関連存在確認は未存在扱いにする。
func (r *fakeBeanArticleRepo) Exists(ctx context.Context, beanID uint64, articleID uint64) (bool, error) {
	return false, nil
}

// Beanに紐づくArticle関連一覧を空配列で返す。
func (r *fakeBeanArticleRepo) ListByBeanID(ctx context.Context, beanID uint64, limit int) ([]*model.BeanArticle, error) {
	return []*model.BeanArticle{}, nil
}

// Articleに紐づくBean関連一覧を空配列で返す。
func (r *fakeBeanArticleRepo) ListByArticleID(ctx context.Context, articleID uint64, limit int) ([]*model.BeanArticle, error) {
	return []*model.BeanArticle{}, nil
}

// Beanに紐づくArticle関連の一括差し替えは成功扱いにする。
func (r *fakeBeanArticleRepo) ReplaceByBeanID(ctx context.Context, beanID uint64, articleIDs []uint64) error {
	return nil
}

type fakeDedupRepo struct{}

// 重複防止keyが未存在として扱われ、イベント保存へ進むようtrueを返す。
func (r *fakeDedupRepo) SetIfNotExists(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	return true, nil
}

// 重複防止keyの存在確認は未存在扱いにする。
func (r *fakeDedupRepo) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

// 重複防止key削除は成功扱いにする。
func (r *fakeDedupRepo) Delete(ctx context.Context, key string) error {
	return nil
}

type fakeRatingRepo struct{}

// 評価作成はController単体テストでは対象外なので成功扱いにする。
func (r *fakeRatingRepo) Create(ctx context.Context, rating *model.Rating) error {
	return nil
}

// User/RankTarget別評価取得は未評価扱いにする。
func (r *fakeRatingRepo) FindByUserAndTarget(ctx context.Context, userID uint64, rankTargetID uint64) (*model.Rating, error) {
	return nil, entity.ErrRatingNotFound
}

// 評価登録または更新を成功扱いにし、評価modelを返す。
func (r *fakeRatingRepo) Upsert(ctx context.Context, userID uint64, rankTargetID uint64, score entity.RatingScore, now time.Time) (*model.Rating, error) {
	return &model.Rating{ID: 1, UserID: userID, RankTargetID: rankTargetID, Score: score}, nil
}

// 評価削除は成功扱いにする。
func (r *fakeRatingRepo) Delete(ctx context.Context, userID uint64, rankTargetID uint64) error {
	return nil
}

// 対象ごとの評価数取得は0件成功扱いにする。
func (r *fakeRatingRepo) CountByTarget(ctx context.Context, rankTargetID uint64) (int64, error) {
	return 0, nil
}

// 評価集計は0件の集計結果として返す。
func (r *fakeRatingRepo) AggregateByTarget(ctx context.Context, rankTargetID uint64) (repository.RatingAggregate, error) {
	return repository.RatingAggregate{}, nil
}

// request_idとIP hashをContextへ設定する。
// Controllerが監査・イベント用メタ情報をbodyではなくContextから取ることを確認するために使う。
func setMeta(c echo.Context) {
	c.Set(ContextRequestIDKey, "req-1")
	c.Set(ContextIPHashKey, "ip-hash-1")
}
