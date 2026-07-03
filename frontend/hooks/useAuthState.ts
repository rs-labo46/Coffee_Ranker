import { useCallback, useEffect, useState } from "react";
import {
  getAccessToken,
  login,
  logout,
  me,
  refreshAuth,
  signup,
} from "../src/api/client";
import type { Notice, User } from "../src/types";

type AuthState = {
  user: User | null;
  loading: boolean;
  notice: Notice | null;
};

export function useAuthState() {
  const [state, setState] = useState<AuthState>({
    user: null,
    loading: true,
    notice: null,
  });

  useEffect(() => {
    let mounted = true;
    async function loadUser() {
      const token = getAccessToken();
      if (token === null) {
        if (mounted) {
          setState({ user: null, loading: false, notice: null });
        }
        return;
      }

      try {
        const user = await me();
        if (mounted) {
          setState({ user, loading: false, notice: null });
        }
      } catch {
        try {
          const refreshed = await refreshAuth();
          if (mounted) {
            setState({ user: refreshed.user, loading: false, notice: null });
          }
        } catch {
          if (mounted) {
            setState({ user: null, loading: false, notice: null });
          }
        }
      }
    }
    void loadUser();
    return () => {
      mounted = false;
    };
  }, []);

  const loginUser = useCallback(async (email: string, password: string) => {
    setState((current) => ({ ...current, loading: true, notice: null }));
    try {
      const response = await login(email, password);
      setState({
        user: response.user,
        loading: false,
        notice: { tone: "success", message: "ログインしました" },
      });
    } catch (error) {
      setState({
        user: null,
        loading: false,
        notice: {
          tone: "error",
          message:
            error instanceof Error ? error.message : "ログインに失敗しました",
        },
      });
    }
  }, []);

  const signupUser = useCallback(
    async (name: string, email: string, password: string) => {
      setState((current) => ({ ...current, loading: true, notice: null }));
      try {
        await signup(name, email, password);
        const response = await login(email, password);
        setState({
          user: response.user,
          loading: false,
          notice: { tone: "success", message: "登録してログインしました" },
        });
      } catch (error) {
        setState((current) => ({
          ...current,
          loading: false,
          notice: {
            tone: "error",
            message:
              error instanceof Error ? error.message : "登録に失敗しました",
          },
        }));
      }
    },
    [],
  );

  const logoutUser = useCallback(async () => {
    setState((current) => ({ ...current, loading: true, notice: null }));
    try {
      await logout();
      setState({
        user: null,
        loading: false,
        notice: { tone: "info", message: "ログアウトしました" },
      });
    } catch {
      setState({
        user: null,
        loading: false,
        notice: { tone: "info", message: "ログアウト状態に戻しました" },
      });
    }
  }, []);

  return {
    ...state,
    loginUser,
    signupUser,
    logoutUser,
    clearNotice: () => setState((current) => ({ ...current, notice: null })),
  };
}
