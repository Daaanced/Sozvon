// sozvon-client/src/components/voiceRooms/VoiceRoomsPage.tsx

import { useEffect, useState, useCallback, useRef } from "react";
import { getRooms, createRoom, deleteRoom, voiceClient } from "../../api/voice";
import { RoomInfo, PeerInfo } from "./voice.types";

// ── Logger ─────────────────────────────────────────────────────────────────

const log = {
  info: (msg: string, ...args: any[]) =>
    console.log(`%c[Voice] ${msg}`, "color:#6ee7b7;font-weight:600", ...args),
  warn: (msg: string, ...args: any[]) =>
    console.warn(`%c[Voice] ${msg}`, "color:#fcd34d;font-weight:600", ...args),
  error: (msg: string, ...args: any[]) =>
    console.error(`%c[Voice] ${msg}`, "color:#f87171;font-weight:600", ...args),
  event: (msg: string, data?: any) =>
    console.log(
      `%c[Voice ←] ${msg}`,
      "color:#818cf8;font-weight:600",
      data ?? "",
    ),
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
  const [activeRoomId, setActiveRoomId] = useState<string | null>(null);
  const [peers, setPeers] = useState<PeerInfo[]>([]);
  const [muted, setMuted] = useState(false);
  const [callActive, setCallActive] = useState(false);
  const [connecting, setConnecting] = useState(false);

  // Аудио элементы для каждого peer
  const audioRefs = useRef<Map<string, HTMLAudioElement>>(new Map());

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

  useEffect(() => {
    voiceClient.setEvents({
      onRoomState: (incomingPeers) => {
        log.event("room_state", incomingPeers);
        setPeers(incomingPeers);
      },
      onPeerJoined: (peer) => {
        log.event("peer_joined", peer);
        setPeers((prev) => {
          if (prev.find((p) => p.peer_id === peer.peer_id)) return prev;
          return [...prev, peer];
        });
        fetchRooms();
      },
      onPeerLeft: (peerId) => {
        log.event("peer_left", { peerId });
        setPeers((prev) => prev.filter((p) => p.peer_id !== peerId));
        // Убираем аудио элемент
        const audio = audioRefs.current.get(peerId);
        if (audio) {
          audio.srcObject = null;
          audioRefs.current.delete(peerId);
        }
        fetchRooms();
      },
      onPeerMuted: (peerId, isMuted) => {
        log.event("peer_muted", { peerId, muted: isMuted });
        setPeers((prev) =>
          prev.map((p) =>
            p.peer_id === peerId ? { ...p, muted: isMuted } : p,
          ),
        );
      },
      onTrack: (peerId, stream) => {
        log.event("track received", { peerId, stream });
        // Создаём аудио элемент для этого peer
        let audio = audioRefs.current.get(peerId);
        if (!audio) {
          audio = new Audio();
          audio.autoplay = true;
          audioRefs.current.set(peerId, audio);
          log.info(`Created audio element for peer ${peerId}`);
        }
        audio.srcObject = stream;
        audio.play().catch((e) => log.error("Audio play failed", e));
      },
      onConnected: () => {
        log.info("WebRTC connected ✓");
        setConnecting(false);
        setCallActive(true);
      },
      onDisconnected: () => {
        log.warn("WebRTC disconnected");
        setCallActive(false);
        setConnecting(false);
      },
      onError: (code, message) => {
        log.error(`Server error [${code}]: ${message}`);
        setError(message);
        setConnecting(false);
      },
    });
  }, [fetchRooms]);

  // Cleanup при размонтировании
  useEffect(() => {
    return () => {
      if (voiceClient.getCurrentRoomId()) {
        log.info("Page unmounted — leaving room");
        voiceClient.leaveRoom();
      }
    };
  }, []);

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

  const handleJoin = async (roomId: string) => {
    if (activeRoomId === roomId) return;
    log.info(`Joining room ${roomId}`);
    setConnecting(true);
    setActiveRoomId(roomId);
    setPeers([]);

    try {
      await voiceClient.joinRoom(roomId);
      log.info("Joined room, starting call...");
      await voiceClient.startCall();
    } catch (e: any) {
      log.error("Join failed", e);
      setError(e.message);
      setConnecting(false);
      setActiveRoomId(null);
    }
  };

  const handleLeave = () => {
    log.info(`Leaving room ${activeRoomId}`);
    voiceClient.leaveRoom();
    setActiveRoomId(null);
    setPeers([]);
    setCallActive(false);
    setMuted(false);
    // Чистим аудио
    audioRefs.current.forEach((a) => {
      a.srcObject = null;
    });
    audioRefs.current.clear();
    fetchRooms();
  };

  const handleMute = () => {
    const next = !muted;
    log.info(next ? "Muting mic" : "Unmuting mic");
    voiceClient.setMuted(next);
    setMuted(next);
  };

  const handleDeleteRoom = async (roomId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    if (activeRoomId === roomId) handleLeave();
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
              onClick={handleMute}
              title={muted ? "Unmute" : "Mute"}
            >
              {muted ? "🔇" : "🎙️"}
            </button>
            <button
              style={{ ...s.callActionBtn, background: "#dc2626" }}
              onClick={handleLeave}
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
                      <button style={s.leaveBtn} onClick={handleLeave}>
                        Leave
                      </button>
                    ) : (
                      <button
                        style={{ ...s.joinBtn, opacity: connecting ? 0.6 : 1 }}
                        onClick={() => handleJoin(room.id)}
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
  return (
    <div style={s.peerBadge}>
      <div style={{ ...s.peerAvatar, ...(live ? s.peerAvatarLive : {}) }}>
        {peer.username?.[0]?.toUpperCase() ?? "?"}
      </div>
      <span style={s.peerName}>{peer.username}</span>
      {peer.muted && <span title="Muted">🔇</span>}
      {live && !peer.muted && <span title="Speaking">🎙️</span>}
    </div>
  );
}

// ── Styles ─────────────────────────────────────────────────────────────────

const s: Record<string, React.CSSProperties> = {
  page: {
    display: "flex",
    flexDirection: "column",
    height: "100%",
    gap: 12,
    fontFamily: "'DM Sans', system-ui, sans-serif",
  },
  header: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
    paddingBottom: 12,
    borderBottom: "1px solid #e5e7eb",
  },
  headerLeft: { display: "flex", alignItems: "center", gap: 12 },
  headerIcon: { fontSize: 28 },
  headerTitle: { fontWeight: 700, fontSize: 18, color: "#111827" },
  headerSub: { fontSize: 12, color: "#6b7280", marginTop: 2 },
  createBtn: {
    padding: "7px 14px",
    background: "#111827",
    color: "#fff",
    border: "none",
    borderRadius: 8,
    cursor: "pointer",
    fontSize: 13,
    fontWeight: 600,
    transition: "opacity 0.15s",
  },
  createRow: {
    display: "flex",
    gap: 8,
  },
  input: {
    flex: 1,
    padding: "8px 12px",
    border: "1.5px solid #e5e7eb",
    borderRadius: 8,
    fontSize: 14,
    outline: "none",
    fontFamily: "inherit",
  },
  errorBanner: {
    background: "#fef2f2",
    border: "1px solid #fecaca",
    color: "#dc2626",
    borderRadius: 8,
    padding: "10px 14px",
    fontSize: 13,
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
  },
  errorClose: {
    background: "none",
    border: "none",
    cursor: "pointer",
    color: "#dc2626",
    fontSize: 14,
  },
  callBar: {
    background: "#111827",
    color: "#fff",
    borderRadius: 10,
    padding: "10px 16px",
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
  },
  callBarLeft: { display: "flex", alignItems: "center", gap: 10 },
  statusDot: {
    width: 8,
    height: 8,
    borderRadius: "50%",
    display: "inline-block",
    flexShrink: 0,
  },
  callBarText: { fontSize: 13, color: "#d1d5db" },
  callBarRoom: { fontSize: 13, fontWeight: 700, color: "#fff" },
  callBarActions: { display: "flex", gap: 8 },
  callActionBtn: {
    padding: "6px 12px",
    border: "none",
    borderRadius: 7,
    color: "#fff",
    cursor: "pointer",
    fontSize: 13,
    fontWeight: 600,
  },
  roomsList: {
    display: "flex",
    flexDirection: "column",
    gap: 8,
    overflowY: "auto",
    flex: 1,
  },
  empty: {
    color: "#9ca3af",
    fontSize: 14,
    textAlign: "center",
    paddingTop: 32,
  },
  roomCard: {
    background: "#fff",
    border: "1.5px solid #e5e7eb",
    borderRadius: 12,
    padding: "14px 16px",
    transition: "border-color 0.15s, box-shadow 0.15s",
  },
  roomCardActive: {
    borderColor: "#6366f1",
    boxShadow: "0 0 0 3px rgba(99,102,241,0.1)",
  },
  roomCardTop: {
    display: "flex",
    alignItems: "center",
    justifyContent: "space-between",
  },
  roomInfo: { display: "flex", alignItems: "center", gap: 12 },
  roomIcon: { fontSize: 22 },
  roomName: { fontWeight: 600, fontSize: 15, color: "#111827" },
  roomMeta: { fontSize: 12, color: "#6b7280", marginTop: 2 },
  roomActions: { display: "flex", alignItems: "center", gap: 6 },
  joinBtn: {
    padding: "6px 16px",
    background: "#6366f1",
    color: "#fff",
    border: "none",
    borderRadius: 7,
    cursor: "pointer",
    fontSize: 13,
    fontWeight: 600,
  },
  leaveBtn: {
    padding: "6px 16px",
    background: "#f3f4f6",
    color: "#374151",
    border: "none",
    borderRadius: 7,
    cursor: "pointer",
    fontSize: 13,
    fontWeight: 600,
  },
  deleteBtn: {
    width: 28,
    height: 28,
    background: "none",
    border: "1px solid #e5e7eb",
    borderRadius: 6,
    cursor: "pointer",
    fontSize: 12,
    color: "#9ca3af",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
  },
  peersList: {
    display: "flex",
    flexWrap: "wrap",
    gap: 6,
    marginTop: 10,
    paddingTop: 10,
    borderTop: "1px solid #f3f4f6",
  },
  peerBadge: {
    display: "flex",
    alignItems: "center",
    gap: 6,
    background: "#f9fafb",
    border: "1px solid #e5e7eb",
    borderRadius: 20,
    padding: "4px 10px 4px 4px",
  },
  peerAvatar: {
    width: 24,
    height: 24,
    borderRadius: "50%",
    background: "#e0e7ff",
    color: "#4338ca",
    display: "flex",
    alignItems: "center",
    justifyContent: "center",
    fontSize: 11,
    fontWeight: 700,
    flexShrink: 0,
  },
  peerAvatarLive: {
    background: "#d1fae5",
    color: "#065f46",
  },
  peerName: { fontSize: 13, color: "#374151", fontWeight: 500 },
};
