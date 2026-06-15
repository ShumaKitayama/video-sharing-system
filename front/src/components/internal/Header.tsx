/* ===================================================
 * components/internal/Header.tsx
 * [保護] ヘッダーバー（ロゴ・検索・ユーザーメニュー）
 * サイドバートグル削除済み
 * =================================================== */

import { useState, useRef, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import type { User } from "../../types";

interface HeaderProps {
  onOpenUpload: () => void;
  onOpenLogin: () => void;
  user: User | null;
  onLogout: () => void;
}

export default function Header({ onOpenUpload, onOpenLogin, user, onLogout }: HeaderProps) {
  const [query, setQuery] = useState("");
  const [showMenu, setShowMenu] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);
  const navigate = useNavigate();

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (query.trim()) {
      navigate(`/search?q=${encodeURIComponent(query.trim())}`);
    }
  };

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setShowMenu(false);
      }
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  return (
    <header className="header" id="main-header">
      <div className="header__left">
        <a className="header__logo" onClick={() => navigate("/")} style={{ cursor: "pointer" }} id="logo-link">
          <span className="header__logo-icon">▶</span>
          <span>VideoShare</span>
        </a>
      </div>

      <div className="header__center">
        <form className="header__search-form" onSubmit={handleSearch}>
          <input
            className="header__search-input"
            type="text"
            placeholder="動画を検索..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            id="search-input"
          />
          <button className="header__search-btn" type="submit" id="search-btn" aria-label="検索">🔍</button>
        </form>
      </div>

      <div className="header__right">
        {user && (
          <button className="header__icon-btn" onClick={onOpenUpload} id="upload-btn" aria-label="アップロード" title="動画をアップロード">
            🎥
          </button>
        )}

        {user ? (
          <div style={{ position: "relative" }} ref={menuRef}>
            <div className="header__avatar" onClick={() => setShowMenu(!showMenu)} id="user-avatar" title={user.display_name}>
              {user.display_name.charAt(0)}
            </div>
            {showMenu && (
              <div className="header__user-menu">
                <div style={{ padding: "12px 20px", borderBottom: "1px solid var(--border-color)" }}>
                  <div style={{ fontWeight: 600 }}>{user.display_name}</div>
                  <div style={{ fontSize: 12, color: "var(--text-secondary)" }}>@{user.username}</div>
                </div>
                <button className="header__user-menu-item" onClick={() => { navigate(`/channel/${user.id}`); setShowMenu(false); }}>
                  👤 マイチャンネル
                </button>
                <button className="header__user-menu-item" onClick={() => { onLogout(); setShowMenu(false); }}>
                  🚪 ログアウト
                </button>
              </div>
            )}
          </div>
        ) : (
          <button className="header__login-btn" onClick={onOpenLogin} id="login-btn">
            ログイン
          </button>
        )}
      </div>
    </header>
  );
}
