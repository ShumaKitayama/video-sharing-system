/* ===================================================
 * pages/LoginPage.tsx
 * ログインページ（モーダルから独立した専用画面）
 * =================================================== */

import { useEffect, useState } from "react";
import { Link, useNavigate, useOutletContext } from "react-router-dom";
import type { useAuth } from "../hooks/useAuth";

type AuthContext = ReturnType<typeof useAuth>;

export default function LoginPage() {
  const { user, login } = useOutletContext<AuthContext>();
  const navigate = useNavigate();

  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (user) navigate("/", { replace: true });
  }, [user, navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim()) {
      setError("ユーザー名を入力してください");
      return;
    }
    if (!password) {
      setError("パスワードを入力してください");
      return;
    }

    setError("");
    setSubmitting(true);
    const success = await login(username.trim(), password);
    setSubmitting(false);

    if (success) {
      navigate("/", { replace: true });
    } else {
      setError("ユーザー名またはパスワードが正しくありません");
    }
  };

  return (
    <div className="auth-page" id="login-page">
      <div className="auth-page__card">
        <h1 className="auth-page__title">ログイン</h1>
        <p className="auth-page__lead">アカウント情報を入力してください。</p>

        <form className="login-form auth-page__form" onSubmit={handleSubmit}>
          {error && <div className="login-form__error">{error}</div>}

          <div>
            <label className="upload-form__label" htmlFor="login-page-username">
              ユーザー名
            </label>
            <input
              className="upload-form__input"
              id="login-page-username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="例: student01"
              autoComplete="username"
              autoFocus
              disabled={submitting}
            />
          </div>

          <div>
            <label className="upload-form__label" htmlFor="login-page-password">
              パスワード
            </label>
            <input
              className="upload-form__input"
              id="login-page-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="パスワード"
              autoComplete="current-password"
              disabled={submitting}
            />
          </div>

          <button
            className="login-form__submit auth-page__submit"
            type="submit"
            id="login-page-submit"
            disabled={submitting}
          >
            {submitting ? "ログイン中..." : "ログイン"}
          </button>
        </form>

        <p className="auth-page__footer">
          アカウントをお持ちでないですか？{" "}
          <Link className="auth-page__link" to="/register">
            新規登録
          </Link>
        </p>
      </div>
    </div>
  );
}
