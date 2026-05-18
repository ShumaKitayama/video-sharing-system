/* ===================================================
 * App.tsx
 * ルーティング定義
 * =================================================== */

import { BrowserRouter, Routes, Route } from "react-router-dom";
import Layout from "./components/internal/Layout";
import HomePage from "./pages/HomePage";
import WatchPage from "./pages/WatchPage";
import SearchResultsPage from "./pages/SearchResultsPage";
import ChannelPage from "./pages/ChannelPage";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<HomePage />} />
          <Route path="/watch/:id" element={<WatchPage />} />
          <Route path="/search" element={<SearchResultsPage />} />
          <Route path="/channel/:id" element={<ChannelPage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
