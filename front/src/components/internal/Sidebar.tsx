/* ===================================================
 * components/internal/Sidebar.tsx
 * [保護] サイドバーナビゲーション
 * =================================================== */

import { useNavigate, useLocation } from "react-router-dom";

interface SidebarProps {
  collapsed: boolean;
  user: { id: string } | null;
}

export default function Sidebar({ collapsed, user }: SidebarProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const path = location.pathname;

  const items = [
    { icon: "🏠", label: "ホーム", path: "/" },
    { icon: "🔥", label: "人気", path: "/?sort=most_viewed" },
    { icon: "⏰", label: "最新", path: "/?sort=latest" },
  ];

  const userItems = user
    ? [
        { icon: "📁", label: "マイ動画", path: `/channel/${user.id}` },
        { icon: "👍", label: "高評価", path: "/?sort=most_liked" },
      ]
    : [];

  return (
    <nav className={`sidebar ${collapsed ? "sidebar--collapsed" : ""}`} id="main-sidebar">
      {items.map((item) => (
        <div
          key={item.label}
          className={`sidebar__item ${path === item.path ? "sidebar__item--active" : ""}`}
          onClick={() => navigate(item.path)}
        >
          <span className="sidebar__item-icon">{item.icon}</span>
          <span className="sidebar__label">{item.label}</span>
        </div>
      ))}

      {userItems.length > 0 && (
        <>
          <div className="sidebar__divider" />
          {!collapsed && <div className="sidebar__section-title">ライブラリ</div>}
          {userItems.map((item) => (
            <div
              key={item.label}
              className={`sidebar__item ${path === item.path ? "sidebar__item--active" : ""}`}
              onClick={() => navigate(item.path)}
            >
              <span className="sidebar__item-icon">{item.icon}</span>
              <span className="sidebar__label">{item.label}</span>
            </div>
          ))}
        </>
      )}

      <div className="sidebar__divider" />
      {!collapsed && (
        <div style={{ padding: "12px 24px", fontSize: 11, color: "var(--text-disabled)", lineHeight: 1.6 }}>
          © 2026 VideoShare<br />教育用ローカルシステム
        </div>
      )}
    </nav>
  );
}
