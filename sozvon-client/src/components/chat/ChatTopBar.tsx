//sozvon-client\src\components\chat\ChatTopBar.tsx

import { useState } from "react";
import type { User } from "../../api/users";
import { styles } from "./chat.styles";

type GroupInfo = {
  name: string;
  picture: string;
};

type Props = {
  user: User | null;
  groupInfo?: GroupInfo | null;
  onCall: () => void;
  onSettings: () => void;
};

export default function ChatTopBar({ user, groupInfo, onCall, onSettings }: Props) {
  const [searchOpen, setSearchOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");

  const displayName = groupInfo?.name ?? user?.name ?? "...";
  const displayPicture = groupInfo?.picture ?? user?.picture ?? null;

  return (
    <div style={styles.bar}>
      <div style={styles.userInfo}>
        {displayPicture && (
          <img src={displayPicture} style={styles.avatar} alt={displayName} />
        )}
        <span style={styles.userName}>{displayName}</span>
      </div>

      <div style={styles.actions}>
        {searchOpen && (
          <input
            autoFocus
            style={styles.searchInput}
            placeholder="Search messages..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            onKeyDown={(e) => e.key === "Escape" && setSearchOpen(false)}
          />
        )}

        <button
          style={styles.iconBtn}
          onClick={() => setSearchOpen((prev) => !prev)}
          title="Search"
        >
          🔍
        </button>

        <button style={styles.iconBtn} onClick={onCall} title="Call">
          📞
        </button>

        <button style={styles.iconBtn} onClick={onSettings} title="Settings">
          ⚙️
        </button>
      </div>
    </div>
  );
}