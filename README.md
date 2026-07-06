# Coffee Ranker

Coffee Rankerは、コーヒー豆と記事を閲覧、検索、保存、評価できるWebアプリです。  
閲覧、検索、保存、評価、モーダル操作などの行動ログを蓄積し、そのデータをランキングや推薦に使います。

## 1. アプリの目的

Coffee Rankerは、単なるコーヒー豆一覧アプリではありません。  
ユーザーやゲストの行動データを集め、そのデータをもとに「どの豆・どの記事がよく見られ、保存され、評価されているか」をランキングや推薦に反映することを目的にしています。

| 観点         | 内容                                                      |
| ------------ | --------------------------------------------------------- |
| 利用者向け   | コーヒー豆や記事を探し、保存・評価できます。              |
| ゲスト向け   | 未ログインでも閲覧、検索、推薦の一部を利用できます。      |
| 管理者向け   | Bean、Article、関連付け、バッチ、監査ログを管理できます。 |
| システム向け | 行動ログを蓄積し、ランキングや推薦に使います。            |
|  |

## 2. デプロイ・運用構成

| 領域              | 利用サービス          |
| ----------------- | --------------------- |
| Frontend          | Vercel                |
| Backend           | Render                |
| Database          | PostgreSQL            |
| Cache / RateLimit | Redis互換のキャッシュ |
| Source Control    | GitHub                |
| CI                | GitHub Actions        |

GitHub Actionsでは、`main`ブランチと`develop`ブランチへのpush、またはpull requestを対象に、BackendとFrontendの品質確認を実行します。  
デプロイは、VercelとRenderのGit連携を前提にしています。

## 3. 主な機能

| 区分           | 内容                                                                                            |
| -------------- | ----------------------------------------------------------------------------------------------- |
| 認証           | ユーザー登録、ログイン、ログアウト、全端末ログアウト、ログイン中ユーザー取得                    |
| ゲスト利用     | ゲストセッションCookieによる未ログイン利用                                                      |
| コンテンツ閲覧 | 公開済みコーヒー豆一覧、豆詳細、記事一覧、記事詳細、関連コンテンツ                              |
| 検索           | コーヒー豆検索、記事検索                                                                        |
| 行動ログ       | impression、content_view、stay、click、save、rating、re_search、modal系イベント                 |
| 保存           | 認証済みユーザーの保存、保存解除、保存一覧、保存済み確認                                        |
| 評価           | 認証済みユーザーのGood/Bad評価、評価取得、評価削除                                              |
| ランキング     | 行動データをもとにしたランキング取得、上位ランキング取得                                        |
| 推薦           | 行動データと興味プロフィールを使った推薦候補の取得                                              |
| 推薦モーダル   | 表示、クリック、閉じる操作の記録と抑制                                                          |
| 管理機能       | Bean/Article管理、関連付け管理、バッチ実行、監査ログ確認、RateLimitリセット、期限切れデータ削除 |

## 4. 技術構成

| 領域              | 使用技術                                 |
| ----------------- | ---------------------------------------- |
| Backend           | Go 1.26.2、Echo、GORM                    |
| Database          | PostgreSQL 16                            |
| Cache / RateLimit | Redis 7                                  |
| Frontend          | React 19、Vite、TypeScript、Tailwind CSS |
| Test              | Go標準testing、Vitest、Testing Library   |
| API仕様           | OpenAPI 3.0.3                            |
| CI                | GitHub Actions                           |

## 5. アーキテクチャ

BackendはClean Architectureベースで構成しています。  
HTTP、DB、Redisなどの外部事情をUsecaseへ直接混ぜず、責務ごとに層を分けています。

| 層         | 主な責務                                                                      |
| ---------- | ----------------------------------------------------------------------------- |
| entity     | enum、ドメインエラー、業務上の固定値を定義します。                            |
| model      | GORMモデルとしてDBテーブル構造を表現します。                                  |
| repository | PostgreSQL、Redisへの読み書きを担当します。                                   |
| usecase    | 業務ルール、トランザクション、ランキング、推薦、認証ロジックを担当します。    |
| controller | HTTPリクエストを受け取り、Usecaseへ渡し、HTTPレスポンスへ変換します。         |
| router     | URL、Controller、Middlewareの対応関係を定義します。                           |
| middleware | 認証、CSRF、RateLimit、CORS、ログ、Request ID、Security Headersを担当します。 |
| validator  | リクエスト値の形式、必須項目、enum、危険な文字列を検証します。                |

依存関係は、ControllerがUsecaseを呼び、UsecaseがRepository interfaceへ依存する形を基本にしています。  
UsecaseはEchoやHTTPレスポンス形式を知りません。

## 6. ディレクトリ構成

```txt
.
├── backend
│   ├── controller     # HTTPリクエストとレスポンスの変換
│   ├── db             # PostgreSQL接続
│   ├── document       # OpenAPI、ER図、仕様書
│   ├── e2e            # HTTP経由のE2Eテスト
│   ├── entity         # enum、ドメインエラー
│   ├── middleware     # 認証、CSRF、RateLimit、CORS、ログなど
│   ├── migrate        # AutoMigrateとSeed
│   ├── model          # GORMモデル
│   ├── repository     # DB/Redisアクセス
│   ├── router         # ルーティング定義
│   ├── usecase        # 業務ロジック
│   └── validator      # 入力値検証
├── frontend
│   ├── hooks          # React hooks
│   ├── public         # 静的ファイル
│   ├── src            # 画面、API client、型定義
│   └── tests          # フロントエンドテスト
├── .github
│   └── workflows      # CI設定
└── docker-compose.yml
```

## 7. API仕様

API仕様はOpenAPIで管理しています。

| ファイル                        | 内容    |
| ------------------------------- | ------- |
| `backend/document/openapi.yaml` | API仕様 |

主なエンドポイントは以下です。詳細なrequest、response、schemaはOpenAPIを確認してください。

| 区分            | エンドポイント例                                                                                                                          |
| --------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| Health          | `GET /health`, `GET /ready`                                                                                                               |
| CSRF            | `GET /auth/csrf`                                                                                                                          |
| Guest           | `POST /guest-session`                                                                                                                     |
| Auth            | `POST /auth/signup`, `POST /auth/login`, `POST /auth/refresh`, `POST /auth/logout`, `POST /auth/logout-all`, `GET /auth/me`               |
| Content         | `GET /beans`, `GET /beans/:id`, `GET /beans/:id/articles`, `GET /articles`, `GET /articles/:slug`, `GET /articles/id/:id`                 |
| Search          | `GET /search/beans`, `GET /search/articles`                                                                                               |
| Event           | `POST /events`                                                                                                                            |
| Saved           | `GET /saved`, `POST /saved`, `GET /saved/:rank_target_id`, `DELETE /saved/:rank_target_id`                                                |
| Rating          | `GET /ratings`, `POST /ratings`, `GET /ratings/:rank_target_id`, `DELETE /ratings/:rank_target_id`                                        |
| Ranking         | `GET /rankings`, `GET /rankings/top`                                                                                                      |
| Recommendation  | `GET /recommendations`                                                                                                                    |
| Modal           | `POST /modals`, `POST /modals/click`, `POST /modals/close`                                                                                |
| Admin           | `GET /admin/beans`, `POST /admin/beans`, `PUT /admin/beans/:id`, `GET /admin/articles`, `POST /admin/articles`, `PUT /admin/articles/:id` |
| Admin Batch     | `POST /admin/batches/ranking`, `POST /admin/batches/interest`, `GET /admin/batches`                                                       |
| Admin Operation | `GET /admin/audits`, `POST /admin/cleanup/expired`, `POST /admin/rate-limits/reset`                                                       |

## 8. 認証とセキュリティ

このリポジトリで確認できる主な対策は以下です。

| 項目             | 内容                                                                                                 |
| ---------------- | ---------------------------------------------------------------------------------------------------- |
| Password         | 生Passwordは保存せず、bcryptでハッシュ化して保存します。                                             |
| Access Token     | JWTをレスポンスJSONで返します。FrontendではsessionStorageに保持します。                              |
| Refresh Token    | 生RefreshTokenはDBに保存せず、Hashを保存します。ブラウザにはHttpOnly Cookieで渡します。              |
| Refresh Rotation | Refresh時にTokenをローテーションします。使用済みTokenの再利用検知があります。                        |
| Token Version    | 全端末ログアウトやRefresh Token再利用検知時にtoken_versionを更新し、既存Access Tokenを無効化します。 |
| CSRF             | Cookieと`X-CSRF-Token`を照合するDouble Submit方式です。                                              |
| Guest Session    | ゲスト用セッションキーはCookieで扱い、DBにはHashを保存します。                                       |
| RateLimit        | Redisを使ってエンドポイント種別ごとに制限します。                                                    |
| CORS             | 許可Originを環境変数で制御し、Cookie送信を許可します。                                               |
| Body Limit       | リクエストBodyサイズを制限します。                                                                   |
| Security Headers | セキュリティ系HTTPヘッダーを設定します。                                                             |
| Request ID       | リクエスト追跡用IDを付与します。                                                                     |
| Client IP Hash   | IPアドレスはHash化して扱います。                                                                     |

## 9. 行動ログとランキング

Coffee Rankerでは、ユーザーの行動をランキングや推薦の材料として扱います。  
単純な閲覧数だけではなく、保存、評価、滞在、再検索、モーダル操作も記録します。

| イベント         | 用途                                                 |
| ---------------- | ---------------------------------------------------- |
| impression       | 一覧やフィード上で表示されたコンテンツを記録します。 |
| content_view     | 詳細閲覧を記録します。                               |
| stay             | 滞在時間を記録します。                               |
| click            | 通常クリックを記録します。                           |
| save             | 保存操作を記録します。                               |
| rating           | Good/Bad評価を記録します。                           |
| re_search        | 再検索を記録します。                                 |
| modal_impression | 推薦モーダルの表示を記録します。                     |
| modal_click      | 推薦モーダル内のクリックを記録します。               |
| modal_close      | 推薦モーダルを閉じた操作を記録します。               |

ランキングは、これらの行動データをもとに集計されます。  
推薦では、行動ログと興味プロフィールを使い、BeanまたはArticleの候補を返します。

## 10. 管理者機能

管理者は、コンテンツ管理と運用管理を行えます。
環境変数にてメールとパスワードを入れています。

| 区分          | 内容                                                         |
| ------------- | ------------------------------------------------------------ |
| Bean管理      | 作成、更新、公開、非公開、一覧取得                           |
| Article管理   | 作成、更新、公開、非公開、一覧取得                           |
| 関連付け管理  | BeanとArticleの関連付け作成、削除、並び順更新                |
| Batch管理     | ランキング集計、興味プロフィール集計、実行履歴取得           |
| Audit管理     | 管理者操作ログの確認                                         |
| Cleanup       | 期限切れRefresh Token、Guest Session、Interest Profileの削除 |
| RateLimit管理 | 管理者によるRateLimitリセット                                |

## 11. バッチ

ランキング集計バッチは、アプリ起動中に日次で実行されます。  
デフォルトのタイムゾーンは`Asia/Tokyo`です。

管理者APIから、ランキング集計と興味プロフィール集計を手動実行できます。

| バッチ         | 内容                                                                   |
| -------------- | ---------------------------------------------------------------------- |
| Ranking Batch  | 行動ログを集計し、ランキング指標を更新します。                         |
| Interest Batch | 行動ログをもとに、ユーザーまたはゲストの興味プロフィールを更新します。 |
| Cleanup        | 期限切れデータを削除します。                                           |

## 12. テストとCI

テストはBackendとFrontendに分かれています。  
GitHub Actionsでは、BackendとFrontendの品質確認を行います。

| 対象     | CIで確認する内容                                      |
| -------- | ----------------------------------------------------- |
| Backend  | `gofmt`、`go vet`、`go test ./...`                    |
| Frontend | `npm ci`、`npm run lint`、`npm test`、`npm run build` |

E2Eテストは`backend/e2e`配下にあります。  
API、PostgreSQL、Redisが起動している状態で、HTTP経由の主要フローを検証するためのテストです。

## 13. 仕様書

仕様確認用の資料は`backend/document`配下にあります。

| ファイル                                        | 内容                                                                             |
| ----------------------------------------------- | -------------------------------------------------------------------------------- |
| `backend/document/openapi.yaml`                 | API仕様                                                                          |
| `backend/document/ER図.pdf`                     | ER図                                                                             |
| `backend/document/コーヒーランカー要件定義.pdf` | 要件定義                                                                         |
| `backend/document/仕様書/*.pdf`                 | Controller、DB、Entity、Middleware、Repository、Router、Usecase、Validatorの仕様 |

## 14. 現在の注意点

現状のコードに合わせた注意点です。  
本番運用やレビュー時に見落とすと危ない点だけを記載しています。

| 注意点                                       | 理由                                                               | 対応方針                                                                            |
| -------------------------------------------- | ------------------------------------------------------------------ | ----------------------------------------------------------------------------------- |
| `AutoMigrate`と`Seed`が起動時に実行されます  | 本番起動時に意図しないDB変更やSeed更新が起きる可能性があります。   | 本番では環境変数で明示的に制御するか、migration jobとして分離する必要があります。   |
| Access TokenをsessionStorageに保持しています | XSSが発生した場合、Access Tokenを読み取られる可能性があります。    | 入力検証、Security Headers、XSS対策を維持しつつ、必要に応じてCookie化を検討します。 |
| GitHub ActionsはCI中心です                   | 現在のworkflowではBackend/Frontendのテストとビルド確認が中心です。 | デプロイはVercel/RenderのGit連携で管理します。                                      |
| バッチはアプリ起動中にスケジュールされます   | 複数インスタンス運用では二重実行リスクがあります。                 | BatchLockで制御しつつ、運用規模に応じて外部スケジューラ化を検討します。             |
