/* ===================================================
 * components/internal/Layout.tsx
 * [保護] ページレイアウト骨格（サイドバーなし）
 * useAuth の非同期 login / logout に対応。
 * =================================================== */

import { useState } from "react";
import { Outlet } from "react-router-dom";
import Header from "./Header";
import UploadModal from "./UploadModal";
import LoginModal from "./LoginModal";
import { useAuth } from "../../hooks/useAuth";

export default function Layout() {
  const [showUpload, setShowUpload] = useState(false);
  const [showLogin, setShowLogin] = useState(false);
  const auth = useAuth();

  return (
    <>
      <Header
        onOpenUpload={() => setShowUpload(true)}
        onOpenLogin={() => setShowLogin(true)}
        user={auth.user}
        onLogout={auth.logout}
      />
      <div className="layout">
        <main className="layout__main">
          <Outlet context={auth} />
        </main>
      </div>
      {showUpload && <UploadModal onClose={() => setShowUpload(false)} />}
      {showLogin && (
        <LoginModal onClose={() => setShowLogin(false)} onLogin={auth.login} />
      )}
    </>
  );
}
