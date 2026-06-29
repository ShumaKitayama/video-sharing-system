/* ===================================================
 * components/internal/LoginModal.tsx
 * [保護] ログインモーダル
 * useAuth の login(username, password) を呼び出す。
 * =================================================== */

import { useState } from "react";

interface LoginModalProps {
  onClose: () => void;
  onLogin: (username: string, password: string) => Promise<boolean>;
}

export default function LoginModal({ onClose, onLogin }: LoginModalProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

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
    const success = await onLogin(username.trim(), password);
    setSubmitting(false);
    if (success) {
      onClose();
    } else {
      setError("ユーザー名またはパスワードが正しくありません");
    }
  };

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal__header">
          <h2 className="modal__title">ログイン</h2>
          <button className="modal__close" onClick={onClose}>✕</button>
        </div>
        <div className="modal__body">
          <form className="login-form" onSubmit={handleSubmit}>
            {error && <div className="login-form__error">{error}</div>}
            <div>
              <label className="upload-form__label">ユーザー名</label>
              <input
                className="upload-form__input"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                placeholder="例: student01"
                id="login-username"
                autoFocus
                autoComplete="username"
                disabled={submitting}
              />
            </div>
            <div>
              <label className="upload-form__label">パスワード</label>
              <input
                className="upload-form__input"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="パスワード"
                id="login-password"
                autoComplete="current-password"
                disabled={submitting}
              />
            </div>
            <button
              className="login-form__submit"
              type="submit"
              id="login-submit"
              disabled={submitting}
            >
              {submitting ? "ログイン中..." : "ログイン"}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
