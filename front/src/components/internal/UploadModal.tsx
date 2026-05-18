/* ===================================================
 * components/internal/UploadModal.tsx
 * [保護] 動画アップロードモーダル
 * =================================================== */

import { useState, useRef } from "react";
import { useVideoUpload } from "../../hooks/useVideoUpload";

interface UploadModalProps {
  onClose: () => void;
}

export default function UploadModal({ onClose }: UploadModalProps) {
  const [file, setFile] = useState<File | null>(null);
  const [title, setTitle] = useState("");
  const [description, setDescription] = useState("");
  const [dragActive, setDragActive] = useState(false);
  const [success, setSuccess] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const { uploading, progress, error, uploadVideo, reset } = useVideoUpload();

  const handleFile = (f: File) => {
    setFile(f);
    if (!title) setTitle(f.name.replace(/\.[^.]+$/, ""));
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setDragActive(false);
    if (e.dataTransfer.files[0]) handleFile(e.dataTransfer.files[0]);
  };

  const handleSubmit = async () => {
    if (!file) return;
    const ok = await uploadVideo(file, title, description);
    if (ok) setSuccess(true);
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  return (
    <div className="modal-overlay" onClick={handleClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal__header">
          <h2 className="modal__title">動画をアップロード</h2>
          <button className="modal__close" onClick={handleClose}>✕</button>
        </div>
        <div className="modal__body">
          {success ? (
            <div className="upload-success">
              <div className="upload-success__icon">✅</div>
              <div className="upload-success__text">アップロードが完了しました</div>
            </div>
          ) : !file ? (
            <>
              <div
                className={`upload-drop ${dragActive ? "upload-drop--active" : ""}`}
                onDragOver={(e) => { e.preventDefault(); setDragActive(true); }}
                onDragLeave={() => setDragActive(false)}
                onDrop={handleDrop}
                onClick={() => inputRef.current?.click()}
              >
                <div className="upload-drop__icon">📤</div>
                <div className="upload-drop__text">ファイルをドラッグ＆ドロップ</div>
                <div className="upload-drop__sub">またはクリックしてファイルを選択（MP4, WebM / 500MB以下）</div>
              </div>
              <input
                ref={inputRef}
                type="file"
                accept="video/mp4,video/webm"
                style={{ display: "none" }}
                onChange={(e) => e.target.files?.[0] && handleFile(e.target.files[0])}
              />
            </>
          ) : (
            <>
              <div style={{ fontSize: 13, color: "var(--text-secondary)", marginBottom: 12 }}>
                📄 {file.name} ({(file.size / 1024 / 1024).toFixed(1)} MB)
              </div>
              <div className="upload-form">
                <div>
                  <label className="upload-form__label">タイトル *</label>
                  <input className="upload-form__input" value={title} onChange={(e) => setTitle(e.target.value)} maxLength={80} id="upload-title" />
                </div>
                <div>
                  <label className="upload-form__label">説明</label>
                  <textarea className="upload-form__textarea" value={description} onChange={(e) => setDescription(e.target.value)} maxLength={1000} id="upload-desc" />
                </div>
                {error && <div className="login-form__error">{error}</div>}
                {uploading && (
                  <div className="upload-progress">
                    <div className="upload-progress__bar"><div className="upload-progress__fill" style={{ width: `${progress}%` }} /></div>
                    <div className="upload-progress__text">{progress}%</div>
                  </div>
                )}
                <button className="upload-form__submit" onClick={handleSubmit} disabled={uploading || !title.trim()} id="upload-submit">
                  {uploading ? "アップロード中..." : "アップロード"}
                </button>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
