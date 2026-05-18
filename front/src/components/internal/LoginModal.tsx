/* ===================================================
 * components/internal/LoginModal.tsx
 * [保護] ログインモーダル
 * =================================================== */

import { useState } from "react";

interface LoginModalProps {
  onClose: () => void;
  onLogin: (username: string) => boolean;
}

export default function LoginModal({ onClose, onLogin }: LoginModalProps) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim()) {
      setError("ユーザー名を入力してください");
      return;
    }
    const success = onLogin(username.trim());
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
                placeholder="例: yamada_taro"
                id="login-username"
                autoFocus
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
              />
            </div>
            <div style={{ fontSize: 12, color: "var(--text-secondary)", background: "var(--bg-hover)", padding: "10px 12px", borderRadius: 8 }}>
              💡 デモ用ユーザー名: <strong>yamada_taro</strong>, <strong>suzuki_hana</strong>, <strong>teacher</strong> など（パスワードは任意）
            </div>
            <button className="login-form__submit" type="submit" id="login-submit">ログイン</button>
          </form>
        </div>
      </div>
    </div>
  );
}
