package controller

import (
	"net/http"

	"coffee-ranker/model"
	"coffee-ranker/usecase"
	"coffee-ranker/validator"

	"github.com/labstack/echo/v4"
)

type AdminBeanController struct {
	beans     *usecase.AdminBeanUsecase
	validator *validator.AdminBeanValidator
}

type AdminArticleController struct {
	articles  *usecase.AdminArticleUsecase
	validator *validator.AdminArticleValidator
}

type AdminRelationController struct {
	relations *usecase.AdminRelationUsecase
	validator *validator.AdminRelationValidator
}

type AdminBatchController struct {
	batches   *usecase.AdminBatchUsecase
	validator *validator.AdminBatchValidator
}

type AdminAuditController struct {
	audits    *usecase.AdminAuditUsecase
	validator *validator.AdminAuditValidator
}

type CleanupController struct {
	cleanup *usecase.CleanupUsecase
}

type AdminRateLimitController struct {
	rates     *usecase.AdminRateLimitUsecase
	validator *validator.AdminRateLimitValidator
}

// NewAdminBeanControllerを生成してDI層やRouterから使えるようにする。
func NewAdminBeanController(beans *usecase.AdminBeanUsecase, validator *validator.AdminBeanValidator) *AdminBeanController {
	// 管理Bean操作はAdminGuard済みの前提で、UsecaseとValidatorだけを保持する。
	return &AdminBeanController{beans: beans, validator: validator}
}

// NewAdminArticleControllerを生成してDI層やRouterから使えるようにする。
func NewAdminArticleController(articles *usecase.AdminArticleUsecase, validator *validator.AdminArticleValidator) *AdminArticleController {
	// 管理Article操作はAdminGuard済みの前提で、UsecaseとValidatorだけを保持する。
	return &AdminArticleController{articles: articles, validator: validator}
}

// NewAdminRelationControllerを生成してDI層やRouterから使えるようにする。
func NewAdminRelationController(relations *usecase.AdminRelationUsecase, validator *validator.AdminRelationValidator) *AdminRelationController {
	// 関連管理はBean/Articleの存在確認をUsecaseに任せ、Controllerでは入力境界だけ扱う。
	return &AdminRelationController{relations: relations, validator: validator}
}

// NewAdminBatchControllerを生成してDI層やRouterから使えるようにする。
func NewAdminBatchController(batches *usecase.AdminBatchUsecase, validator *validator.AdminBatchValidator) *AdminBatchController {
	// 手動バッチ実行はUsecaseへ委譲し、Controllerではownerや対象IDの形式だけ確認する。
	return &AdminBatchController{batches: batches, validator: validator}
}

// NewAdminAuditControllerを生成してDI層やRouterから使えるようにする。
func NewAdminAuditController(audits *usecase.AdminAuditUsecase, validator *validator.AdminAuditValidator) *AdminAuditController {
	// 監査ログ検索のQueryをValidatorで整え、検索処理はUsecaseへ渡す。
	return &AdminAuditController{audits: audits, validator: validator}
}

// NewCleanupControllerを生成してDI層やRouterから使えるようにする。
func NewCleanupController(cleanup *usecase.CleanupUsecase) *CleanupController {
	return &CleanupController{cleanup: cleanup}
}

// NewAdminRateLimitControllerを生成してDI層やRouterから使えるようにする。
func NewAdminRateLimitController(rates *usecase.AdminRateLimitUsecase, validator *validator.AdminRateLimitValidator) *AdminRateLimitController {
	// RateLimit resetのkey形式だけを確認し、Redis操作はUsecase/Repositoryへ任せる。
	return &AdminRateLimitController{rates: rates, validator: validator}
}

// Admin用に公開状態に関係なくBean一覧を取得する。
func (h *AdminBeanController) List(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return writeError(c, err)
	}
	offset, err := parseIntQuery(c, "offset")
	if err != nil {
		return writeError(c, err)
	}
	beans, err := h.beans.List(c.Request().Context(), usecase.Page{Limit: limit, Offset: offset}, meta)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, beans)
}

// AdminのBean作成Requestを検証してAdminBeanUsecaseへ渡す。
// 管理者権限はAdminGuard前提で、DB作成可否はUsecaseが判断する。
func (h *AdminBeanController) Create(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.AdminBeanRequest
	// JSON bodyをRequest DTOへ読み取る。
	// ここではまだDB存在確認や業務判断は行わない。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Bean(req, false)
	if err != nil {
		return writeError(c, err)
	}
	bean := beanFromAdminRequest(valid)
	if err := h.beans.Create(c.Request().Context(), bean, meta); err != nil {
		return writeError(c, err)
	}
	// 作成成功としてHTTP201。
	return c.JSON(http.StatusCreated, bean)
}

// AdminのBean更新IDとRequest bodyを検証。
// pathのIDを正とし、bodyのIDと揃えてUsecaseへ渡す。
func (h *AdminBeanController) Update(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	var req validator.AdminBeanRequest
	// JSON bodyをRequest DTOへ読み取る。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	req.ID = id
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Bean(req, true)
	if err != nil {
		return writeError(c, err)
	}
	bean := beanFromAdminRequest(valid)
	if err := h.beans.Update(c.Request().Context(), bean, meta); err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, bean)
}

// Bean公開対象IDを検証して公開処理をUsecaseへ渡す。
func (h *AdminBeanController) Publish(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.ID(id); err != nil {
		return writeError(c, err)
	}
	if err := h.beans.Publish(c.Request().Context(), id, meta); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Bean非公開対象IDを検証して非公開処理をUsecaseへ渡す。
func (h *AdminBeanController) Unpublish(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.ID(id); err != nil {
		return writeError(c, err)
	}
	if err := h.beans.Unpublish(c.Request().Context(), id, meta); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Admin用に公開状態に関係なくArticle一覧を取得する。
func (h *AdminArticleController) List(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return writeError(c, err)
	}
	offset, err := parseIntQuery(c, "offset")
	if err != nil {
		return writeError(c, err)
	}
	articles, err := h.articles.List(c.Request().Context(), usecase.Page{Limit: limit, Offset: offset}, meta)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, articles)
}

// AdminのArticle作成Requestを検証してAdminArticleUsecaseへ渡す。
func (h *AdminArticleController) Create(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.AdminArticleRequest
	// JSON bodyをRequest DTOへ読み取る。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Article(req, false)
	if err != nil {
		return writeError(c, err)
	}
	article := articleFromAdminRequest(valid)
	if err := h.articles.Create(c.Request().Context(), article, meta); err != nil {
		return writeError(c, err)
	}
	// 作成成功としてHTTP 201で返す。
	return c.JSON(http.StatusCreated, article)
}

// AdminのArticle更新IDとRequest bodyを検証。
// pathのIDを正とし、bodyのIDと揃えてUsecaseへ渡す。
func (h *AdminArticleController) Update(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	var req validator.AdminArticleRequest
	// JSON bodyをRequest DTOへ読み取る。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	req.ID = id
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Article(req, true)
	if err != nil {
		return writeError(c, err)
	}
	article := articleFromAdminRequest(valid)
	if err := h.articles.Update(c.Request().Context(), article, meta); err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, article)
}

// Article公開対象IDを検証して公開処理をUsecaseへ渡す。
func (h *AdminArticleController) Publish(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.ID(id); err != nil {
		return writeError(c, err)
	}
	if err := h.articles.Publish(c.Request().Context(), id, meta); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Article非公開対象IDを検証して非公開処理をUsecaseへ渡す。
func (h *AdminArticleController) Unpublish(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	id, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.ID(id); err != nil {
		return writeError(c, err)
	}
	if err := h.articles.Unpublish(c.Request().Context(), id, meta); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// Bean/Article関連作成Requestを検証してUsecaseへ渡す。
func (h *AdminRelationController) Create(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.RelationRequest
	// JSON bodyをRequest DTOへ読み取る。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Relation(req)
	if err != nil {
		return writeError(c, err)
	}
	relation, err := h.relations.Create(c.Request().Context(), valid.BeanID, valid.ArticleID, valid.DisplayOrder, meta)
	if err != nil {
		return writeError(c, err)
	}
	// 作成成功としてHTTP 201で返す。
	return c.JSON(http.StatusCreated, relation)
}

// 関連削除対象のbean_id/article_idを検証。
// 削除可能かどうかはUsecaseで判断。
func (h *AdminRelationController) Delete(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	beanID, err := parseUintParam(c, "bean_id")
	if err != nil {
		return writeError(c, err)
	}
	articleID, err := parseUintParam(c, "article_id")
	if err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Relation(validator.RelationRequest{BeanID: beanID, ArticleID: articleID})
	if err != nil {
		return writeError(c, err)
	}
	if err := h.relations.Delete(c.Request().Context(), valid.BeanID, valid.ArticleID, meta); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// 指定Beanの関連Article一括差し替えRequestを検証。
func (h *AdminRelationController) ReplaceByBeanID(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	beanID, err := parseUintParam(c, "bean_id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.BeanID(beanID); err != nil {
		return writeError(c, err)
	}
	var req validator.ReplaceRelationsRequest
	// JSON bodyをRequest DTOへ読み取る。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Replace(req)
	if err != nil {
		return writeError(c, err)
	}
	if err := h.relations.ReplaceByBeanID(c.Request().Context(), beanID, valid.ArticleIDs, meta); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// 指定Beanの関連Article表示順更新Requestを検証。
// 表示順の更新処理と関連整合性はUsecaseで扱う。
func (h *AdminRelationController) UpdateDisplayOrder(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	beanID, err := parseUintParam(c, "bean_id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.BeanID(beanID); err != nil {
		return writeError(c, err)
	}
	var req validator.ReplaceRelationsRequest
	// JSON bodyをRequest DTOへ読み取る。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Replace(req)
	if err != nil {
		return writeError(c, err)
	}
	if err := h.relations.UpdateDisplayOrder(c.Request().Context(), beanID, valid.ArticleIDs, meta); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ランキング手動バッチ実行Requestを検証。
// 二重実行防止や集計処理はAdminBatchUsecaseへ任せる。
func (h *AdminBatchController) RunRanking(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.BatchRunRequest
	// JSON bodyをRequest DTOへ読み取る。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Batch(req)
	if err != nil {
		return writeError(c, err)
	}
	run, err := h.batches.RunRanking(c.Request().Context(), meta.AdminUserID, valid.Owner, meta.AuditMeta)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusAccepted, run)
}

// 興味プロフィール手動バッチ実行Requestを検証。
// 対象User/Guestの集計処理はAdminBatchUsecaseへ任せる。
func (h *AdminBatchController) RunInterest(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.BatchRunRequest
	// JSON bodyをRequest DTOへ読み取る。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Batch(req)
	if err != nil {
		return writeError(c, err)
	}
	run, err := h.batches.RunInterest(c.Request().Context(), meta.AdminUserID, valid.Owner, valid.UserIDs, valid.GuestSessionIDs, meta.AuditMeta)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusAccepted, run)
}

// バッチ実行履歴一覧のページングを検証。
func (h *AdminBatchController) ListRuns(c echo.Context) error {
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return writeError(c, err)
	}
	offset, err := parseIntQuery(c, "offset")
	if err != nil {
		return writeError(c, err)
	}
	page, err := h.validator.List(validator.PageQuery{Limit: limit, Offset: offset})
	if err != nil {
		return writeError(c, err)
	}
	runs, err := h.batches.ListRuns(c.Request().Context(), usecase.Page{Limit: page.Limit, Offset: page.Offset})
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, runs)
}

// 最新バッチ履歴取得用job名を検証。
func (h *AdminBatchController) Latest(c echo.Context) error {
	jobName, err := h.validator.JobName(c.QueryParam("job_name"))
	if err != nil {
		return writeError(c, err)
	}
	run, err := h.batches.Latest(c.Request().Context(), jobName)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, run)
}

// 監査ログ一覧queryを検証してUsecaseへ渡す。
// 監査ログ検索条件の形式だけを確認します。
func (h *AdminAuditController) List(c echo.Context) error {
	limit, err := parseIntQuery(c, "limit")
	if err != nil {
		return writeError(c, err)
	}
	offset, err := parseIntQuery(c, "offset")
	if err != nil {
		return writeError(c, err)
	}
	actorUserID, err := parseIntQuery(c, "actor_user_id")
	if err != nil {
		return writeError(c, err)
	}
	targetID, err := parseIntQuery(c, "target_id")
	if err != nil {
		return writeError(c, err)
	}
	filter, err := h.validator.List(validator.AuditQuery{ActorType: c.QueryParam("actor_type"), ActorUserID: actorUserID, Action: c.QueryParam("action"), TargetType: c.QueryParam("target_type"), TargetID: targetID, Limit: limit, Offset: offset})
	if err != nil {
		return writeError(c, err)
	}
	logs, err := h.audits.List(c.Request().Context(), filter)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, logs)
}

// 監査ログ詳細IDを検証してUsecaseへ渡す。
// 対象ログが存在するかどうかはUsecaseで判断。
func (h *AdminAuditController) FindByID(c echo.Context) error {
	id, err := parseUintParam(c, "id")
	if err != nil {
		return writeError(c, err)
	}
	if err := h.validator.ID(id); err != nil {
		return writeError(c, err)
	}
	log, err := h.audits.FindByID(c.Request().Context(), id)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, log)
}

// 監査ログ一覧queryを検証してUsecaseへ渡す。
// 監査ログ検索条件の形式だけを確認。
func (h *AdminAuditController) ListByRequestID(c echo.Context) error {
	requestID, err := h.validator.RequestID(c.Param("request_id"))
	if err != nil {
		return writeError(c, err)
	}
	logs, err := h.audits.ListByRequestID(c.Request().Context(), requestID)
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, logs)
}

// 期限切れデータ削除Usecaseを呼び出す。
// 入力値がないため、Controllerでは認可後の実行境界だけを担当。
func (h *CleanupController) DeleteExpired(c echo.Context) error {
	result, err := h.cleanup.DeleteExpired(c.Request().Context())
	if err != nil {
		return writeError(c, err)
	}
	// Usecaseの結果をHTTPレスポンスDTOとして返す。
	return c.JSON(http.StatusOK, result)
}

// RateLimit reset対象keyを検証してUsecaseへ渡す。

func (h *AdminRateLimitController) Reset(c echo.Context) error {
	meta, err := adminMeta(c)
	if err != nil {
		return writeError(c, err)
	}
	var req validator.RateLimitResetRequest
	// JSON bodyをRequest DTOへ読み取る。
	if err := c.Bind(&req); err != nil {
		return writeError(c, err)
	}
	// ValidatorでUsecaseへ渡してよい形に。
	valid, err := h.validator.Reset(req)
	if err != nil {
		return writeError(c, err)
	}
	if err := h.rates.Reset(c.Request().Context(), valid.Key, meta); err != nil {
		return writeError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// 検証済みAdminBeanRequestをmodel.Beanへ変換。
// Controller内ではDB操作せず、Usecaseに渡す入力形へ。
func beanFromAdminRequest(req validator.AdminBeanRequest) *model.Bean {
	return &model.Bean{ID: req.ID, Name: req.Name, Roaster: req.Roaster, Origin: req.Origin, Region: req.Region, Farm: req.Farm, Variety: req.Variety, RoastLevel: req.RoastLevel, Acidity: req.Acidity, Bitterness: req.Bitterness, Flavor: req.Flavor, Aroma: req.Aroma, Body: req.Body, FlavorNote: req.FlavorNote, Description: req.Description, ImageURL: req.ImageURL, IsPublished: req.IsPublished}
}

// 検証済みAdminArticleRequestをmodel.Articleへ変換。
// Controller内ではDB操作せず、Usecaseに渡す入力形へ。
func articleFromAdminRequest(req validator.AdminArticleRequest) *model.Article {
	return &model.Article{ID: req.ID, Title: req.Title, Slug: req.Slug, Summary: req.Summary, Body: req.Body, Category: req.Category, SourceName: req.SourceName, SourceURL: req.SourceURL, ImageURL: req.ImageURL, IsPublished: req.IsPublished, PublishedAt: req.PublishedAt}
}
