/* ===================================================
 * pages/RegisterPage.tsx
 * アカウント登録ページ
 * useAuth の register() を通じてバックエンドに登録する。
 * =================================================== */

import { useEffect, useState } from "react";
import { Link, useNavigate, useOutletContext } from "react-router-dom";
import type { useAuth } from "../hooks/useAuth";

type AuthContext = ReturnType<typeof useAuth>;

const USERNAME_RE = /^[a-zA-Z0-9_]{3,32}$/;

export default function RegisterPage() {
  const { user, register } = useOutletContext<AuthContext>();
  const navigate = useNavigate();

  const [username, setUsername] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [error, setError] = useState("");
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (user) navigate("/", { replace: true });
  }, [user, navigate]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setFieldErrors({});

    const trimmedUsername = username.trim();
    const trimmedDisplayName = displayName.trim();

    if (!USERNAME_RE.test(trimmedUsername)) {
      setFieldErrors({
        username: "3〜32文字の英数字とアンダースコアのみ利用できます",
      });
      return;
    }
    if (trimmedDisplayName.length < 1 || trimmedDisplayName.length > 40) {
      setFieldErrors({
        display_name: "1〜40文字で入力してください",
      });
      return;
    }
    if (password.length < 8 || password.length > 72) {
      setFieldErrors({
        password: "8〜72文字で入力してください",
      });
      return;
    }
    if (password !== passwordConfirm) {
      setFieldErrors({
        password_confirm: "パスワードが一致しません",
      });
      return;
    }

    setSubmitting(true);
    const result = await register({
      username: trimmedUsername,
      display_name: trimmedDisplayName,
      password,
    });
    setSubmitting(false);

    if (result.success) {
      navigate("/", { replace: true });
      return;
    }

    setError(result.message);
    if (result.fieldErrors) setFieldErrors(result.fieldErrors);
  };

  return (
    <div className="auth-page" id="register-page">
      <div className="auth-page__card">
        <h1 className="auth-page__title">アカウント登録</h1>
        <p className="auth-page__lead">
          ユーザー名と表示名を決めて、VideoShare に参加しましょう。
        </p>

        <form className="login-form auth-page__form" onSubmit={handleSubmit}>
          {error && <div className="login-form__error">{error}</div>}

          <div>
            <label className="upload-form__label" htmlFor="register-username">
              ユーザー名
            </label>
            <input
              className="upload-form__input"
              id="register-username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              placeholder="例: student01"
              autoComplete="username"
              autoFocus
              disabled={submitting}
            />
            {fieldErrors.username && (
              <p className="auth-page__field-error">{fieldErrors.username}</p>
            )}
            <p className="auth-page__hint">
              3〜32文字。英数字とアンダースコア（_）のみ
            </p>
          </div>

          <div>
            <label className="upload-form__label" htmlFor="register-display-name">
              表示名
            </label>
            <input
              className="upload-form__input"
              id="register-display-name"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="例: 山田 太郎"
              autoComplete="name"
              disabled={submitting}
            />
            {fieldErrors.display_name && (
              <p className="auth-page__field-error">{fieldErrors.display_name}</p>
            )}
          </div>

          <div>
            <label className="upload-form__label" htmlFor="register-password">
              パスワード
            </label>
            <input
              className="upload-form__input"
              id="register-password"
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="8文字以上"
              autoComplete="new-password"
              disabled={submitting}
            />
            {fieldErrors.password && (
              <p className="auth-page__field-error">{fieldErrors.password}</p>
            )}
          </div>

          <div>
            <label className="upload-form__label" htmlFor="register-password-confirm">
              パスワード（確認）
            </label>
            <input
              className="upload-form__input"
              id="register-password-confirm"
              type="password"
              value={passwordConfirm}
              onChange={(e) => setPasswordConfirm(e.target.value)}
              placeholder="もう一度入力"
              autoComplete="new-password"
              disabled={submitting}
            />
            {fieldErrors.password_confirm && (
              <p className="auth-page__field-error">{fieldErrors.password_confirm}</p>
            )}
          </div>

          <button
            className="login-form__submit auth-page__submit"
            type="submit"
            id="register-submit"
            disabled={submitting}
          >
            {submitting ? "登録中..." : "アカウントを作成"}
          </button>
        </form>

        <p className="auth-page__footer">
          すでにアカウントをお持ちですか？{" "}
          <Link className="auth-page__link" to="/login">
            ログイン
          </Link>
        </p>
      </div>
    </div>
  );
}
