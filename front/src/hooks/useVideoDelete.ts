/* ===================================================
 * hooks/useVideoDelete.ts
 * [保護] 動画削除を抽象化するカスタムフック
 * =================================================== */

import { useCallback, useState } from "react";
import { videoService } from "../services/api";

interface UseVideoDeleteResult {
  deleting: boolean;
  deleteVideo: (id: string) => Promise<boolean>;
}

export function useVideoDelete(): UseVideoDeleteResult {
  const [deleting, setDeleting] = useState(false);

  const deleteVideo = useCallback(async (id: string): Promise<boolean> => {
    if (!window.confirm("この動画を削除しますか？元に戻せません。")) {
      return false;
    }

    setDeleting(true);
    try {
      await videoService.deleteVideo(id);
      return true;
    } catch {
      window.alert("動画の削除に失敗しました");
      return false;
    } finally {
      setDeleting(false);
    }
  }, []);

  return { deleting, deleteVideo };
}
