package db

import "testing"

// DB接続に必須の環境変数不足を起動前に検知できることを確認する。
// DB接続エラーを曖昧なpanicにせず、どの設定が足りないかを明示するためのテスト。
func TestValidateEnvRequiresPostgresValues(t *testing.T) {
	setRequiredPostgresEnv(t)
	t.Setenv("POSTGRES_PASSWORD", "")

	if err := validateEnv(); err == nil {
		t.Fatal("expected missing env error")
	}
}

// 必須環境変数が揃っていればvalidateEnvが成功することを確認する。
// 実DBへ接続せず、起動設定の最低限だけを単体テストで固定する。
func TestValidateEnvAllowsCompletePostgresValues(t *testing.T) {
	setRequiredPostgresEnv(t)

	if err := validateEnv(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// 任意環境変数が空の場合にfallbackを使うことを確認する。
// POSTGRES_SSLMODEなどの任意値で空文字がそのままDSNに入るのを防ぐ。
func TestEnvReturnsFallbackForBlankOptionalValue(t *testing.T) {
	t.Setenv("POSTGRES_SSLMODE", "")
	if got := env("POSTGRES_SSLMODE", "disable"); got != "disable" {
		t.Fatalf("env fallback = %q", got)
	}

	t.Setenv("POSTGRES_SSLMODE", "require")
	if got := env("POSTGRES_SSLMODE", "disable"); got != "require" {
		t.Fatalf("env value = %q", got)
	}
}

// åDB環境変数テストで必要な共通値を設定する。
func setRequiredPostgresEnv(t *testing.T) {
	t.Helper()
	t.Setenv("POSTGRES_USER", "coffee")
	t.Setenv("POSTGRES_PASSWORD", "password")
	t.Setenv("POSTGRES_HOST", "localhost")
	t.Setenv("POSTGRES_PORT", "5432")
	t.Setenv("POSTGRES_DB", "coffee_ranker")
}
