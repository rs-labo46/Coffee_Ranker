//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const (
	csrfHeaderName     = "X-CSRF-Token"
	csrfCookieName     = "csrf_token"
	refreshCookieName  = "refresh_token"
	defaultContentType = "application/json"
)

// apiClientは、実際に起動しているHTTP APIへアクセスするE2E専用client。
// CookieJarを持たせ、RefreshToken cookieとCSRF cookieをPostman相当の状態で扱う。
type apiClient struct {
	t           *testing.T
	baseURL     string
	base        *url.URL
	http        *http.Client
	accessToken string
	csrfToken   string
	userID      uint64
	email       string
	password    string
}

// apiResponseは、bodyを読み切った後でも検証しやすいようにstatus/header/bodyを保持する。
type apiResponse struct {
	Status int
	Header http.Header
	Body   []byte
}

// TestAPIE2E_AuthFlowは、認証の主要経路をHTTP経由で検証する。
// signup/login/me/refresh/logout/logout-all/reuse検知をまとめて確認する。
func TestAPIE2E_AuthFlow(t *testing.T) {
	resetE2EState(t)
	c := newAPIClient(t)

	c.getAndRequire("/health", http.StatusOK, false)
	c.issueCSRF()
	c.signupUniqueUser()
	c.login(c.email, c.password)
	c.getAndRequire("/auth/me", http.StatusOK, true)

	// 通常のRefresh rotationが2回連続で成功することを確認する。
	c.refreshAndRequire(http.StatusOK)
	c.refreshAndRequire(http.StatusOK)
	c.getAndRequire("/auth/me", http.StatusOK, true)

	// logout後はRefreshToken cookieが使えず、refreshが401になることを確認する。
	c.postJSONAndRequire("/auth/logout", nil, http.StatusNoContent, true, true)
	c.issueCSRF()
	c.refreshAndRequire(http.StatusUnauthorized)

	// logout-all後はtoken_versionが上がり、既存AccessTokenが無効になることを確認する。
	c.login(c.email, c.password)
	c.issueCSRF()
	c.postJSONAndRequire("/auth/logout-all", nil, http.StatusNoContent, true, true)
	c.getAndRequire("/auth/me", http.StatusUnauthorized, true)

	// 古いRefreshTokenを再利用したとき、reuse検知で401になり、既存AccessTokenも無効になることを確認する。
	c.login(c.email, c.password)
	c.issueCSRF()
	oldRefresh := c.currentRefreshToken()
	if oldRefresh == "" {
		t.Fatal("login後のrefresh_token cookieが取得できない")
	}
	c.refreshAndRequire(http.StatusOK)
	c.setRefreshToken(oldRefresh)
	c.refreshAndRequire(http.StatusUnauthorized)
	c.getAndRequire("/auth/me", http.StatusUnauthorized, true)
}

// TestAPIE2E_CoreAPIsは、Backend完成時点で確認するAPI E2Eをまとめて検証する。
// Content/Event/Saved/Rating/Recommendation/Admin BatchをPostman手動操作の代替として実行する。
func TestAPIE2E_CoreAPIs(t *testing.T) {
	resetE2EState(t)
	c := newAPIClient(t)

	c.getAndRequire("/health", http.StatusOK, false)
	c.getAndRequire("/ready", http.StatusOK, false)
	c.issueCSRF()
	c.signupUniqueUser()
	c.login(c.email, c.password)

	var rankTargetID uint64

	t.Run("Content", func(t *testing.T) {
		c.getAndRequire("/beans?limit=10&offset=0", http.StatusOK, false)

		articles := c.getAndRequire("/articles?limit=10&offset=0", http.StatusOK, false)
		if articleID, articleSlug := firstArticleIdentity(articles); articleID > 0 && articleSlug != "" {
			c.getAndRequire(fmt.Sprintf("/articles/id/%d", articleID), http.StatusOK, true)
			c.getAndRequire("/articles/"+url.PathEscape(articleSlug), http.StatusOK, true)
		}

		top := c.getAndRequire("/rankings/top?limit=10", http.StatusOK, false)
		rankTargetID = firstRankTargetID(top)
		if rankTargetID == 0 {
			t.Fatalf("rank_target_idを取得できない。公開済みContentとcontent_metricsが必要。先に管理者でranking batchを実行するか、seedを用意する")
		}

		c.getAndRequire("/rankings?content_type=bean&limit=10&offset=0", http.StatusOK, false)
		c.getAndRequire("/rankings?content_type=article&limit=10&offset=0", http.StatusOK, false)
	})

	t.Run("Search", func(t *testing.T) {
		c.getAndRequire("/search/beans?q=ethiopia&limit=10&offset=0", http.StatusOK, true)
		c.getAndRequire("/search/articles?q=brew&limit=10&offset=0", http.StatusOK, true)
	})

	t.Run("Event", func(t *testing.T) {
		unique := strconv.FormatInt(time.Now().UnixNano(), 10)
		c.recordEvent(map[string]interface{}{
			"event_type":        "impression",
			"rank_target_id":    rankTargetID,
			"placement":         "search_result",
			"page_path":         "/search",
			"dedup_key":         "e2e-impression-" + unique,
			"dedup_ttl_seconds": 60,
		})
		c.recordEvent(map[string]interface{}{
			"event_type":        "content_view",
			"rank_target_id":    rankTargetID,
			"placement":         "search_result",
			"page_path":         "/search",
			"dedup_key":         "e2e-content-view-" + unique,
			"dedup_ttl_seconds": 60,
		})
		c.recordEvent(map[string]interface{}{
			"event_type":        "stay",
			"rank_target_id":    rankTargetID,
			"placement":         "bean_detail",
			"dwell_ms":          30000,
			"page_path":         "/beans/e2e",
			"dedup_key":         "e2e-stay-" + unique,
			"dedup_ttl_seconds": 60,
		})
		c.recordEvent(map[string]interface{}{
			"event_type":              "re_search",
			"search_condition_hash":   strings.Repeat("a", 64),
			"previous_condition_hash": strings.Repeat("b", 64),
			"search_keyword":          "ethiopia",
			"search_origin":           "Ethiopia",
			"search_roast_level":      "light",
			"search_acidity":          4,
			"search_bitterness":       2,
			"search_flavor":           5,
			"search_aroma":            4,
			"search_body":             3,
			"page_path":               "/search",
		})
	})

	t.Run("Saved", func(t *testing.T) {
		c.postJSONAndRequire("/saved", map[string]interface{}{
			"rank_target_id": rankTargetID,
			"placement":      "search_result",
			"page_path":      "/search",
		}, http.StatusCreated, true, true)
		c.getAndRequire("/saved?limit=10&offset=0", http.StatusOK, true)
		c.getAndRequire(fmt.Sprintf("/saved/%d", rankTargetID), http.StatusOK, true)
		c.deleteAndRequire(fmt.Sprintf("/saved/%d", rankTargetID), http.StatusNoContent, true, true)
		c.getAndRequire(fmt.Sprintf("/saved/%d", rankTargetID), http.StatusOK, true)
	})

	t.Run("Rating", func(t *testing.T) {
		c.postJSONAndRequire("/ratings", map[string]interface{}{
			"rank_target_id": rankTargetID,
			"score":          1,
			"placement":      "search_result",
			"page_path":      "/search",
		}, http.StatusOK, true, true)
		c.getAndRequire(fmt.Sprintf("/ratings/%d", rankTargetID), http.StatusOK, true)
		c.postJSONAndRequire("/ratings", map[string]interface{}{
			"rank_target_id": rankTargetID,
			"score":          -1,
			"placement":      "search_result",
			"page_path":      "/search",
		}, http.StatusOK, true, true)
		c.deleteAndRequire(fmt.Sprintf("/ratings/%d", rankTargetID), http.StatusNoContent, true, true)
		c.getAndRequire(fmt.Sprintf("/ratings/%d", rankTargetID), http.StatusNotFound, true)
	})

	t.Run("Recommendation", func(t *testing.T) {
		// Recommendation検証は、Event/Saved/Ratingで使ったUserの閲覧履歴に引っ張られないよう専用Userで行う。
		recClient := newAPIClient(t)
		recClient.issueCSRF()
		recClient.signupUniqueUser()
		recClient.login(recClient.email, recClient.password)

		resp := recClient.getAndRequire("/recommendations?content_type=bean&limit=10&offset=0", http.StatusOK, true)
		items := objectItems(resp)
		if len(items) > 0 {
			if _, ok := items[0]["reasons"]; !ok {
				t.Fatalf("recommendation itemにreasonsがない: %s", string(resp.Body))
			}
		}

		// 直近閲覧済み除外を検証する。content_view済みのrank_target_idが推薦候補に混ざらないことを見る。
		unique := strconv.FormatInt(time.Now().UnixNano(), 10)
		recClient.recordEvent(map[string]interface{}{
			"event_type":        "content_view",
			"rank_target_id":    rankTargetID,
			"placement":         "search_result",
			"page_path":         "/search",
			"dedup_key":         "e2e-viewed-exclusion-" + unique,
			"dedup_ttl_seconds": 60,
		})
		afterView := recClient.getAndRequire("/recommendations?limit=50&offset=0", http.StatusOK, true)
		if containsRankTargetID(afterView, rankTargetID) {
			t.Fatalf("閲覧済みrank_target_id=%dがrecommendationsに含まれている: %s", rankTargetID, string(afterView.Body))
		}
	})

	t.Run("AdminBatch", func(t *testing.T) {
		admin := newAPIClient(t)
		adminEmail := firstNonEmpty(os.Getenv("E2E_ADMIN_EMAIL"), os.Getenv("SEED_ADMIN_EMAIL"))
		adminPassword := firstNonEmpty(os.Getenv("E2E_ADMIN_PASSWORD"), os.Getenv("SEED_ADMIN_PASSWORD"))
		if adminEmail == "" || adminPassword == "" {
			t.Skip("E2E_ADMIN_EMAIL/E2E_ADMIN_PASSWORD、またはSEED_ADMIN_EMAIL/SEED_ADMIN_PASSWORDが未設定のためAdmin Batch E2Eをskip")
		}

		admin.issueCSRF()
		admin.login(adminEmail, adminPassword)
		admin.postJSONAndRequire("/admin/batches/ranking", map[string]interface{}{
			"owner": "e2e-ranking",
		}, http.StatusAccepted, true, true)
		admin.postJSONAndRequire("/admin/batches/interest", map[string]interface{}{
			"owner":    "e2e-interest",
			"user_ids": []uint64{c.userID},
		}, http.StatusAccepted, true, true)
		admin.getAndRequire("/admin/batches?limit=10&offset=0", http.StatusOK, true)
		admin.getAndRequire("/admin/batches/latest?job_name=ranking", http.StatusOK, true)
		admin.getAndRequire("/admin/batches/latest?job_name=interest", http.StatusOK, true)
	})
}

// resetE2EStateは、E2E開始前にRedis上のrate limit / dedup / suppressionを初期化する。
// AuthFlowとCoreAPIsを連続実行しても、前のテストのRateLimitが次のテストを429にしないようにする。
func resetE2EState(t *testing.T) {
	t.Helper()

	if os.Getenv("E2E_SKIP_REDIS_RESET") == "1" {
		return
	}

	addr := strings.TrimSpace(os.Getenv("E2E_REDIS_ADDR"))
	if addr == "" {
		addr = "localhost:26379"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("E2E_REDIS_PASSWORD"),
		DB:       envInt("E2E_REDIS_DB", 0),
	})
	defer rdb.Close()

	if err := rdb.FlushDB(ctx).Err(); err != nil {
		t.Fatalf("E2E Redis初期化失敗 addr=%s: %v。必要なら E2E_REDIS_ADDR=localhost:26379 を確認する", addr, err)
	}
}

// newAPIClientは、E2E_BASE_URLを基準にHTTP clientを生成する。
// 未指定時はdocker composeの標準想定であるhttp://localhost:8080を使う。
func newAPIClient(t *testing.T) *apiClient {
	t.Helper()

	baseURL := strings.TrimRight(firstNonEmpty(os.Getenv("E2E_BASE_URL"), "http://localhost:8080"), "/")
	parsed, err := url.Parse(baseURL)
	if err != nil {
		t.Fatalf("E2E_BASE_URLが不正: %v", err)
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar作成失敗: %v", err)
	}
	return &apiClient{
		t:       t,
		baseURL: baseURL,
		base:    parsed,
		http: &http.Client{
			Jar:     jar,
			Timeout: 15 * time.Second,
		},
	}
}

// signupUniqueUserは、再実行してもemail重複しにくいテストUserを作成する。
func (c *apiClient) signupUniqueUser() {
	c.t.Helper()
	stamp := time.Now().UnixNano()
	c.email = fmt.Sprintf("e2e-user-%d@example.com", stamp)
	c.password = "Password123!"
	c.postJSONAndRequire("/auth/signup", map[string]interface{}{
		"name":     "E2E User",
		"email":    c.email,
		"password": c.password,
	}, http.StatusCreated, false, false)
}

// loginは、AccessTokenとUserIDをレスポンスから保存する。
func (c *apiClient) login(email string, password string) {
	c.t.Helper()
	resp := c.postJSONAndRequire("/auth/login", map[string]interface{}{
		"email":    email,
		"password": password,
	}, http.StatusOK, false, false)
	m := jsonObject(c.t, resp.Body)
	accessToken := stringField(m, "access_token")
	if accessToken == "" {
		c.t.Fatalf("login responseにaccess_tokenがない: %s", string(resp.Body))
	}
	c.accessToken = accessToken
	if user, ok := m["user"].(map[string]interface{}); ok {
		c.userID = uintField(user, "id")
	}
}

// issueCSRFは、CSRF cookieとheader用tokenを取得する。
func (c *apiClient) issueCSRF() {
	c.t.Helper()
	resp := c.getAndRequire("/auth/csrf", http.StatusOK, false)
	m := jsonObject(c.t, resp.Body)
	c.csrfToken = stringField(m, csrfCookieName)
	if c.csrfToken == "" {
		c.csrfToken = stringField(m, "csrf_token")
	}
	if c.csrfToken == "" {
		c.t.Fatalf("csrf_tokenが取得できない: %s", string(resp.Body))
	}
}

// refreshAndRequireは、refresh結果が200なら新しいAccessTokenを保存する。
func (c *apiClient) refreshAndRequire(status int) apiResponse {
	c.t.Helper()
	resp := c.postJSONAndRequire("/auth/refresh", nil, status, false, true)
	if status == http.StatusOK {
		m := jsonObject(c.t, resp.Body)
		accessToken := stringField(m, "access_token")
		if accessToken == "" {
			c.t.Fatalf("refresh responseにaccess_tokenがない: %s", string(resp.Body))
		}
		c.accessToken = accessToken
	}
	return resp
}

// recordEventは、Event APIが201またはdedup時204で成功することを確認する。
func (c *apiClient) recordEvent(body map[string]interface{}) {
	c.t.Helper()
	resp := c.do(http.MethodPost, "/events", body, true, true)
	if resp.Status != http.StatusCreated && resp.Status != http.StatusNoContent {
		c.t.Fatalf("POST /events status=%d body=%s", resp.Status, string(resp.Body))
	}
}

func (c *apiClient) getAndRequire(path string, status int, auth bool) apiResponse {
	c.t.Helper()
	resp := c.do(http.MethodGet, path, nil, auth, false)
	if resp.Status != status {
		c.t.Fatalf("GET %s status=%d want=%d body=%s", path, resp.Status, status, string(resp.Body))
	}
	return resp
}

func (c *apiClient) postJSONAndRequire(path string, body interface{}, status int, auth bool, csrf bool) apiResponse {
	c.t.Helper()
	resp := c.do(http.MethodPost, path, body, auth, csrf)
	if resp.Status != status {
		c.t.Fatalf("POST %s status=%d want=%d body=%s", path, resp.Status, status, string(resp.Body))
	}
	return resp
}

func (c *apiClient) deleteAndRequire(path string, status int, auth bool, csrf bool) apiResponse {
	c.t.Helper()
	resp := c.do(http.MethodDelete, path, nil, auth, csrf)
	if resp.Status != status {
		c.t.Fatalf("DELETE %s status=%d want=%d body=%s", path, resp.Status, status, string(resp.Body))
	}
	return resp
}

// doは、認証Header、CSRF Header、CookieJarを組み合わせてHTTPリクエストを送る。
func (c *apiClient) do(method string, path string, body interface{}, auth bool, csrf bool) apiResponse {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("request json marshal失敗: %v", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		c.t.Fatalf("request作成失敗: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", defaultContentType)
	}
	if auth {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	if csrf {
		req.Header.Set(csrfHeaderName, c.csrfToken)
	}

	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("request失敗 %s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatalf("response body読取失敗: %v", err)
	}
	return apiResponse{Status: res.StatusCode, Header: res.Header.Clone(), Body: bodyBytes}
}

// currentRefreshTokenは、CookieJar内の現在のrefresh_tokenを返す。
func (c *apiClient) currentRefreshToken() string {
	c.t.Helper()
	for _, cookie := range c.http.Jar.Cookies(c.base) {
		if cookie.Name == refreshCookieName {
			return cookie.Value
		}
	}
	return ""
}

// setRefreshTokenは、reuse検知テスト用に古いrefresh_tokenをCookieJarへ戻す。
func (c *apiClient) setRefreshToken(value string) {
	c.t.Helper()
	c.http.Jar.SetCookies(c.base, []*http.Cookie{{Name: refreshCookieName, Value: value, Path: "/"}})
}

func jsonObject(t *testing.T, body []byte) map[string]interface{} {
	t.Helper()
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("JSON object decode失敗: %v body=%s", err, string(body))
	}
	return m
}

func objectItems(resp apiResponse) []map[string]interface{} {
	var value interface{}
	if err := json.Unmarshal(resp.Body, &value); err != nil {
		return nil
	}
	return objectItemsFromValue(value)
}

// objectItemsFromValueは、APIごとに異なる一覧レスポンス形を吸収する。
// 配列直返し、items/data、RankingResultのMetricsなどを共通処理する。
func objectItemsFromValue(value interface{}) []map[string]interface{} {
	switch v := value.(type) {
	case []interface{}:
		return objectsFromArray(v)
	case map[string]interface{}:
		for _, key := range []string{"items", "data", "beans", "articles", "rankings", "metrics", "Metrics"} {
			if arr, ok := v[key].([]interface{}); ok {
				return objectsFromArray(arr)
			}
		}
	}
	return nil
}

func objectsFromArray(items []interface{}) []map[string]interface{} {
	objects := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		if obj, ok := item.(map[string]interface{}); ok {
			objects = append(objects, obj)
		}
	}
	return objects
}

func firstRankTargetID(resp apiResponse) uint64 {
	for _, item := range objectItems(resp) {
		if id := uintField(item, "rank_target_id"); id > 0 {
			return id
		}
		if rankTarget, ok := item["rank_target"].(map[string]interface{}); ok {
			if id := uintField(rankTarget, "id"); id > 0 {
				return id
			}
		}
	}
	return 0
}

func containsRankTargetID(resp apiResponse, want uint64) bool {
	for _, item := range objectItems(resp) {
		if uintField(item, "rank_target_id") == want {
			return true
		}
	}
	return false
}

func firstArticleIdentity(resp apiResponse) (uint64, string) {
	for _, item := range objectItems(resp) {
		id := uintField(item, "id")
		slug := stringField(item, "slug")
		if id > 0 && slug != "" {
			return id, slug
		}
	}
	return 0, ""
}

func stringField(m map[string]interface{}, key string) string {
	if value, ok := m[key].(string); ok {
		return value
	}
	return ""
}

func uintField(m map[string]interface{}, key string) uint64 {
	switch v := m[key].(type) {
	case float64:
		if v > 0 {
			return uint64(v)
		}
	case int:
		if v > 0 {
			return uint64(v)
		}
	case json.Number:
		parsed, _ := strconv.ParseUint(v.String(), 10, 64)
		return parsed
	case string:
		parsed, _ := strconv.ParseUint(v, 10, 64)
		return parsed
	}
	return 0
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
