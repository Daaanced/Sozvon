// sozvon-client/src/components/voiceRooms/VoiceRoomsPage.tsx

import { useEffect, useState, useCallback, useMemo } from "react";
import { getRooms, createRoom, deleteRoom } from "../../api/voice";
import { RoomInfo, PeerInfo } from "./voice.types";
import { s } from "./voice.styles";
import { useVoiceContext } from "../../context/VoiceContext";
// ── Logger ─────────────────────────────────────────────────────────────────

const log = {
  info: (msg: string, ...args: any[]) =>
    console.log(`%c[Voice] ${msg}`, "color:#6ee7b7;font-weight:600", ...args),
  warn: (msg: string, ...args: any[]) =>
    console.warn(`%c[Voice] ${msg}`, "color:#fcd34d;font-weight:600", ...args),
  error: (msg: string, ...args: any[]) =>
    console.error(`%c[Voice] ${msg}`, "color:#f87171;font-weight:600", ...args),
};

// ── Component ──────────────────────────────────────────────────────────────

export default function VoiceRoomsPage() {
  const [rooms, setRooms] = useState<RoomInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [newRoomName, setNewRoomName] = useState("");
  const [creating, setCreating] = useState(false);
  const [showCreateInput, setShowCreateInput] = useState(false);

  // Текущая комната и участники
  const {
    activeRoomId,
    peers,
    muted,
    callActive,
    connecting,
    joinRoom,
    leaveRoom,
    toggleMute,
  } = useVoiceContext();

  // ── Загрузка комнат ──────────────────────────────────────────────────────
  const fetchRooms = useCallback(async () => {
    try {
      log.info("Fetching rooms...");
      const data = await getRooms();
      setRooms(data.rooms ?? []);
      log.info(`Loaded ${data.total} rooms`, data.rooms);
    } catch (e: any) {
      log.error("Failed to fetch rooms", e);
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchRooms();
    // Периодически обновляем список чтобы видеть новых участников
    const interval = setInterval(fetchRooms, 8000);
    return () => clearInterval(interval);
  }, [fetchRooms]);

  // ── События голосового клиента ───────────────────────────────────────────

  // ── Действия ─────────────────────────────────────────────────────────────

  const handleCreateRoom = async () => {
    const name = newRoomName.trim();
    if (!name) return;
    setCreating(true);
    try {
      log.info(`Creating room: "${name}"`);
      const result = await createRoom(name);
      log.info("Room created", result);
      setNewRoomName("");
      setShowCreateInput(false);
      await fetchRooms();
    } catch (e: any) {
      log.error("Failed to create room", e);
      setError(e.message);
    } finally {
      setCreating(false);
    }
  };

  const handleDeleteRoom = async (roomId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (activeRoomId === roomId) leaveRoom();
    try {
      log.info(`Deleting room ${roomId}`);
      await deleteRoom(roomId);
      await fetchRooms();
    } catch (err: any) {
      log.error("Delete failed", err);
    }
  };

  // ── Render ────────────────────────────────────────────────────────────────

  return (
    <div style={s.page}>
      {/* Header */}
      <div style={s.header}>
        <div style={s.headerLeft}>
          <span style={s.headerIcon}>🔊</span>
          <div>
            <div style={s.headerTitle}>Voice Rooms</div>
            <div style={s.headerSub}>
              {rooms.length} active room{rooms.length !== 1 ? "s" : ""}
            </div>
          </div>
        </div>
        <button
          style={s.createBtn}
          onClick={() => setShowCreateInput((v) => !v)}
        >
          {showCreateInput ? "✕" : "+ New Room"}
        </button>
      </div>

      {/* Create room input */}
      {showCreateInput && (
        <div style={s.createRow}>
          <input
            style={s.input}
            placeholder="Room name..."
            value={newRoomName}
            onChange={(e) => setNewRoomName(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleCreateRoom()}
            autoFocus
          />
          <button
            style={{ ...s.createBtn, opacity: creating ? 0.6 : 1 }}
            onClick={handleCreateRoom}
            disabled={creating || !newRoomName.trim()}
          >
            {creating ? "Creating..." : "Create"}
          </button>
        </div>
      )}

      {error && (
        <div style={s.errorBanner}>
          ⚠️ {error}
          <button style={s.errorClose} onClick={() => setError("")}>
            ✕
          </button>
        </div>
      )}

      {/* Active call bar */}
      {activeRoomId && (
        <div style={s.callBar}>
          <div style={s.callBarLeft}>
            <span
              style={{
                ...s.statusDot,
                background: callActive ? "#4ade80" : "#facc15",
              }}
            />
            <span style={s.callBarText}>
              {connecting
                ? "Connecting..."
                : callActive
                  ? "Connected"
                  : "In room"}
            </span>
            <span style={s.callBarRoom}>
              {rooms.find((r) => r.id === activeRoomId)?.name ??
                activeRoomId.slice(0, 8)}
            </span>
          </div>
          <div style={s.callBarActions}>
            <button
              style={{
                ...s.callActionBtn,
                background: muted ? "#ef4444" : "#374151",
              }}
              onClick={toggleMute}
              title={muted ? "Unmute" : "Mute"}
            >
              {muted ? "🔇" : "🎙️"}
            </button>
            <button
              style={{ ...s.callActionBtn, background: "#dc2626" }}
              onClick={leaveRoom}
            >
              📵 Leave
            </button>
          </div>
        </div>
      )}

      {/* Rooms list */}
      <div style={s.roomsList}>
        {loading ? (
          <div style={s.empty}>Loading rooms...</div>
        ) : rooms.length === 0 ? (
          <div style={s.empty}>No rooms yet. Create one above.</div>
        ) : (
          rooms.map((room) => {
            const isActive = room.id === activeRoomId;
            return (
              <div
                key={room.id}
                style={{ ...s.roomCard, ...(isActive ? s.roomCardActive : {}) }}
              >
                <div style={s.roomCardTop}>
                  <div style={s.roomInfo}>
                    <span style={s.roomIcon}>🔊</span>
                    <div>
                      <div style={s.roomName}>{room.name}</div>
                      <div style={s.roomMeta}>
                        {room.peer_count} participant
                        {room.peer_count !== 1 ? "s" : ""}
                      </div>
                    </div>
                  </div>
                  <div style={s.roomActions}>
                    {isActive ? (
                      <button style={s.leaveBtn} onClick={leaveRoom}>
                        Leave
                      </button>
                    ) : (
                      <button
                        style={{ ...s.joinBtn, opacity: connecting ? 0.6 : 1 }}
                        onClick={() => joinRoom(room.id, room.name)}
                        disabled={connecting}
                      >
                        Join
                      </button>
                    )}
                    <button
                      style={s.deleteBtn}
                      onClick={(e) => handleDeleteRoom(room.id, e)}
                      title="Delete room"
                    >
                      ✕
                    </button>
                  </div>
                </div>

                {/* Участники комнаты */}
                {room.peers && room.peers.length > 0 && (
                  <div style={s.peersList}>
                    {room.peers.map((peer) => (
                      <PeerBadge key={peer.peer_id} peer={peer} />
                    ))}
                  </div>
                )}

                {/* Живые участники (из WS событий) при активной комнате */}
                {isActive && peers.length > 0 && (
                  <div style={s.peersList}>
                    {peers.map((peer) => (
                      <PeerBadge key={peer.peer_id} peer={peer} live />
                    ))}
                  </div>
                )}
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}

// ── PeerBadge ──────────────────────────────────────────────────────────────

function PeerBadge({ peer, live }: { peer: PeerInfo; live?: boolean }) {
  // Оптимизируем создание объекта стилей
  const avatarStyle = useMemo(
    () => ({
      ...s.peerAvatar,
      ...(live ? s.peerAvatarLive : {}),
    }),
    [live],
  );

  const displayName = peer.username || "Anonymous";
  const firstLetter = displayName[0].toUpperCase();

  return (
    <div style={s.peerBadge}>
      <div style={avatarStyle}>{firstLetter}</div>
      <span style={s.peerName}>{displayName}</span>
      {peer.muted && <span title="Muted">🔇</span>}
      {live && !peer.muted && <span title="Speaking">🎙️</span>}
    </div>
  );
}
