/* ===================================================
 * components/internal/Layout.tsx
 * [保護] ページレイアウト骨格
 * =================================================== */

import { useState } from "react";
import { Outlet } from "react-router-dom";
import Header from "./Header";
import Sidebar from "./Sidebar";
import UploadModal from "./UploadModal";
import LoginModal from "./LoginModal";
import { useAuth } from "../../hooks/useAuth";

export default function Layout() {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
  const [showUpload, setShowUpload] = useState(false);
  const [showLogin, setShowLogin] = useState(false);
  const { user, login, logout } = useAuth();

  return (
    <>
      <Header
        onToggleSidebar={() => setSidebarCollapsed(!sidebarCollapsed)}
        onOpenUpload={() => setShowUpload(true)}
        onOpenLogin={() => setShowLogin(true)}
        user={user}
        onLogout={logout}
      />
      <div className="layout">
        <Sidebar collapsed={sidebarCollapsed} user={user} />
        <main className={`layout__main ${sidebarCollapsed ? "layout__main--expanded" : ""}`}>
          <Outlet />
        </main>
      </div>
      {showUpload && <UploadModal onClose={() => setShowUpload(false)} />}
      {showLogin && <LoginModal onClose={() => setShowLogin(false)} onLogin={login} />}
    </>
  );
}
