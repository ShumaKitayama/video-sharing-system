/* ===================================================
 * hooks/useAuth.ts
 * [保護] 認証状態を管理するカスタムフック
 * =================================================== */

import { useState, useCallback } from "react";
import type { User } from "../types";
import { mockUsers } from "../mock/data";

interface UseAuthResult {
  user: User | null;
  isLoggedIn: boolean;
  login: (username: string) => boolean;
  logout: () => void;
}

export function useAuth(): UseAuthResult {
  const [user, setUser] = useState<User | null>(null);

  const login = useCallback((username: string): boolean => {
    // モック: usernameに一致するユーザーでログインする
    const found = mockUsers.find((u) => u.username === username);
    if (found) {
      setUser(found);
      return true;
    }
    return false;
  }, []);

  const logout = useCallback(() => {
    setUser(null);
  }, []);

  return {
    user,
    isLoggedIn: user !== null,
    login,
    logout,
  };
}
