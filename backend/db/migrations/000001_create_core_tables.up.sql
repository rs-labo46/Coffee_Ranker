-- DBテーブルを作成。
-- 生パスワード、生RefreshToken、生ゲストセッションキーは列として持たず、hash列だけを保存する。

CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL CHECK (name <> ''),
    email TEXT NOT NULL CHECK (email <> ''),
    password_hash TEXT NOT NULL CHECK (password_hash <> ''),
    role TEXT NOT NULL,
    status TEXT NOT NULL,
    token_version INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_users_role CHECK (role IN ('user', 'admin')),
    CONSTRAINT chk_users_status CHECK (status IN ('active', 'suspended', 'deleted'))
);

CREATE UNIQUE INDEX uq_users_email ON users (email);
CREATE INDEX idx_users_role ON users (role);
CREATE INDEX idx_users_status ON users (status);

CREATE TABLE refresh_tokens (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON UPDATE CASCADE,
    token_hash TEXT NOT NULL CHECK (token_hash <> ''),
    family_id TEXT NOT NULL CHECK (family_id <> ''),
    used_at TIMESTAMPTZ,
    replaced_by_token_id BIGINT REFERENCES refresh_tokens(id) ON UPDATE CASCADE,
    revoked_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE UNIQUE INDEX uq_refresh_tokens_token_hash ON refresh_tokens (token_hash);
CREATE INDEX idx_refresh_tokens_family_id ON refresh_tokens (family_id);
CREATE INDEX idx_refresh_tokens_used_at ON refresh_tokens (used_at);
CREATE INDEX idx_refresh_tokens_replaced_by_token_id ON refresh_tokens (replaced_by_token_id);
CREATE INDEX idx_refresh_tokens_revoked_at ON refresh_tokens (revoked_at);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);

CREATE TABLE guest_sessions (
    id BIGSERIAL PRIMARY KEY,
    session_key_hash TEXT NOT NULL CHECK (session_key_hash <> ''),
    first_seen_at TIMESTAMPTZ NOT NULL,
    last_seen_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE UNIQUE INDEX uq_guest_sessions_session_key_hash ON guest_sessions (session_key_hash);
CREATE INDEX idx_guest_sessions_last_seen_at ON guest_sessions (last_seen_at);
CREATE INDEX idx_guest_sessions_expires_at ON guest_sessions (expires_at);

CREATE TABLE beans (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL CHECK (name <> ''),
    roaster TEXT,
    origin TEXT,
    region TEXT,
    farm TEXT,
    variety TEXT,
    roast_level TEXT NOT NULL,
    acidity INTEGER,
    bitterness INTEGER,
    flavor INTEGER,
    aroma INTEGER,
    body INTEGER,
    flavor_note TEXT,
    description TEXT,
    image_url TEXT,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_beans_roast_level CHECK (roast_level IN ('light', 'medium', 'dark')),
    CONSTRAINT chk_beans_acidity CHECK (acidity IS NULL OR acidity BETWEEN 1 AND 5),
    CONSTRAINT chk_beans_bitterness CHECK (bitterness IS NULL OR bitterness BETWEEN 1 AND 5),
    CONSTRAINT chk_beans_flavor CHECK (flavor IS NULL OR flavor BETWEEN 1 AND 5),
    CONSTRAINT chk_beans_aroma CHECK (aroma IS NULL OR aroma BETWEEN 1 AND 5),
    CONSTRAINT chk_beans_body CHECK (body IS NULL OR body BETWEEN 1 AND 5)
);

CREATE INDEX idx_beans_name ON beans (name);
CREATE INDEX idx_beans_roaster ON beans (roaster);
CREATE INDEX idx_beans_origin ON beans (origin);
CREATE INDEX idx_beans_roast_level ON beans (roast_level);
CREATE INDEX idx_beans_is_published ON beans (is_published);

CREATE TABLE articles (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL CHECK (title <> ''),
    slug TEXT NOT NULL CHECK (slug <> ''),
    summary TEXT NOT NULL CHECK (summary <> ''),
    body TEXT,
    category TEXT,
    source_name TEXT,
    source_url TEXT,
    image_url TEXT,
    is_published BOOLEAN NOT NULL DEFAULT FALSE,
    published_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_articles_title ON articles (title);
CREATE UNIQUE INDEX uq_articles_slug ON articles (slug);
CREATE INDEX idx_articles_category ON articles (category);
CREATE INDEX idx_articles_is_published ON articles (is_published);
CREATE INDEX idx_articles_published_at ON articles (published_at);

CREATE TABLE bean_articles (
    id BIGSERIAL PRIMARY KEY,
    bean_id BIGINT NOT NULL REFERENCES beans(id) ON UPDATE CASCADE,
    article_id BIGINT NOT NULL REFERENCES articles(id) ON UPDATE CASCADE,
    display_order INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bean_articles_bean_id ON bean_articles (bean_id);
CREATE INDEX idx_bean_articles_article_id ON bean_articles (article_id);
CREATE INDEX idx_bean_articles_display_order ON bean_articles (display_order);
CREATE UNIQUE INDEX uq_bean_articles_bean_article ON bean_articles (bean_id, article_id);

CREATE TABLE rank_targets (
    id BIGSERIAL PRIMARY KEY,
    content_type TEXT NOT NULL,
    content_id BIGINT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_rank_targets_content_type CHECK (content_type IN ('bean', 'article'))
);

CREATE INDEX idx_rank_targets_content_type ON rank_targets (content_type);
CREATE INDEX idx_rank_targets_is_active ON rank_targets (is_active);
CREATE UNIQUE INDEX uq_rank_targets_content ON rank_targets (content_type, content_id);

CREATE TABLE modal_display_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON UPDATE CASCADE,
    guest_session_id BIGINT REFERENCES guest_sessions(id) ON UPDATE CASCADE,
    rank_target_id BIGINT NOT NULL REFERENCES rank_targets(id) ON UPDATE CASCADE,
    trigger TEXT NOT NULL,
    page_path TEXT NOT NULL CHECK (page_path <> ''),
    shown_at TIMESTAMPTZ NOT NULL,
    clicked_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_modal_display_logs_actor CHECK (
        (user_id IS NOT NULL AND guest_session_id IS NULL)
        OR (user_id IS NULL AND guest_session_id IS NOT NULL)
    ),
    CONSTRAINT chk_modal_display_logs_trigger CHECK (
        trigger IN (
            'first_visit',
            'scroll_end',
            'bean_stay',
            'article_stay',
            'same_origin_viewed',
            'same_roast_clicked',
            'saved_content',
            'good_rating',
            're_search'
        )
    )
);

CREATE INDEX idx_modal_display_logs_user_id ON modal_display_logs (user_id);
CREATE INDEX idx_modal_display_logs_guest_session_id ON modal_display_logs (guest_session_id);
CREATE INDEX idx_modal_display_logs_rank_target_id ON modal_display_logs (rank_target_id);
CREATE INDEX idx_modal_display_logs_trigger ON modal_display_logs (trigger);
CREATE INDEX idx_modal_display_logs_shown_at ON modal_display_logs (shown_at);

CREATE TABLE modal_block_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON UPDATE CASCADE,
    guest_session_id BIGINT REFERENCES guest_sessions(id) ON UPDATE CASCADE,
    candidate_rank_target_id BIGINT REFERENCES rank_targets(id) ON UPDATE CASCADE,
    reason TEXT NOT NULL,
    page_path TEXT NOT NULL CHECK (page_path <> ''),
    blocked_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_modal_block_logs_actor CHECK (
        (user_id IS NOT NULL AND guest_session_id IS NULL)
        OR (user_id IS NULL AND guest_session_id IS NOT NULL)
    ),
    CONSTRAINT chk_modal_block_logs_reason CHECK (
        reason IN (
            'page_just_opened',
            'page_limit_reached',
            'session_limit_reached',
            'recently_shown',
            'recently_closed',
            'during_save',
            'during_rating',
            'login_modal_open',
            'already_saved',
            'no_candidate'
        )
    )
);

CREATE INDEX idx_modal_block_logs_user_id ON modal_block_logs (user_id);
CREATE INDEX idx_modal_block_logs_guest_session_id ON modal_block_logs (guest_session_id);
CREATE INDEX idx_modal_block_logs_candidate_rank_target_id ON modal_block_logs (candidate_rank_target_id);
CREATE INDEX idx_modal_block_logs_reason ON modal_block_logs (reason);
CREATE INDEX idx_modal_block_logs_blocked_at ON modal_block_logs (blocked_at);

CREATE TABLE action_events (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON UPDATE CASCADE,
    guest_session_id BIGINT REFERENCES guest_sessions(id) ON UPDATE CASCADE,
    event_type TEXT NOT NULL,
    rank_target_id BIGINT REFERENCES rank_targets(id) ON UPDATE CASCADE,
    placement TEXT NOT NULL,
    dwell_ms BIGINT,
    rating_score INTEGER,
    search_condition_hash TEXT,
    previous_condition_hash TEXT,
    search_keyword TEXT,
    search_origin TEXT,
    search_roast_level TEXT,
    search_acidity INTEGER,
    search_bitterness INTEGER,
    search_aroma INTEGER,
    search_flavor INTEGER,
    search_body INTEGER,
    search_category TEXT,
    modal_display_log_id BIGINT REFERENCES modal_display_logs(id) ON UPDATE CASCADE,
    page_path TEXT NOT NULL CHECK (page_path <> ''),
    referrer_path TEXT,
    user_agent TEXT,
    ip_address_hash TEXT,
    request_id TEXT,
    occurred_at TIMESTAMPTZ NOT NULL,
    CONSTRAINT chk_action_events_actor CHECK (
        (user_id IS NOT NULL AND guest_session_id IS NULL)
        OR (user_id IS NULL AND guest_session_id IS NOT NULL)
    ),
    CONSTRAINT chk_action_events_event_type CHECK (
        event_type IN (
            'content_view',
            'impression',
            'stay',
            'click',
            'save',
            'rating',
            're_search',
            'modal_impression',
            'modal_click',
            'modal_close'
        )
    ),
    CONSTRAINT chk_action_events_placement CHECK (
        placement IN (
            'top',
            'search_result',
            'bean_detail',
            'article_detail',
            'related_article',
            'related_bean',
            'modal',
            'saved_list'
        )
    ),
    CONSTRAINT chk_action_events_dwell_ms CHECK (
        dwell_ms IS NULL
        OR (dwell_ms >= 3000 AND dwell_ms <= 1800000)
    ),
    CONSTRAINT chk_action_events_rating_score CHECK (
        rating_score IS NULL
        OR rating_score IN (-1, 1)
    ),
    CONSTRAINT chk_action_events_search_roast_level CHECK (
        search_roast_level IS NULL
        OR search_roast_level IN ('light', 'medium', 'dark')
    ),
    CONSTRAINT chk_action_events_search_acidity CHECK (
        search_acidity IS NULL
        OR search_acidity BETWEEN 1 AND 5
    ),
    CONSTRAINT chk_action_events_search_bitterness CHECK (
        search_bitterness IS NULL
        OR search_bitterness BETWEEN 1 AND 5
    ),
    CONSTRAINT chk_action_events_search_aroma CHECK (
        search_aroma IS NULL
        OR search_aroma BETWEEN 1 AND 5
    ),
    CONSTRAINT chk_action_events_search_flavor CHECK (
        search_flavor IS NULL
        OR search_flavor BETWEEN 1 AND 5
    ),
    CONSTRAINT chk_action_events_search_body CHECK (
        search_body IS NULL
        OR search_body BETWEEN 1 AND 5
    ),
    CONSTRAINT chk_action_events_field_by_type CHECK (
        (
            event_type IN ('content_view', 'impression', 'click')
            AND rank_target_id IS NOT NULL
            AND dwell_ms IS NULL
            AND rating_score IS NULL
            AND search_condition_hash IS NULL
            AND modal_display_log_id IS NULL
        )
        OR (
            event_type = 'stay'
            AND rank_target_id IS NOT NULL
            AND dwell_ms IS NOT NULL
            AND rating_score IS NULL
            AND search_condition_hash IS NULL
            AND modal_display_log_id IS NULL
        )
        OR (
            event_type = 'rating'
            AND rank_target_id IS NOT NULL
            AND rating_score IN (-1, 1)
            AND dwell_ms IS NULL
            AND search_condition_hash IS NULL
            AND modal_display_log_id IS NULL
        )
        OR (
            event_type = 'save'
            AND rank_target_id IS NOT NULL
            AND dwell_ms IS NULL
            AND rating_score IS NULL
            AND search_condition_hash IS NULL
            AND modal_display_log_id IS NULL
        )
        OR (
            event_type = 're_search'
            AND rank_target_id IS NULL
            AND dwell_ms IS NULL
            AND rating_score IS NULL
            AND search_condition_hash IS NOT NULL
            AND modal_display_log_id IS NULL
        )
        OR (
            event_type IN ('modal_impression', 'modal_click', 'modal_close')
            AND rank_target_id IS NOT NULL
            AND dwell_ms IS NULL
            AND rating_score IS NULL
            AND search_condition_hash IS NULL
            AND modal_display_log_id IS NOT NULL
        )
    )
);

CREATE INDEX idx_action_events_user_id ON action_events (user_id);
CREATE INDEX idx_action_events_guest_session_id ON action_events (guest_session_id);
CREATE INDEX idx_action_events_event_type ON action_events (event_type);
CREATE INDEX idx_action_events_rank_target_id ON action_events (rank_target_id);
CREATE INDEX idx_action_events_placement ON action_events (placement);
CREATE INDEX idx_action_events_search_condition_hash ON action_events (search_condition_hash);
CREATE INDEX idx_action_events_modal_display_log_id ON action_events (modal_display_log_id);
CREATE INDEX idx_action_events_request_id ON action_events (request_id);
CREATE INDEX idx_action_events_occurred_at ON action_events (occurred_at);

CREATE TABLE saved_items (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON UPDATE CASCADE,
    rank_target_id BIGINT NOT NULL REFERENCES rank_targets(id) ON UPDATE CASCADE,
    removed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_saved_items_user_id ON saved_items (user_id);
CREATE INDEX idx_saved_items_rank_target_id ON saved_items (rank_target_id);
CREATE INDEX idx_saved_items_removed_at ON saved_items (removed_at);
CREATE UNIQUE INDEX uq_saved_items_user_target ON saved_items (user_id, rank_target_id);

CREATE TABLE ratings (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON UPDATE CASCADE,
    rank_target_id BIGINT NOT NULL REFERENCES rank_targets(id) ON UPDATE CASCADE,
    score INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_ratings_score CHECK (score IN (-1, 1))
);

CREATE INDEX idx_ratings_user_id ON ratings (user_id);
CREATE INDEX idx_ratings_rank_target_id ON ratings (rank_target_id);
CREATE INDEX idx_ratings_score ON ratings (score);
CREATE UNIQUE INDEX uq_ratings_user_target ON ratings (user_id, rank_target_id);

CREATE TABLE content_metrics (
    id BIGSERIAL PRIMARY KEY,
    rank_target_id BIGINT NOT NULL REFERENCES rank_targets(id) ON UPDATE CASCADE,
    score DOUBLE PRECISION NOT NULL DEFAULT 0,
    impression_count BIGINT NOT NULL DEFAULT 0,
    content_view_count BIGINT NOT NULL DEFAULT 0,
    click_count BIGINT NOT NULL DEFAULT 0,
    stay_total_ms BIGINT NOT NULL DEFAULT 0,
    save_count BIGINT NOT NULL DEFAULT 0,
    rating_count BIGINT NOT NULL DEFAULT 0,
    good_count BIGINT NOT NULL DEFAULT 0,
    bad_count BIGINT NOT NULL DEFAULT 0,
    rating_avg DOUBLE PRECISION NOT NULL DEFAULT 0,
    good_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    bad_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    modal_impression_count BIGINT NOT NULL DEFAULT 0,
    modal_click_count BIGINT NOT NULL DEFAULT 0,
    modal_close_count BIGINT NOT NULL DEFAULT 0,
    click_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    save_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    modal_click_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    modal_close_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX uq_content_metrics_rank_target ON content_metrics (rank_target_id);
CREATE INDEX idx_content_metrics_score ON content_metrics (score);
CREATE INDEX idx_content_metrics_period_start ON content_metrics (period_start);
CREATE INDEX idx_content_metrics_period_end ON content_metrics (period_end);
CREATE INDEX idx_content_metrics_calculated_at ON content_metrics (calculated_at);

CREATE TABLE interest_profiles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON UPDATE CASCADE,
    guest_session_id BIGINT REFERENCES guest_sessions(id) ON UPDATE CASCADE,
    dimension TEXT NOT NULL,
    value TEXT NOT NULL CHECK (value <> ''),
    score DOUBLE PRECISION NOT NULL DEFAULT 0,
    last_event_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_interest_profiles_actor CHECK (
        (user_id IS NOT NULL AND guest_session_id IS NULL)
        OR (user_id IS NULL AND guest_session_id IS NOT NULL)
    ),
    CONSTRAINT chk_interest_profiles_dimension CHECK (
        dimension IN (
            'origin',
            'roast_level',
            'acidity',
            'bitterness',
            'flavor',
            'aroma',
            'body',
            'article_category'
        )
    )
);

CREATE INDEX idx_interest_profiles_user_id ON interest_profiles (user_id);
CREATE INDEX idx_interest_profiles_guest_session_id ON interest_profiles (guest_session_id);
CREATE INDEX idx_interest_profiles_dimension ON interest_profiles (dimension);
CREATE INDEX idx_interest_profiles_value ON interest_profiles (value);
CREATE INDEX idx_interest_profiles_score ON interest_profiles (score);
CREATE INDEX idx_interest_profiles_last_event_at ON interest_profiles (last_event_at);
CREATE INDEX idx_interest_profiles_expires_at ON interest_profiles (expires_at);
CREATE UNIQUE INDEX uq_interest_profiles_user_dimension_value
    ON interest_profiles (user_id, dimension, value)
    WHERE user_id IS NOT NULL;
CREATE UNIQUE INDEX uq_interest_profiles_guest_dimension_value
    ON interest_profiles (guest_session_id, dimension, value)
    WHERE guest_session_id IS NOT NULL;

CREATE TABLE batch_runs (
    id BIGSERIAL PRIMARY KEY,
    job_name TEXT NOT NULL CHECK (job_name <> ''),
    status TEXT NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    rows_processed BIGINT NOT NULL DEFAULT 0,
    error_message TEXT,
    triggered_by TEXT NOT NULL,
    triggered_user_id BIGINT REFERENCES users(id) ON UPDATE CASCADE,
    CONSTRAINT chk_batch_runs_status CHECK (status IN ('running', 'success', 'failed')),
    CONSTRAINT chk_batch_runs_triggered_by CHECK (triggered_by IN ('user', 'admin', 'system'))
);

CREATE INDEX idx_batch_runs_job_name ON batch_runs (job_name);
CREATE INDEX idx_batch_runs_status ON batch_runs (status);
CREATE INDEX idx_batch_runs_started_at ON batch_runs (started_at);
CREATE INDEX idx_batch_runs_triggered_by ON batch_runs (triggered_by);
CREATE INDEX idx_batch_runs_triggered_user_id ON batch_runs (triggered_user_id);

CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    actor_type TEXT NOT NULL,
    actor_user_id BIGINT REFERENCES users(id) ON UPDATE CASCADE,
    action TEXT NOT NULL,
    target_type TEXT,
    target_id BIGINT,
    detail TEXT,
    ip_address_hash TEXT,
    request_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_audit_logs_actor_type CHECK (actor_type IN ('user', 'admin', 'system')),
    CONSTRAINT chk_audit_logs_action CHECK (
        action IN (
            'login',
            'logout',
            'refresh_reuse_detected',
            'bean_create',
            'bean_update',
            'bean_publish',
            'bean_unpublish',
            'article_create',
            'article_update',
            'article_publish',
            'article_unpublish',
            'ranking_batch_run',
            'manual_batch_run'
        )
    )
);

CREATE INDEX idx_audit_logs_actor_type ON audit_logs (actor_type);
CREATE INDEX idx_audit_logs_actor_user_id ON audit_logs (actor_user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs (action);
CREATE INDEX idx_audit_logs_target_type ON audit_logs (target_type);
CREATE INDEX idx_audit_logs_target_id ON audit_logs (target_id);
CREATE INDEX idx_audit_logs_request_id ON audit_logs (request_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);
