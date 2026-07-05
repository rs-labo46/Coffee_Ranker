package usecase

// このファイルは、Usecaseの単体テストで使うfake実装をまとめたもの。
// 実DBやRedisを使わず、Repositoryの戻り値・呼び出し状況だけを確認できるようにする。

import (
	"context"
	"errors"
	"testing"
	"time"

	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
)

// 期待した業務エラーが返っているか確認するテスト補助関数。
func assertErrorIs(t *testing.T, got error, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("error = %v, want %v", got, want)
	}
}

// エラーが返っていないことを確認するテスト補助関数。
func assertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakePasswordHasher struct {
	hash    string
	hashErr error
	cmpErr  error
	lastRaw string
}

func (f *fakePasswordHasher) Hash(ctx context.Context, password string) (string, error) {
	f.lastRaw = password
	if f.hashErr != nil {
		return "", f.hashErr
	}
	if f.hash != "" {
		return f.hash, nil
	}
	return "hashed:" + password, nil
}

func (f *fakePasswordHasher) Compare(ctx context.Context, password string, passwordHash string) error {
	f.lastRaw = password
	return f.cmpErr
}

type fakeTokenManager struct {
	refreshPlain string
	refreshHash  string
	refreshErr   error
	familyID     string
	familyErr    error
	accessToken  string
	accessErr    error
	hashValue    string
	hashErr      error
}

func (f *fakeTokenManager) NewRefreshToken(ctx context.Context) (string, string, error) {
	if f.refreshErr != nil {
		return "", "", f.refreshErr
	}
	plain := f.refreshPlain
	if plain == "" {
		plain = "plain-refresh"
	}
	hash := f.refreshHash
	if hash == "" {
		hash = "hash-refresh"
	}
	return plain, hash, nil
}

func (f *fakeTokenManager) NewFamilyID(ctx context.Context) (string, error) {
	if f.familyErr != nil {
		return "", f.familyErr
	}
	if f.familyID != "" {
		return f.familyID, nil
	}
	return "family-1", nil
}

func (f *fakeTokenManager) HashRefreshToken(ctx context.Context, token string) (string, error) {
	if f.hashErr != nil {
		return "", f.hashErr
	}
	if f.hashValue != "" {
		return f.hashValue, nil
	}
	return "hash:" + token, nil
}

func (f *fakeTokenManager) IssueAccessToken(ctx context.Context, user *model.User, now time.Time) (string, error) {
	if f.accessErr != nil {
		return "", f.accessErr
	}
	if f.accessToken != "" {
		return f.accessToken, nil
	}
	return "access-token", nil
}

type fakeGuestKeys struct {
	plain   string
	hash    string
	newErr  error
	hashErr error
}

func (f *fakeGuestKeys) NewGuestSessionKey(ctx context.Context) (string, string, error) {
	if f.newErr != nil {
		return "", "", f.newErr
	}
	plain := f.plain
	if plain == "" {
		plain = "guest-plain"
	}
	hash := f.hash
	if hash == "" {
		hash = "guest-hash"
	}
	return plain, hash, nil
}

func (f *fakeGuestKeys) HashGuestSessionKey(ctx context.Context, key string) (string, error) {
	if f.hashErr != nil {
		return "", f.hashErr
	}
	if f.hash != "" {
		return f.hash, nil
	}
	return "hash:" + key, nil
}

type fakeUserRepo struct {
	created        *model.User
	createErr      error
	byID           *model.User
	findByIDErr    error
	byEmail        *model.User
	findEmailErr   error
	existsEmail    bool
	existsEmailErr error
	incrementedID  uint64
	incrementErr   error
}

func (f *fakeUserRepo) Create(ctx context.Context, user *model.User) error {
	f.created = user
	if user.ID == 0 {
		user.ID = 1
	}
	return f.createErr
}
func (f *fakeUserRepo) FindByID(ctx context.Context, id uint64) (*model.User, error) {
	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}
	if f.byID != nil {
		return f.byID, nil
	}
	return &model.User{ID: id, Status: entity.UserStatusActive, Role: entity.UserRoleUser}, nil
}
func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	if f.findEmailErr != nil {
		return nil, f.findEmailErr
	}
	if f.byEmail != nil {
		return f.byEmail, nil
	}
	return &model.User{ID: 1, Email: email, PasswordHash: "hash", Status: entity.UserStatusActive}, nil
}
func (f *fakeUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return f.existsEmail, f.existsEmailErr
}
func (f *fakeUserRepo) UpdateTokenVersion(ctx context.Context, userID uint64, tokenVersion int) error {
	return nil
}
func (f *fakeUserRepo) IncrementTokenVersion(ctx context.Context, userID uint64) error {
	f.incrementedID = userID
	return f.incrementErr
}
func (f *fakeUserRepo) UpdateStatus(ctx context.Context, userID uint64, status entity.UserStatus) error {
	return nil
}
func (f *fakeUserRepo) List(ctx context.Context, limit int, offset int) ([]*model.User, error) {
	return nil, nil
}

type fakeRefreshTokenRepo struct {
	created           []*model.RefreshToken
	createErr         error
	byHash            *model.RefreshToken
	byHashErr         error
	byHashUser        *model.RefreshToken
	byHashUserErr     error
	forUpdate         *model.RefreshToken
	forUpdateErr      error
	markedUsedID      uint64
	markUsedErr       error
	revokedFamily     string
	revokeByFamilyErr error
	revokedUserID     uint64
	revokeUserErr     error
	deleteCount       int64
	deleteErr         error
}

func (f *fakeRefreshTokenRepo) FindByTokenHash(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	if f.byHashErr != nil {
		return nil, f.byHashErr
	}
	if f.byHash != nil {
		return f.byHash, nil
	}
	return nil, entity.ErrNotFound
}
func (f *fakeRefreshTokenRepo) FindByTokenHashWithUser(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	if f.byHashUserErr != nil {
		return nil, f.byHashUserErr
	}
	if f.byHashUser != nil {
		return f.byHashUser, nil
	}
	return nil, entity.ErrNotFound
}
func (f *fakeRefreshTokenRepo) FindByTokenHashWithUserForUpdate(ctx context.Context, tokenHash string) (*model.RefreshToken, error) {
	if f.forUpdateErr != nil {
		return nil, f.forUpdateErr
	}
	if f.forUpdate != nil {
		return f.forUpdate, nil
	}
	return nil, entity.ErrNotFound
}
func (f *fakeRefreshTokenRepo) MarkUsed(ctx context.Context, id uint64, usedAt time.Time, replacedByTokenID uint64) error {
	f.markedUsedID = id
	return f.markUsedErr
}
func (f *fakeRefreshTokenRepo) Create(ctx context.Context, token *model.RefreshToken) error {
	f.created = append(f.created, token)
	if token.ID == 0 {
		token.ID = uint64(len(f.created))
	}
	return f.createErr
}
func (f *fakeRefreshTokenRepo) Revoke(ctx context.Context, id uint64, revokedAt time.Time) error {
	return nil
}
func (f *fakeRefreshTokenRepo) RevokeByFamilyID(ctx context.Context, familyID string, revokedAt time.Time) error {
	f.revokedFamily = familyID
	return f.revokeByFamilyErr
}
func (f *fakeRefreshTokenRepo) RevokeByUserID(ctx context.Context, userID uint64, revokedAt time.Time) error {
	f.revokedUserID = userID
	return f.revokeUserErr
}
func (f *fakeRefreshTokenRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return f.deleteCount, f.deleteErr
}

type fakeGuestSessionRepo struct {
	created     *model.GuestSession
	createErr   error
	byID        *model.GuestSession
	byIDErr     error
	byHash      *model.GuestSession
	byHashErr   error
	touchedID   uint64
	touchErr    error
	deleteCount int64
	deleteErr   error
}

func (f *fakeGuestSessionRepo) Create(ctx context.Context, session *model.GuestSession) error {
	f.created = session
	if session.ID == 0 {
		session.ID = 1
	}
	return f.createErr
}
func (f *fakeGuestSessionRepo) FindByID(ctx context.Context, id uint64) (*model.GuestSession, error) {
	if f.byIDErr != nil {
		return nil, f.byIDErr
	}
	if f.byID != nil {
		return f.byID, nil
	}
	return nil, entity.ErrNotFound
}
func (f *fakeGuestSessionRepo) FindBySessionKeyHash(ctx context.Context, hash string) (*model.GuestSession, error) {
	if f.byHashErr != nil {
		return nil, f.byHashErr
	}
	if f.byHash != nil {
		return f.byHash, nil
	}
	return nil, entity.ErrNotFound
}
func (f *fakeGuestSessionRepo) Touch(ctx context.Context, id uint64, lastSeenAt time.Time, expiresAt time.Time) error {
	f.touchedID = id
	return f.touchErr
}
func (f *fakeGuestSessionRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return f.deleteCount, f.deleteErr
}

type fakeAuditRepo struct {
	created    []*model.AuditLog
	createErr  error
	byID       *model.AuditLog
	findErr    error
	listed     []*model.AuditLog
	lastFilter repository.AuditLogFilter
	listErr    error
	byRequest  []*model.AuditLog
}

func (f *fakeAuditRepo) Create(ctx context.Context, log *model.AuditLog) error {
	f.created = append(f.created, log)
	return f.createErr
}
func (f *fakeAuditRepo) FindByID(ctx context.Context, id uint64) (*model.AuditLog, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.byID != nil {
		return f.byID, nil
	}
	return &model.AuditLog{ID: id}, nil
}
func (f *fakeAuditRepo) List(ctx context.Context, filter repository.AuditLogFilter) ([]*model.AuditLog, error) {
	f.lastFilter = filter
	return f.listed, f.listErr
}
func (f *fakeAuditRepo) ListByActor(ctx context.Context, actorType entity.AuditActorType, actorUserID *uint64, limit int) ([]*model.AuditLog, error) {
	return nil, nil
}
func (f *fakeAuditRepo) ListByTarget(ctx context.Context, targetType string, targetID uint64, limit int) ([]*model.AuditLog, error) {
	return nil, nil
}
func (f *fakeAuditRepo) ListByRequestID(ctx context.Context, requestID string) ([]*model.AuditLog, error) {
	return f.byRequest, nil
}

type fakeBeanRepo struct {
	beans                []*model.Bean
	created              *model.Bean
	updated              *model.Bean
	byID                 *model.Bean
	byIDs                map[uint64]*model.Bean
	findErr              error
	published            *model.Bean
	publishedErr         error
	listLimit            int
	listOffset           int
	searchFilter         repository.BeanSearchFilter
	ids                  []uint64
	exists               bool
	existsErr            error
	publishedExists      bool
	updatePublishedID    uint64
	updatePublishedValue bool
	updatePublishedErr   error
}

func (f *fakeBeanRepo) Create(ctx context.Context, bean *model.Bean) error {
	f.created = bean
	if bean.ID == 0 {
		bean.ID = 1
	}
	return nil
}
func (f *fakeBeanRepo) Update(ctx context.Context, bean *model.Bean) error {
	f.updated = bean
	return nil
}
func (f *fakeBeanRepo) FindByID(ctx context.Context, id uint64) (*model.Bean, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.byID != nil {
		return f.byID, nil
	}
	return &model.Bean{ID: id}, nil
}
func (f *fakeBeanRepo) FindPublishedByID(ctx context.Context, id uint64) (*model.Bean, error) {
	if f.publishedErr != nil {
		return nil, f.publishedErr
	}
	if f.published != nil {
		return f.published, nil
	}
	return &model.Bean{ID: id, IsPublished: true}, nil
}
func (f *fakeBeanRepo) ListAll(ctx context.Context, limit int, offset int) ([]*model.Bean, error) {
	return f.beans, nil
}

func (f *fakeBeanRepo) ListPublished(ctx context.Context, limit int, offset int) ([]*model.Bean, error) {
	f.listLimit = limit
	f.listOffset = offset
	return []*model.Bean{{ID: 1}}, nil
}
func (f *fakeBeanRepo) SearchPublished(ctx context.Context, filter repository.BeanSearchFilter) ([]*model.Bean, error) {
	f.searchFilter = filter
	return []*model.Bean{{ID: 1}}, nil
}
func (f *fakeBeanRepo) FindByIDs(ctx context.Context, ids []uint64) ([]*model.Bean, error) {
	// RecommendationUsecaseの興味プロフィール照合を検証できるよう、
	// テストごとに指定した属性付きBeanを優先して返す。
	f.ids = ids
	beans := make([]*model.Bean, 0, len(ids))
	for _, id := range ids {
		if f.byIDs != nil {
			if bean, ok := f.byIDs[id]; ok {
				beans = append(beans, bean)
				continue
			}
		}
		beans = append(beans, &model.Bean{ID: id})
	}
	return beans, nil
}
func (f *fakeBeanRepo) ExistsByID(ctx context.Context, id uint64) (bool, error) {
	return f.exists, f.existsErr
}
func (f *fakeBeanRepo) ExistsPublishedByID(ctx context.Context, id uint64) (bool, error) {
	return f.publishedExists, nil
}
func (f *fakeBeanRepo) UpdatePublished(ctx context.Context, id uint64, isPublished bool) error {
	f.updatePublishedID = id
	f.updatePublishedValue = isPublished
	return f.updatePublishedErr
}

type fakeArticleRepo struct {
	articles             []*model.Article
	created              *model.Article
	updated              *model.Article
	byID                 *model.Article
	byIDs                map[uint64]*model.Article
	findErr              error
	published            *model.Article
	publishedErr         error
	publishedSlug        *model.Article
	publishedSlugErr     error
	listLimit            int
	listOffset           int
	searchFilter         repository.ArticleSearchFilter
	ids                  []uint64
	exists               bool
	existsErr            error
	publishedExists      bool
	updatePublishedID    uint64
	updatePublishedValue bool
	updatePublishedErr   error
}

func (f *fakeArticleRepo) Create(ctx context.Context, article *model.Article) error {
	f.created = article
	if article.ID == 0 {
		article.ID = 1
	}
	return nil
}
func (f *fakeArticleRepo) Update(ctx context.Context, article *model.Article) error {
	f.updated = article
	return nil
}
func (f *fakeArticleRepo) FindByID(ctx context.Context, id uint64) (*model.Article, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	if f.byID != nil {
		return f.byID, nil
	}
	return &model.Article{ID: id}, nil
}
func (f *fakeArticleRepo) FindPublishedByID(ctx context.Context, id uint64) (*model.Article, error) {
	if f.publishedErr != nil {
		return nil, f.publishedErr
	}
	if f.published != nil {
		return f.published, nil
	}
	return &model.Article{ID: id, IsPublished: true}, nil
}
func (f *fakeArticleRepo) FindBySlug(ctx context.Context, slug string) (*model.Article, error) {
	return &model.Article{ID: 1, Slug: slug}, nil
}
func (f *fakeArticleRepo) FindPublishedBySlug(ctx context.Context, slug string) (*model.Article, error) {
	if f.publishedSlugErr != nil {
		return nil, f.publishedSlugErr
	}
	if f.publishedSlug != nil {
		return f.publishedSlug, nil
	}
	return &model.Article{ID: 1, Slug: slug, IsPublished: true}, nil
}
func (f *fakeArticleRepo) ListAll(ctx context.Context, limit int, offset int) ([]*model.Article, error) {
	return f.articles, nil
}

func (f *fakeArticleRepo) ListPublished(ctx context.Context, limit int, offset int) ([]*model.Article, error) {
	f.listLimit = limit
	f.listOffset = offset
	return []*model.Article{{ID: 1}}, nil
}
func (f *fakeArticleRepo) SearchPublished(ctx context.Context, filter repository.ArticleSearchFilter) ([]*model.Article, error) {
	f.searchFilter = filter
	return []*model.Article{{ID: 1}}, nil
}
func (f *fakeArticleRepo) FindByIDs(ctx context.Context, ids []uint64) ([]*model.Article, error) {
	// RecommendationUsecaseのカテゴリ照合を検証できるよう、
	// テストごとに指定した属性付きArticleを優先して返す。
	f.ids = ids
	articles := make([]*model.Article, 0, len(ids))
	for _, id := range ids {
		if f.byIDs != nil {
			if article, ok := f.byIDs[id]; ok {
				articles = append(articles, article)
				continue
			}
		}
		articles = append(articles, &model.Article{ID: id})
	}
	return articles, nil
}
func (f *fakeArticleRepo) ExistsByID(ctx context.Context, id uint64) (bool, error) {
	return f.exists, f.existsErr
}
func (f *fakeArticleRepo) ExistsPublishedByID(ctx context.Context, id uint64) (bool, error) {
	return f.publishedExists, nil
}
func (f *fakeArticleRepo) UpdatePublished(ctx context.Context, id uint64, isPublished bool) error {
	f.updatePublishedID = id
	f.updatePublishedValue = isPublished
	return f.updatePublishedErr
}

type fakeBeanArticleRepo struct {
	created           *model.BeanArticle
	deleteBeanID      uint64
	deleteArticleID   uint64
	exists            bool
	existsErr         error
	replaceBeanID     uint64
	replaceArticleIDs []uint64
	replaceErr        error
}

func (f *fakeBeanArticleRepo) Create(ctx context.Context, relation *model.BeanArticle) error {
	f.created = relation
	if relation.ID == 0 {
		relation.ID = 1
	}
	return nil
}
func (f *fakeBeanArticleRepo) Delete(ctx context.Context, beanID uint64, articleID uint64) error {
	f.deleteBeanID = beanID
	f.deleteArticleID = articleID
	return nil
}
func (f *fakeBeanArticleRepo) Exists(ctx context.Context, beanID uint64, articleID uint64) (bool, error) {
	return f.exists, f.existsErr
}
func (f *fakeBeanArticleRepo) ListByBeanID(ctx context.Context, beanID uint64, limit int) ([]*model.BeanArticle, error) {
	return []*model.BeanArticle{{BeanID: beanID, ArticleID: 1}}, nil
}
func (f *fakeBeanArticleRepo) ListByArticleID(ctx context.Context, articleID uint64, limit int) ([]*model.BeanArticle, error) {
	return []*model.BeanArticle{{BeanID: 1, ArticleID: articleID}}, nil
}
func (f *fakeBeanArticleRepo) ReplaceByBeanID(ctx context.Context, beanID uint64, articleIDs []uint64) error {
	f.replaceBeanID = beanID
	f.replaceArticleIDs = append([]uint64(nil), articleIDs...)
	return f.replaceErr
}

type fakeRankTargetRepo struct {
	existsActive        bool
	existsActiveErr     error
	findOrCreateTarget  *model.RankTarget
	findOrCreateErr     error
	findByContentTarget *model.RankTarget
	findByContentErr    error
	updatedID           uint64
	updatedActive       bool
	updateErr           error
}

func (f *fakeRankTargetRepo) Create(ctx context.Context, target *model.RankTarget) error {
	if target.ID == 0 {
		target.ID = 1
	}
	return nil
}
func (f *fakeRankTargetRepo) FindByID(ctx context.Context, id uint64) (*model.RankTarget, error) {
	return &model.RankTarget{ID: id}, nil
}
func (f *fakeRankTargetRepo) FindByContent(ctx context.Context, contentType entity.ContentType, contentID uint64) (*model.RankTarget, error) {
	if f.findByContentErr != nil {
		return nil, f.findByContentErr
	}
	if f.findByContentTarget != nil {
		return f.findByContentTarget, nil
	}
	return &model.RankTarget{ID: 10, ContentType: contentType, ContentID: contentID}, nil
}
func (f *fakeRankTargetRepo) FindOrCreate(ctx context.Context, contentType entity.ContentType, contentID uint64) (*model.RankTarget, error) {
	if f.findOrCreateErr != nil {
		return nil, f.findOrCreateErr
	}
	if f.findOrCreateTarget != nil {
		return f.findOrCreateTarget, nil
	}
	return &model.RankTarget{ID: 10, ContentType: contentType, ContentID: contentID, IsActive: true}, nil
}
func (f *fakeRankTargetRepo) FindByIDs(ctx context.Context, ids []uint64) ([]*model.RankTarget, error) {
	return nil, nil
}
func (f *fakeRankTargetRepo) ListActiveByType(ctx context.Context, contentType entity.ContentType) ([]*model.RankTarget, error) {
	return nil, nil
}
func (f *fakeRankTargetRepo) ExistsActiveByID(ctx context.Context, id uint64) (bool, error) {
	return f.existsActive, f.existsActiveErr
}
func (f *fakeRankTargetRepo) UpdateActive(ctx context.Context, id uint64, isActive bool) error {
	f.updatedID = id
	f.updatedActive = isActive
	return f.updateErr
}

type fakeActionEventRepo struct {
	created           []*model.ActionEvent
	createErr         error
	lastSearch        *string
	lastSearchErr     error
	recent            []*model.ActionEvent
	contentAggregates []repository.ContentMetricAggregate
	userInterest      []repository.InterestAggregate
	guestInterest     []repository.InterestAggregate
}

func (f *fakeActionEventRepo) Create(ctx context.Context, event *model.ActionEvent) error {
	f.created = append(f.created, event)
	return f.createErr
}
func (f *fakeActionEventRepo) BulkCreate(ctx context.Context, events []*model.ActionEvent) error {
	f.created = append(f.created, events...)
	return nil
}
func (f *fakeActionEventRepo) FindRecentByActor(ctx context.Context, userID *uint64, guestSessionID *uint64, limit int) ([]*model.ActionEvent, error) {
	return f.recent, nil
}
func (f *fakeActionEventRepo) FindLastSearchHash(ctx context.Context, userID *uint64, guestSessionID *uint64) (*string, error) {
	return f.lastSearch, f.lastSearchErr
}
func (f *fakeActionEventRepo) AggregateContentMetrics(ctx context.Context, periodStart time.Time, periodEnd time.Time) ([]repository.ContentMetricAggregate, error) {
	return f.contentAggregates, nil
}
func (f *fakeActionEventRepo) AggregateUserInterest(ctx context.Context, userID uint64, periodStart time.Time, periodEnd time.Time) ([]repository.InterestAggregate, error) {
	return f.userInterest, nil
}
func (f *fakeActionEventRepo) AggregateGuestInterest(ctx context.Context, guestSessionID uint64, periodStart time.Time, periodEnd time.Time) ([]repository.InterestAggregate, error) {
	return f.guestInterest, nil
}
func (f *fakeActionEventRepo) CountByActorAndTypeSince(ctx context.Context, userID *uint64, guestSessionID *uint64, eventType entity.EventType, since time.Time) (int64, error) {
	return 0, nil
}
func (f *fakeActionEventRepo) CountByTargetAndTypeSince(ctx context.Context, rankTargetID uint64, eventType entity.EventType, since time.Time) (int64, error) {
	return 0, nil
}

type fakeDedupRepo struct {
	setOK  bool
	setErr error
	called bool
}

func (f *fakeDedupRepo) SetIfNotExists(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	f.called = true
	return f.setOK, f.setErr
}
func (f *fakeDedupRepo) Exists(ctx context.Context, key string) (bool, error) { return false, nil }
func (f *fakeDedupRepo) Delete(ctx context.Context, key string) error         { return nil }

type fakeSavedRepo struct {
	saved             bool
	savedIDs          map[uint64]bool
	saveOrRestoreItem *model.SavedItem
	saveOrRestoreErr  error
	removed           bool
	listLimit         int
	listOffset        int
}

func (f *fakeSavedRepo) Create(ctx context.Context, item *model.SavedItem) error { return nil }
func (f *fakeSavedRepo) FindByUserAndTarget(ctx context.Context, userID uint64, rankTargetID uint64) (*model.SavedItem, error) {
	return &model.SavedItem{UserID: userID, RankTargetID: rankTargetID}, nil
}
func (f *fakeSavedRepo) SaveOrRestore(ctx context.Context, userID uint64, rankTargetID uint64, now time.Time) (*model.SavedItem, error) {
	if f.saveOrRestoreErr != nil {
		return nil, f.saveOrRestoreErr
	}
	if f.saveOrRestoreItem != nil {
		return f.saveOrRestoreItem, nil
	}
	return &model.SavedItem{UserID: userID, RankTargetID: rankTargetID}, nil
}
func (f *fakeSavedRepo) Remove(ctx context.Context, userID uint64, rankTargetID uint64, removedAt time.Time) error {
	f.removed = true
	return nil
}
func (f *fakeSavedRepo) ListActiveByUserID(ctx context.Context, userID uint64, limit int, offset int) ([]*model.SavedItem, error) {
	f.listLimit = limit
	f.listOffset = offset
	return []*model.SavedItem{{UserID: userID}}, nil
}
func (f *fakeSavedRepo) ListActiveRankTargetIDsByUser(ctx context.Context, userID uint64, rankTargetIDs []uint64) (map[uint64]bool, error) {
	if f.savedIDs != nil {
		return f.savedIDs, nil
	}
	ids := make(map[uint64]bool)
	if f.saved {
		for _, id := range rankTargetIDs {
			ids[id] = true
		}
	}
	return ids, nil
}
func (f *fakeSavedRepo) ExistsActive(ctx context.Context, userID uint64, rankTargetID uint64) (bool, error) {
	return f.saved, nil
}
func (f *fakeSavedRepo) CountActiveByTarget(ctx context.Context, rankTargetID uint64) (int64, error) {
	return 0, nil
}

type fakeRatingRepo struct {
	upserted  *model.Rating
	upsertErr error
	deleted   bool
}

func (f *fakeRatingRepo) Create(ctx context.Context, rating *model.Rating) error { return nil }
func (f *fakeRatingRepo) FindByUserAndTarget(ctx context.Context, userID uint64, rankTargetID uint64) (*model.Rating, error) {
	return &model.Rating{UserID: userID, RankTargetID: rankTargetID}, nil
}
func (f *fakeRatingRepo) ListByUserID(ctx context.Context, userID uint64, limit int, offset int) ([]*model.Rating, error) {
	return []*model.Rating{}, nil
}
func (f *fakeRatingRepo) Upsert(ctx context.Context, userID uint64, rankTargetID uint64, score entity.RatingScore, now time.Time) (*model.Rating, error) {
	if f.upsertErr != nil {
		return nil, f.upsertErr
	}
	f.upserted = &model.Rating{UserID: userID, RankTargetID: rankTargetID, Score: score}
	return f.upserted, nil
}
func (f *fakeRatingRepo) Delete(ctx context.Context, userID uint64, rankTargetID uint64) error {
	f.deleted = true
	return nil
}
func (f *fakeRatingRepo) CountByTarget(ctx context.Context, rankTargetID uint64) (int64, error) {
	return 0, nil
}
func (f *fakeRatingRepo) AggregateByTarget(ctx context.Context, rankTargetID uint64) (repository.RatingAggregate, error) {
	return repository.RatingAggregate{}, nil
}

var _ repository.IRatingRepository = (*fakeRatingRepo)(nil)

type fakeContentMetricRepo struct {
	bulk       []*model.ContentMetric
	bulkErr    error
	ranking    []*model.ContentMetric
	listLimit  int
	listOffset int
	top        []*model.ContentMetric
}

func (f *fakeContentMetricRepo) Upsert(ctx context.Context, metric *model.ContentMetric) error {
	return nil
}
func (f *fakeContentMetricRepo) BulkUpsert(ctx context.Context, metrics []*model.ContentMetric) error {
	f.bulk = metrics
	return f.bulkErr
}
func (f *fakeContentMetricRepo) FindByRankTargetID(ctx context.Context, rankTargetID uint64) (*model.ContentMetric, error) {
	return nil, nil
}
func (f *fakeContentMetricRepo) FindByRankTargetIDs(ctx context.Context, rankTargetIDs []uint64) ([]*model.ContentMetric, error) {
	return nil, nil
}
func (f *fakeContentMetricRepo) ListRanking(ctx context.Context, contentType *entity.ContentType, limit int, offset int) ([]*model.ContentMetric, error) {
	f.listLimit = limit
	f.listOffset = offset
	return f.ranking, nil
}
func (f *fakeContentMetricRepo) ListTopByScore(ctx context.Context, limit int) ([]*model.ContentMetric, error) {
	f.listLimit = limit
	return f.top, nil
}
func (f *fakeContentMetricRepo) LatestCalculatedAt(ctx context.Context) (*time.Time, error) {
	return nil, nil
}

type fakeInterestRepo struct {
	bulk        []*model.InterestProfile
	topUser     []*model.InterestProfile
	topGuest    []*model.InterestProfile
	deleteCount int64
	deleteErr   error
}

func (f *fakeInterestRepo) Upsert(ctx context.Context, profile *model.InterestProfile) error {
	return nil
}
func (f *fakeInterestRepo) BulkUpsert(ctx context.Context, profiles []*model.InterestProfile) error {
	f.bulk = profiles
	return nil
}
func (f *fakeInterestRepo) FindByUser(ctx context.Context, userID uint64) ([]*model.InterestProfile, error) {
	return f.topUser, nil
}
func (f *fakeInterestRepo) FindByGuestSession(ctx context.Context, guestSessionID uint64) ([]*model.InterestProfile, error) {
	return f.topGuest, nil
}
func (f *fakeInterestRepo) ListTopByUser(ctx context.Context, userID uint64, limit int) ([]*model.InterestProfile, error) {
	// RecommendationUsecaseがUserの興味プロフィールを本当に取得しているかをテストで確認する。
	return f.topUser, nil
}
func (f *fakeInterestRepo) ListTopByGuest(ctx context.Context, guestSessionID uint64, limit int) ([]*model.InterestProfile, error) {
	// RecommendationUsecaseがGuestの興味プロフィールを本当に取得しているかをテストで確認する。
	return f.topGuest, nil
}
func (f *fakeInterestRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) {
	return f.deleteCount, f.deleteErr
}

type fakeBatchRunRepo struct {
	created       []*model.BatchRun
	markSuccessID uint64
	markFailedID  uint64
	listLimit     int
	listOffset    int
	latest        *model.BatchRun
}

func (f *fakeBatchRunRepo) Create(ctx context.Context, run *model.BatchRun) error {
	if run.ID == 0 {
		run.ID = uint64(len(f.created) + 1)
	}
	f.created = append(f.created, run)
	return nil
}
func (f *fakeBatchRunRepo) MarkSuccess(ctx context.Context, id uint64, finishedAt time.Time, rowsProcessed int64) error {
	f.markSuccessID = id
	return nil
}
func (f *fakeBatchRunRepo) MarkFailed(ctx context.Context, id uint64, finishedAt time.Time, rowsProcessed int64, errorMessage string) error {
	f.markFailedID = id
	return nil
}
func (f *fakeBatchRunRepo) FindLatestByJobName(ctx context.Context, jobName string) (*model.BatchRun, error) {
	if f.latest != nil {
		return f.latest, nil
	}
	return &model.BatchRun{ID: 1, JobName: jobName}, nil
}
func (f *fakeBatchRunRepo) FindRunningByJobName(ctx context.Context, jobName string) (*model.BatchRun, error) {
	return nil, entity.ErrNotFound
}
func (f *fakeBatchRunRepo) List(ctx context.Context, limit int, offset int) ([]*model.BatchRun, error) {
	f.listLimit = limit
	f.listOffset = offset
	return []*model.BatchRun{{ID: 1}}, nil
}

type fakeRateLimitRepo struct {
	result   repository.RateLimitResult
	err      error
	resetKey string
	resetErr error
	takeKey  string
}

func (f *fakeRateLimitRepo) Take(ctx context.Context, key string, capacity int, refillRate float64, now time.Time) (repository.RateLimitResult, error) {
	f.takeKey = key
	return f.result, f.err
}
func (f *fakeRateLimitRepo) Reset(ctx context.Context, key string) error {
	f.resetKey = key
	return f.resetErr
}

type fakeBatchLockRepo struct {
	locked     bool
	acquireErr error
	released   bool
}

func (f *fakeBatchLockRepo) Acquire(ctx context.Context, key string, owner string, ttl time.Duration) (bool, error) {
	return f.locked, f.acquireErr
}
func (f *fakeBatchLockRepo) Release(ctx context.Context, key string, owner string) error {
	f.released = true
	return nil
}
func (f *fakeBatchLockRepo) Extend(ctx context.Context, key string, owner string, ttl time.Duration) (bool, error) {
	return true, nil
}

type fakeTxManager struct {
	repos  ITxRepos
	called bool
}

func (f *fakeTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context, tx ITxRepos) error) error {
	f.called = true
	return fn(ctx, f.repos)
}

type fakeTxRepos struct {
	user     repository.IUserRepository
	refresh  repository.IRefreshTokenRepository
	bean     repository.IBeanRepository
	article  repository.IArticleRepository
	rank     repository.IRankTargetRepository
	relation repository.IBeanArticleRepository
	metric   repository.IContentMetricRepository
	run      repository.IBatchRunRepository
}

func (f fakeTxRepos) User() repository.IUserRepository                   { return f.user }
func (f fakeTxRepos) RefreshToken() repository.IRefreshTokenRepository   { return f.refresh }
func (f fakeTxRepos) Bean() repository.IBeanRepository                   { return f.bean }
func (f fakeTxRepos) Article() repository.IArticleRepository             { return f.article }
func (f fakeTxRepos) RankTarget() repository.IRankTargetRepository       { return f.rank }
func (f fakeTxRepos) BeanArticle() repository.IBeanArticleRepository     { return f.relation }
func (f fakeTxRepos) ContentMetric() repository.IContentMetricRepository { return f.metric }
func (f fakeTxRepos) BatchRun() repository.IBatchRunRepository           { return f.run }

type fakeModalDisplayRepo struct {
	created      *model.ModalDisplayLog
	pageCount    int64
	sessionCount int64
	clicked      bool
	closed       bool
}

func (f *fakeModalDisplayRepo) Create(ctx context.Context, log *model.ModalDisplayLog) error {
	f.created = log
	if log.ID == 0 {
		log.ID = 1
	}
	return nil
}
func (f *fakeModalDisplayRepo) FindByIDForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64) (*model.ModalDisplayLog, error) {
	return &model.ModalDisplayLog{ID: id, RankTargetID: 1}, nil
}
func (f *fakeModalDisplayRepo) MarkClickedForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64, clickedAt time.Time) error {
	f.clicked = true
	return nil
}
func (f *fakeModalDisplayRepo) MarkClosedForActor(ctx context.Context, id uint64, userID *uint64, guestSessionID *uint64, closedAt time.Time) error {
	f.closed = true
	return nil
}
func (f *fakeModalDisplayRepo) CountShownInSession(ctx context.Context, userID *uint64, guestSessionID *uint64, since time.Time) (int64, error) {
	return f.sessionCount, nil
}
func (f *fakeModalDisplayRepo) CountShownOnPage(ctx context.Context, userID *uint64, guestSessionID *uint64, pagePath string, since time.Time) (int64, error) {
	return f.pageCount, nil
}
func (f *fakeModalDisplayRepo) WasTargetShownRecently(ctx context.Context, userID *uint64, guestSessionID *uint64, rankTargetID uint64, since time.Time) (bool, error) {
	return false, nil
}
func (f *fakeModalDisplayRepo) WasTargetClosedRecently(ctx context.Context, userID *uint64, guestSessionID *uint64, rankTargetID uint64, since time.Time) (bool, error) {
	return false, nil
}

type fakeModalBlockRepo struct{ created []*model.ModalBlockLog }

func (f *fakeModalBlockRepo) Create(ctx context.Context, log *model.ModalBlockLog) error {
	f.created = append(f.created, log)
	return nil
}
func (f *fakeModalBlockRepo) ListByActor(ctx context.Context, userID *uint64, guestSessionID *uint64, limit int) ([]*model.ModalBlockLog, error) {
	return nil, nil
}
func (f *fakeModalBlockRepo) CountByReasonSince(ctx context.Context, reason entity.ModalBlockReason, since time.Time) (int64, error) {
	return 0, nil
}
func (f *fakeModalBlockRepo) ListByCandidate(ctx context.Context, rankTargetID uint64, limit int) ([]*model.ModalBlockLog, error) {
	return nil, nil
}

type fakeModalSuppressionRepo struct {
	shown       bool
	closed      bool
	setShownID  uint64
	setClosedID uint64
}

func (f *fakeModalSuppressionRepo) SetShown(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error {
	f.setShownID = rankTargetID
	return nil
}
func (f *fakeModalSuppressionRepo) WasShown(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error) {
	return f.shown, nil
}
func (f *fakeModalSuppressionRepo) IncrementPageCount(ctx context.Context, actorKey string, pagePath string, ttl time.Duration) (int64, error) {
	return 0, nil
}
func (f *fakeModalSuppressionRepo) IncrementSessionCount(ctx context.Context, actorKey string, ttl time.Duration) (int64, error) {
	return 0, nil
}
func (f *fakeModalSuppressionRepo) SetClosed(ctx context.Context, actorKey string, rankTargetID uint64, ttl time.Duration) error {
	f.setClosedID = rankTargetID
	return nil
}
func (f *fakeModalSuppressionRepo) WasClosed(ctx context.Context, actorKey string, rankTargetID uint64) (bool, error) {
	return f.closed, nil
}
