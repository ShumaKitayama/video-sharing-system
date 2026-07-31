import { defineConfig, type ProxyOptions } from 'vite'
import react from '@vitejs/plugin-react'

/**
 * リモート（Vercel）のバックエンドへ中継する dev proxy を作る。
 *
 * ブラウザから API を直接呼ぶと、ログイン Cookie が `SameSite=Lax` のため
 * クロスサイト扱いになって送信されない。開発サーバー自身が API へ中継し、
 * ブラウザから見て「同一オリジン」にすることで Cookie が正しく動く。
 * これは本番の front/vercel.json の rewrite と同じ考え方である。
 */
function createRemoteProxy(target: string): ProxyOptions {
  return {
    target,
    changeOrigin: true,
    configure: (proxy) => {
      proxy.on('proxyRes', (proxyRes) => {
        const cookies = proxyRes.headers['set-cookie']
        if (!cookies) return
        // 本番 API は Secure 付きで Cookie を発行するが、http://localhost では
        // 保存を拒否するブラウザがあるため、開発時だけ属性を外す。
        proxyRes.headers['set-cookie'] = cookies.map((cookie) =>
          cookie.replace(/;\s*Secure/gi, ''),
        )
      })
    },
  }
}

// https://vite.dev/config/
export default defineConfig(() => {
  // Docker（front/Dockerfile.dev）から起動したときだけ true になる
  const useRemoteBackend = process.env.VITE_REMOTE_BACKEND === 'true'
  const remoteBackendUrl =
    process.env.VITE_REMOTE_BACKEND_URL ?? 'https://video-sharing-api.vercel.app'

  return {
    plugins: [react()],
    server: {
      port: 5173,
      // バインドマウント越しのファイル変更は inotify で検知できないため、
      // Docker のときだけポーリングに切り替えて HMR を動かす。
      watch:
        process.env.VITE_DEV_POLLING === 'true'
          ? { usePolling: true, interval: 300 }
          : undefined,
      // front/vercel.json の rewrite と同じパスを中継する
      proxy: useRemoteBackend
        ? {
            '/api': createRemoteProxy(remoteBackendUrl),
            '/health': createRemoteProxy(remoteBackendUrl),
            '/ready': createRemoteProxy(remoteBackendUrl),
          }
        : undefined,
    },
  }
})
