/* ===================================================
 * hooks/useAuth.ts
 * [保護] 認証状態を管理するカスタムフック
 * ページ読み込み時に GET /auth/me でセッション復元し、
 * login / logout はバックエンドの実エンドポイントを呼ぶ。
 * =================================================== */

import { useState, useEffect, useCallback } from "react";
import type { User } from "../types";
import { API_BASE_URL, endpoints } from "../api/endpoints";
import { authService, ApiError, type RegisterInput } from "../services/api";

interface UseAuthResult {
  user: User | null;
  isLoggedIn: boolean;
  loading: boolean;
  login: (username: string, password: string) => Promise<boolean>;
  register: (input: RegisterInput) => Promise<{ success: true } | { success: false; message: string; fieldErrors?: Record<string, string> }>;
  logout: () => Promise<void>;
}

export function useAuth(): UseAuthResult {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  // ページ読み込み時にセッションを復元する
  useEffect(() => {
    fetch(`${API_BASE_URL}${endpoints.auth.me}`, {
      credentials: "include",
    })
      .then((res) => {
        if (!res.ok) return null;
        return res.json();
      })
      .then((body) => {
        if (body?.data) setUser(body.data as User);
      })
      .catch(() => {
        // 未ログインや通信エラーは無視して未ログイン状態のまま
      })
      .finally(() => setLoading(false));
  }, []);

  /** ユーザー名とパスワードでログインする */
  const login = useCallback(
    async (username: string, password: string): Promise<boolean> => {
      try {
        const res = await fetch(`${API_BASE_URL}${endpoints.auth.login}`, {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ username, password }),
        });

        if (!res.ok) return false;

        const body = await res.json();
        if (body?.data?.user) {
          setUser(body.data.user as User);
          return true;
        }
        return false;
      } catch {
        return false;
      }
    },
    [],
  );

  /** 新規アカウント登録する（成功時は自動ログイン） */
  const register = useCallback(
    async (
      input: RegisterInput,
    ): Promise<
      | { success: true }
      | { success: false; message: string; fieldErrors?: Record<string, string> }
    > => {
      try {
        const registeredUser = await authService.register(input);
        setUser(registeredUser as User);
        return { success: true };
      } catch (e) {
        if (e instanceof ApiError) {
          const fieldErrors: Record<string, string> = {};
          if (e.details) {
            for (const d of e.details) {
              if (d.field && d.message) fieldErrors[d.field] = d.message;
            }
          }

          const message =
            e.status === 409
              ? "このユーザー名はすでに使われています"
              : e.status === 400
                ? "入力内容を確認してください"
                : "登録に失敗しました";

          return { success: false, message, fieldErrors };
        }
        return { success: false, message: "サーバーに接続できませんでした" };
      }
    },
    [],
  );

  /** ログアウトする */
  const logout = useCallback(async (): Promise<void> => {
    try {
      await fetch(`${API_BASE_URL}${endpoints.auth.logout}`, {
        method: "POST",
        credentials: "include",
      });
    } catch {
      // エラーでも状態はクリアする
    } finally {
      setUser(null);
    }
  }, []);

  return {
    user,
    isLoggedIn: user !== null,
    loading,
    login,
    register,
    logout,
  };
}
