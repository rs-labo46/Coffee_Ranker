package usecase

import (
	"coffee-ranker/entity"
	"coffee-ranker/model"
	"coffee-ranker/repository"
	"context"
	"errors"
	"time"
)

// GuestSession keyの生成とhash化をUsecaseから呼ぶためのinterface。
type GuestKeyManager interface {
	NewGuestSessionKey(ctx context.Context) (plain string, hash string, err error)
	HashGuestSessionKey(ctx context.Context, key string) (string, error)
}

// 未ログインを一時的に識別する
type GuestSessionUsecase struct {
	sessions repository.GuestSessionRepository
	keys     GuestKeyManager
	ttl      time.Duration
}

// クライアントへ返すsessionKey
type GuestSessionResult struct {
	Session    *model.GuestSession
	SessionKey string
	Created    bool
}

// ゲストセッションの管理に必要なRepository、key生成、TTLの受け取り
func NewGuestSessionUsecase(sessions repository.GuestSessionRepository, keys GuestKeyManager, ttl time.Duration) *GuestSessionUsecase {
	return &GuestSessionUsecase{sessions: sessions, keys: keys, ttl: ttl}
}

// sessionKeyからGuestSessionを取得または新規作成する。
// 有効な既存Sessionがあれば最終アクセス日時と期限を更新し、存在しなければ新規作成する。
func (u *GuestSessionUsecase) GetOrCreateGuestSession(ctx context.Context, sessionKey string) (GuestSessionResult, error) {
	if sessionKey != "" {
		hash, err := u.keys.HashGuestSessionKey(ctx, sessionKey)
		if err != nil {
			return GuestSessionResult{}, entity.ErrInvalidInput
		}

		session, err := u.sessions.FindBySessionKeyHash(ctx, hash)
		if err == nil {
			touched, err := u.touchValidSession(ctx, session)
			if err != nil {
				return GuestSessionResult{}, err
			}

			return GuestSessionResult{
				Session:    touched,
				SessionKey: sessionKey,
				Created:    false,
			}, nil
		}

		if !errors.Is(err, entity.ErrNotFound) {
			return GuestSessionResult{}, err
		}
	}

	now := time.Now()
	plain, hash, err := u.keys.NewGuestSessionKey(ctx)
	if err != nil {
		return GuestSessionResult{}, entity.ErrCreateFailed
	}

	session := &model.GuestSession{
		SessionKeyHash: hash,
		FirstSeenAt:    now,
		LastSeenAt:     now,
		ExpiresAt:      now.Add(u.ttl),
	}

	if err := u.sessions.Create(ctx, session); err != nil {
		return GuestSessionResult{}, err
	}

	return GuestSessionResult{
		Session:    session,
		SessionKey: plain,
		Created:    true,
	}, nil
}

// 既存GuestSessionの最終アクセス日時と期限を更新する。
func (u *GuestSessionUsecase) TouchGuestSession(ctx context.Context, guestSessionID uint64) (*model.GuestSession, error) {
	if guestSessionID == 0 {
		return nil, entity.ErrInvalidInput
	}

	session, err := u.sessions.FindByID(ctx, guestSessionID)
	if err != nil {
		return nil, err
	}

	return u.touchValidSession(ctx, session)
}

// 有効なGuestSessionだけ最終アクセス日時と期限を更新する。
// 期限切れの場合は更新せずエラーを返す。
func (u *GuestSessionUsecase) touchValidSession(ctx context.Context, session *model.GuestSession) (*model.GuestSession, error) {
	now := time.Now()

	if !session.ExpiresAt.After(now) {
		return nil, entity.ErrGuestSessionNotFound
	}

	expiresAt := now.Add(u.ttl)

	if err := u.sessions.Touch(ctx, session.ID, now, expiresAt); err != nil {
		return nil, err
	}

	session.LastSeenAt = now
	session.ExpiresAt = expiresAt

	return session, nil
}
