/* ===================================================
 * components/student/ChannelAvatar.tsx
 * [学生編集可] チャンネルアバターコンポーネント
 * =================================================== */

interface ChannelAvatarProps {
  displayName: string;
  size?: number;
  className?: string;
}

const avatarColors = [
  "linear-gradient(135deg, #667eea, #764ba2)",
  "linear-gradient(135deg, #f093fb, #f5576c)",
  "linear-gradient(135deg, #4facfe, #00f2fe)",
  "linear-gradient(135deg, #43e97b, #38f9d7)",
  "linear-gradient(135deg, #fa709a, #fee140)",
  "linear-gradient(135deg, #a18cd1, #fbc2eb)",
  "linear-gradient(135deg, #fccb90, #d57eeb)",
  "linear-gradient(135deg, #89f7fe, #66a6ff)",
];

function getColorForName(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i++) {
    hash = name.charCodeAt(i) + ((hash << 5) - hash);
  }
  return avatarColors[Math.abs(hash) % avatarColors.length];
}

export default function ChannelAvatar({ displayName, size = 36, className = "" }: ChannelAvatarProps) {
  const initial = displayName.charAt(0);
  const bg = getColorForName(displayName);

  return (
    <div
      className={className}
      style={{
        width: size,
        height: size,
        borderRadius: "50%",
        background: bg,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        fontWeight: 600,
        fontSize: size * 0.4,
        color: "#fff",
        flexShrink: 0,
      }}
    >
      {initial}
    </div>
  );
}
