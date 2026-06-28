package usecase

import (
	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
	"context"
	"errors"
	"strings"
	"time"
)

// password hash化と照合。
type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
	Compare(ctx context.Context, password string, passwordHash string) error
}

// RefreshToken生成、hash化、AccessToken発行。
type TokenManager interface {
	NewRefreshToken(ctx context.Context) (plain string, hash string, err error)
	NewFamilyID(ctx context.Context) (string, error)
	HashRefreshToken(ctx context.Context, token string) (string, error)
	IssueAccessToken(ctx context.Context, user *model.User, now time.Time) (string, error)
}

// Signup、Login、Refresh、Logout、Meを扱う認証。
type AuthUsecase struct {
	users         repository.UserRepository
	refreshTokens repository.RefreshTokenRepository
	audits        repository.AuditLogRepository
	txManager     repository.TxManager
	passwords     PasswordHasher
	tokens        TokenManager
	refreshTTL    time.Duration
}

// 監査ログに付与するrequest_idとIP hashをまとめる。
type AuditMeta struct {
	RequestID     *string
	IPAddressHash *string
}

// SignupでControllerから受け取る入力値。
type SignupInput struct {
	Name     string
	Email    string
	Password string
}

// LoginでControllerから受け取る入力値と監査情報。
type LoginInput struct {
	Email    string
	Password string
	Meta     AuditMeta
}

// Login成功時に返すUser、AccessToken、RefreshToken。
type AuthTokenResult struct {
	User         *model.User
	AccessToken  string
	RefreshToken string
}

// Refresh成功時に返すUser、AccessToken、RefreshToken。
type RefreshResult struct {
	User         *model.User
	AccessToken  string
	RefreshToken string
}

// emailの前後空白を取り除き、小文字に統一。同じ文字列でも別アカウントとして登録できる可能性があるから。
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// 認証済みUserIDが0でないことを確認。
func requireUserID(userID uint64) error {
	if userID == 0 {
		return entity.ErrUnauthorized
	}
	return nil
}

// 監査ログの作成失敗で、本体の処理を失敗させない
func safeAudit(ctx context.Context, repo repository.AuditLogRepository, log *model.AuditLog) {
	if repo == nil || log == nil {
		return
	}
	_ = repo.Create(ctx, log)
}

// 保存するログの内容
func auditLog(actorType entity.AuditActorType, actorUserID *uint64, action entity.AuditAction, targetType *string, targetID *uint64, detail *string, meta AuditMeta) *model.AuditLog {
	return &model.AuditLog{
		ActorType:     actorType,
		ActorUserID:   actorUserID,
		Action:        action,
		TargetType:    targetType,
		TargetID:      targetID,
		Detail:        detail,
		IPAddressHash: meta.IPAddressHash,
		RequestID:     meta.RequestID,
	}
}

// 認証処理に必要なRepository、TxManager、token/password処理の受け取り
func NewAuthUsecase(users repository.UserRepository, refreshTokens repository.RefreshTokenRepository, audits repository.AuditLogRepository, txManager repository.TxManager, passwords PasswordHasher, tokens TokenManager, refreshTTL time.Duration) *AuthUsecase {
	return &AuthUsecase{
		users:         users,
		refreshTokens: refreshTokens,
		audits:        audits,
		txManager:     txManager,
		passwords:     passwords,
		tokens:        tokens,
		refreshTTL:    refreshTTL,
	}
}

// ユーザー作成,emailの重複確認とパスワードのハッシュか
func (u *AuthUsecase) Signup(ctx context.Context, input SignupInput) (*model.User, error) {
	//入力値が空だった場合
	if input.Name == "" || input.Email == "" || input.Password == "" {
		return nil, entity.ErrInvalidInput
	}

	email := normalizeEmail(input.Email)
	exists, err := u.users.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	//すでにあった場合
	if exists {
		return nil, entity.ErrEmailAlreadyExists
	}

	//passwordのハッシュか
	hash, err := u.passwords.Hash(ctx, input.Password)
	if err != nil {
		return nil, entity.ErrCreateFailed
	}

	user := &model.User{
		Name:         input.Name,
		Email:        email,
		PasswordHash: hash,
		Role:         entity.UserRoleUser,
		Status:       entity.UserStatusActive,
		TokenVersion: 0,
	}
	if err := u.users.Create(ctx, user); err != nil {
		return nil, err
	}
	//作成できたらuserを返す
	return user, nil
}

// ログイン:email検索、status確認、password照合、RefreshToken発行、AccessToken発行
func (u *AuthUsecase) Login(ctx context.Context, input LoginInput) (AuthTokenResult, error) {
	//入力された値がからだった場合
	if input.Email == "" || input.Password == "" {
		return AuthTokenResult{}, entity.ErrInvalidInput
	}

	user, err := u.users.FindByEmail(ctx, normalizeEmail(input.Email))
	if err != nil {
		//このemailは登録されてない
		if errors.Is(err, entity.ErrNotFound) {
			return AuthTokenResult{}, entity.ErrInvalidCredentials
		}
		return AuthTokenResult{}, err
	}

	//取得してユーザーがログイン可能かどうか
	if err := ensureActiveUser(user); err != nil {
		return AuthTokenResult{}, err
	}

	//入力されたパスワードと、DBに保存されているハッシュ化済みパスワードかどうか比較する。
	if err := u.passwords.Compare(ctx, input.Password, user.PasswordHash); err != nil {
		return AuthTokenResult{}, entity.ErrInvalidCredentials
	}

	//ログインできたら、RefreshToken用の生値とDB保存用のhash、family_idを作る。
	plainRefresh, refreshHash, err := u.tokens.NewRefreshToken(ctx)
	if err != nil {
		return AuthTokenResult{}, entity.ErrCreateFailed
	}
	//新しいRefreshTokenを作っても同じ family_id を引き継ぐ。reuse検知やlogout時に、同じfamilyをまとめて失効するために使う。
	familyID, err := u.tokens.NewFamilyID(ctx)
	if err != nil {
		return AuthTokenResult{}, entity.ErrCreateFailed
	}

	now := time.Now()
	refreshToken := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: refreshHash,
		FamilyID:  familyID,
		ExpiresAt: now.Add(u.refreshTTL),
		CreatedAt: now,
	}

	//RefreshTokenをDBに保存
	if err := u.refreshTokens.Create(ctx, refreshToken); err != nil {
		return AuthTokenResult{}, err
	}

	// AccessTokenを作成
	accessToken, err := u.tokens.IssueAccessToken(ctx, user, now)
	if err != nil {
		return AuthTokenResult{}, entity.ErrCreateFailed
	}
	//ログイン成功を監査ログに残す
	safeAudit(ctx, u.audits, auditLog(entity.AuditActorUser, &user.ID, entity.AuditActionLogin, nil, nil, nil, input.Meta))

	return AuthTokenResult{User: user, AccessToken: accessToken, RefreshToken: plainRefresh}, nil
}

// RefreshToken rotationを行い、正常なら新しいAccessTokenとRefreshTokenを返す。
func (u *AuthUsecase) Refresh(ctx context.Context, refreshToken string, meta AuditMeta) (RefreshResult, error) {
	if refreshToken == "" {
		return RefreshResult{}, entity.ErrInvalidToken
	}

	// DBにはRefreshTokenを保存しない。
	// 送られてきた生RefreshTokenをhash化し、DBのtoken_hashと照合する。
	oldHash, err := u.tokens.HashRefreshToken(ctx, refreshToken)
	if err != nil {
		return RefreshResult{}, entity.ErrInvalidToken
	}

	// 新しいRefreshTokenの生値とDB保存用hashを作成。
	// 生値はControllerでCookieに設定し、hashだけをDBに保存。
	newPlain, newHash, err := u.tokens.NewRefreshToken(ctx)
	if err != nil {
		return RefreshResult{}, entity.ErrCreateFailed
	}

	now := time.Now()

	var user *model.User
	var newToken *model.RefreshToken
	var accessToken string
	var reuseDetected bool
	var reuseUserID *uint64

	err = u.txManager.WithinTx(ctx, func(ctx context.Context, tx repository.TxRepos) error {
		// 同じRefreshTokenを二重に使わせないため、対象行をlockして取得。
		token, err := tx.RefreshToken().FindByTokenHashWithUserForUpdate(ctx, oldHash)
		if err != nil {
			if errors.Is(err, entity.ErrNotFound) {
				return entity.ErrUnauthorized
			}
			return err
		}

		// tokenに紐づくUserがログイン可能な状態か確認する。
		user = &token.User
		if err := ensureActiveUser(user); err != nil {
			return err
		}

		// RefreshToken自体が期限切れなら、AccessTokenは再発行しない。
		if !token.ExpiresAt.After(now) {
			return entity.ErrRefreshTokenExpired
		}

		// ログアウト済み・強制失効済みのRefreshTokenは使わせない。
		if token.RevokedAt != nil {
			return entity.ErrRefreshTokenRevoked
		}

		// used_at が入っているRefreshTokenが再送された場合は、reuse検知として扱う。
		// 盗難疑いがあるため、同じfamilyのRefreshTokenを失効し、既存AccessTokenも無効化する。
		if token.UsedAt != nil {
			reuseDetected = true

			userID := token.UserID
			reuseUserID = &userID

			if err := tx.RefreshToken().RevokeFamily(ctx, token.FamilyID, now); err != nil {
				return err
			}

			if err := tx.User().IncrementTokenVersion(ctx, token.UserID); err != nil {
				return err
			}

			return nil
		}

		// 正常なrotationとして、同じfamily_idで新しいRefreshTokenを作成。
		newToken = &model.RefreshToken{
			UserID:    token.UserID,
			TokenHash: newHash,
			FamilyID:  token.FamilyID,
			ExpiresAt: now.Add(u.refreshTTL),
			CreatedAt: now,
		}

		// 新しいRefreshTokenをDBに保存。
		if err := tx.RefreshToken().Create(ctx, newToken); err != nil {
			return err
		}

		// 古いRefreshTokenを使用済みにし、置き換え先のRefreshToken IDを保存。
		if err := tx.RefreshToken().MarkUsed(ctx, token.ID, now, newToken.ID); err != nil {
			return err
		}

		// AccessToken生成に失敗した場合、古いRefreshTokenの使用済み化だけが残るのを防ぐ。
		// そのため、AccessToken生成もTx内で行い、失敗時はrotation全体をrollback。
		accessToken, err = u.tokens.IssueAccessToken(ctx, user, now)
		if err != nil {
			return entity.ErrCreateFailed
		}

		return nil
	})
	if err != nil {
		return RefreshResult{}, err
	}

	// RefreshToken再利用検知時の処理。監査ログを残し、新tokenは返さずエラーにする
	if reuseDetected {
		safeAudit(ctx, u.audits, auditLog(entity.AuditActorUser, reuseUserID, entity.AuditActionRefreshReuseDetected, nil, nil, nil, meta))
		return RefreshResult{}, entity.ErrRefreshTokenReuseDetected
	}

	if user == nil || newToken == nil {
		return RefreshResult{}, entity.ErrRepositoryFailed
	}

	if accessToken == "" {
		return RefreshResult{}, entity.ErrCreateFailed
	}

	return RefreshResult{
		User:         user,
		AccessToken:  accessToken,
		RefreshToken: newPlain, //newPlainをRefreshTokenとして返す。DBにはnewHashを保存済み
	}, nil
}

// 現在端末のRefreshToken familyだけを失効します。
func (u *AuthUsecase) LogoutCurrentFamily(ctx context.Context, userID uint64, refreshToken string, meta AuditMeta) error {
	//認証済みUserIDがあるか確認
	if err := requireUserID(userID); err != nil {
		return err
	}

	//CookieにRefreshTokenがないか確認。
	if refreshToken == "" {
		return nil //ログアウト成功扱いに
	}

	//生RefreshTokenをDB検索用hashに変換
	hash, err := u.tokens.HashRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil
	}

	//DBからRefreshTokenを探す
	token, err := u.refreshTokens.FindByTokenHashWithUser(ctx, hash)
	if err != nil {
		if errors.Is(err, entity.ErrNotFound) {
			return nil
		}
		return err
	}

	//token所有者と認証Userが一致するか確認
	if token.UserID != userID {
		return entity.ErrUnauthorized
	}
	// すでに使えないtokenか確認かどうか
	if token.RevokedAt != nil || token.UsedAt != nil || !token.ExpiresAt.After(time.Now()) {
		return nil
	}
	//同じfamily_idのRefreshTokenをまとめて失効
	if err := u.refreshTokens.RevokeFamily(ctx, token.FamilyID, time.Now()); err != nil {
		return err
	}
	//ogout監査ログを作成
	safeAudit(ctx, u.audits, auditLog(entity.AuditActorUser, &userID, entity.AuditActionLogout, nil, nil, nil, meta))
	return nil
}

// 全RefreshTokenを失効し、token_versionを増やして既存AccessTokenも無効化します。
func (u *AuthUsecase) LogoutAllDevices(ctx context.Context, userID uint64, meta AuditMeta) error {
	//認証済みUserIDを確認
	if err := requireUserID(userID); err != nil {
		return err
	}
	now := time.Now()
	//DB更新をTxでまとめる
	if err := u.txManager.WithinTx(ctx, func(ctx context.Context, tx repository.TxRepos) error {
		//そのUserの全RefreshTokenを失効
		if err := tx.RefreshToken().RevokeByUserID(ctx, userID, now); err != nil {
			return err
		}
		//既存AccessTokenも無効化
		return tx.User().IncrementTokenVersion(ctx, userID)
	}); err != nil {
		return err
	}
	detail := "all_devices"
	//全端末logoutとして監査ログ作成
	safeAudit(ctx, u.audits, auditLog(entity.AuditActorUser, &userID, entity.AuditActionLogout, nil, nil, &detail, meta))
	return nil
}

// 認証済みUserIDから自分のUser情報を取得。
func (u *AuthUsecase) Me(ctx context.Context, userID uint64) (*model.User, error) {
	//認証済みUserIDがあるか確認
	if err := requireUserID(userID); err != nil {
		return nil, err
	}
	//DBからUserを取得
	user, err := u.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	//Userが利用可能か確認
	if err := ensureActiveUser(user); err != nil {
		return nil, err
	}
	return user, nil //User情報を返す。
}

// Userが存在し、active状態であることを確認。
func ensureActiveUser(user *model.User) error {
	if user == nil {
		return entity.ErrUnauthorized
	}
	switch user.Status {
	case entity.UserStatusActive:
		return nil
	case entity.UserStatusSuspended:
		return entity.ErrUserSuspended
	case entity.UserStatusDeleted:
		return entity.ErrUserDeleted
	default:
		return entity.ErrUnauthorized
	}
}
