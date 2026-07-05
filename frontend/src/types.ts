export type ContentType = "bean" | "article";
export type RoastLevel = "light" | "medium" | "dark";
export type Placement =
  | "top"
  | "search_result"
  | "bean_detail"
  | "article_detail"
  | "related_article"
  | "related_bean"
  | "modal"
  | "saved_list";

export type ModalTrigger =
  | "first_visit"
  | "scroll_end"
  | "bean_stay"
  | "article_stay"
  | "same_origin_viewed"
  | "same_roast_clicked"
  | "saved_content"
  | "good_rating"
  | "re_search";

export type ModalDisplayLog = {
  id: number;
  user_id?: number;
  guest_session_id?: number;
  rank_target_id: number;
  trigger: ModalTrigger;
  page_path: string;
  shown_at: string;
  clicked_at?: string;
  closed_at?: string;
  created_at: string;
};

export type EventType =
  | "content_view"
  | "impression"
  | "stay"
  | "click"
  | "re_search";
export type RatingScore = 1 | -1;
export type RankTargetID = number;
export type FeedItemKey = string;
export type FeedFilter = "all" | "bean" | "article";
export type SortKey = "score" | "newest" | "popular";
export type AppView = "feed" | "detail" | "search" | "account" | "admin";
export type DetailReturnView = "feed" | "search" | "account";

export type ApiErrorBody = {
  code: string;
  message: string;
};

export type User = {
  id: number;
  name: string;
  email: string;
  role: "user" | "admin";
  status: "active" | "suspended" | "deleted";
  created_at?: string;
  updated_at?: string;
};

export type AuthResponse = {
  user: User;
  access_token: string;
};

export type CSRFResponse = {
  csrf_token: string;
};

export type GuestSessionResponse = {
  id: number;
  created: boolean;
  expires_at: string;
};

export type Bean = {
  id: number;
  name: string;
  roaster?: string;
  origin?: string;
  region?: string;
  farm?: string;
  variety?: string;
  roast_level: RoastLevel;
  acidity?: number;
  bitterness?: number;
  flavor?: number;
  aroma?: number;
  body?: number;
  flavor_note?: string;
  description?: string;
  image_url?: string;
  is_published: boolean;
  created_at: string;
  updated_at: string;
};

export type Article = {
  id: number;
  title: string;
  slug: string;
  summary: string;
  body?: string;
  category?: string;
  source_name?: string;
  source_url?: string;
  image_url?: string;
  is_published: boolean;
  published_at?: string;
  created_at: string;
  updated_at: string;
};

export type ContentMetric = {
  id: number;
  rank_target_id: number;
  score: number;
  impression_count: number;
  content_view_count: number;
  click_count: number;
  stay_total_ms: number;
  save_count: number;
  rating_count: number;
  good_count: number;
  bad_count: number;
  re_search_count: number;
  rating_avg: number;
  good_rate: number;
  bad_rate: number;
  modal_impression_count: number;
  modal_click_count: number;
  modal_close_count: number;
  click_rate: number;
  save_rate: number;
  re_search_rate: number;
  modal_click_rate: number;
  modal_close_rate: number;
  period_start: string;
  period_end: string;
  calculated_at: string;
  updated_at: string;
};

export type RecommendationReason = {
  dimension: string;
  value: string;
  score: number;
  message: string;
};

export type RecommendationItem = {
  rank_target_id: number;
  content_type: ContentType;
  content_id: number;
  score: number;
  base_score: number;
  interest_score: number;
  reasons: RecommendationReason[] | null;
  metric: ContentMetric;
};

export type RankTarget = {
  id: number;
  content_type: ContentType;
  content_id: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type RankingResult = {
  metrics: ContentMetric[];
  targets: RankTarget[];
  beans: Bean[];
  articles: Article[];
};

export type SavedItem = {
  id: number;
  user_id: number;
  rank_target_id: number;
  removed_at?: string;
  created_at: string;
  updated_at: string;
};

export type Rating = {
  id: number;
  user_id: number;
  rank_target_id: number;
  score: RatingScore;
  created_at: string;
  updated_at: string;
};

export type BatchRun = {
  id: number;
  job_name: string;
  status: "running" | "success" | "failed";
  started_at: string;
  finished_at?: string;
  rows_processed: number;
  error_message?: string;
  triggered_by: "user" | "admin" | "system";
  triggered_user_id?: number;
};

export type AuditLog = {
  id: number;
  actor_type: "user" | "admin" | "system";
  actor_user_id?: number;
  action: string;
  target_type?: string;
  target_id?: number;
  detail?: string;
  request_id?: string;
  created_at: string;
};

export type AdminPanel =
  | "dashboard"
  | "beans"
  | "articles"
  | "relations"
  | "batches"
  | "audit";

export type AdminBeanInput = {
  name: string;
  roaster?: string;
  origin?: string;
  region?: string;
  farm?: string;
  variety?: string;
  roast_level: RoastLevel;
  acidity?: number;
  bitterness?: number;
  flavor?: number;
  aroma?: number;
  body?: number;
  flavor_note?: string;
  description?: string;
  image_url?: string;
  is_published: boolean;
};

export type AdminArticleInput = {
  title: string;
  slug: string;
  summary: string;
  body?: string;
  category?: "brewing" | "roast" | "beans" | "recipe";
  source_name?: string;
  source_url?: string;
  image_url?: string;
  is_published: boolean;
};

export type FeedItem = {
  key: FeedItemKey;
  contentType: ContentType;
  contentId: number;
  rankTargetId?: RankTargetID;
  title: string;
  subtitle: string;
  summary: string;
  body?: string;
  imageUrl?: string;
  badge: string;
  score?: number;
  reasons: RecommendationReason[];
  metric?: ContentMetric;
  bean?: Bean;
  article?: Article;
  isSaved?: boolean;
  ratingScore?: RatingScore | null;
};

export type FeedState = {
  items: FeedItem[];
  catalogItems: FeedItem[];
  loading: boolean;
  error: string | null;
};

export type SearchState = {
  q: string;
  sort: SortKey;
  contentType: FeedFilter;
  roastLevel: "" | RoastLevel;
  category: "" | "brewing" | "roast" | "beans" | "recipe";
};

export type Notice = {
  tone: "info" | "success" | "error";
  message: string;
};
