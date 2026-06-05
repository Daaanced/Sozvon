// sozvon-client/src/context/VoiceContext.tsx
import { createContext, useContext, useEffect, useRef, useState } from "react";

import { voiceClient } from "../api/voice";
import { PeerInfo } from "../components/voiceRooms/voice.types";

type VoiceContextType = {
  activeRoomId: string | null;
  activeRoomName: string | null;
  peers: PeerInfo[];

  muted: boolean;
  callActive: boolean;
  connecting: boolean;

  joinRoom: (id: string, name: string) => Promise<void>;
  leaveRoom: () => void;
  toggleMute: () => void;
};

const VoiceContext = createContext<VoiceContextType | null>(null);

export function useVoiceContext() {
  const ctx = useContext(VoiceContext);

  if (!ctx) {
    throw new Error("useVoiceContext must be used inside VoiceProvider");
  }

  return ctx;
}

export function VoiceProvider({ children }: { children: React.ReactNode }) {
  const [activeRoomId, setActiveRoomId] = useState<string | null>(null);

  const [activeRoomName, setActiveRoomName] = useState<string | null>(null);

  const [peers, setPeers] = useState<PeerInfo[]>([]);

  const [muted, setMuted] = useState(false);

  const [callActive, setCallActive] = useState(false);

  const [connecting, setConnecting] = useState(false);

  const audioRefs = useRef<Map<string, HTMLAudioElement>>(new Map());

  useEffect(() => {
    voiceClient.setEvents({
      onRoomState: setPeers,

      onPeerJoined: (peer) => {
        setPeers((prev) => {
          if (prev.some((p) => p.peer_id === peer.peer_id)) {
            return prev;
          }

          return [...prev, peer];
        });
      },
      onPeerMuted: (peerId, isMuted) => {
        setPeers((prev) =>
          prev.map((p) =>
            p.peer_id === peerId
              ? {
                  ...p,
                  muted: isMuted,
                }
              : p,
          ),
        );
      },
      onPeerLeft: (peerId) => {
        setPeers((prev) => prev.filter((p) => p.peer_id !== peerId));
      },

      onTrack: (peerId, stream) => {
        let audio = audioRefs.current.get(peerId);

        if (!audio) {
          audio = new Audio();

          audio.autoplay = true;

          audioRefs.current.set(peerId, audio);
        }

        audio.srcObject = stream;
      },

      onConnected: () => {
        setConnecting(false);
        setCallActive(true);
      },

      onDisconnected: () => {
        setCallActive(false);
        setConnecting(false);
      },
    });
  }, []);

  async function joinRoom(id: string, name: string) {
    if (activeRoomId === id) return;

    try {
      setConnecting(true);

      await voiceClient.joinRoom(id);

      await voiceClient.startCall();

      setActiveRoomId(id);
      setActiveRoomName(name);
    } catch (e) {
      console.error("Join failed:", e);

      setActiveRoomId(null);
      setActiveRoomName(null);

      throw e;
    } finally {
      setConnecting(false);
    }
  }

  function leaveRoom() {
    voiceClient.leaveRoom();

    setActiveRoomId(null);
    setActiveRoomName(null);

    setPeers([]);
    setMuted(false);
    setCallActive(false);

    audioRefs.current.forEach((a) => {
      a.srcObject = null;
    });

    audioRefs.current.clear();
  }

  function toggleMute() {
    const next = !muted;

    voiceClient.setMuted(next);

    setMuted(next);
  }

  return (
    <VoiceContext.Provider
      value={{
        activeRoomId,
        activeRoomName,
        peers,
        muted,
        callActive,
        connecting,
        joinRoom,
        leaveRoom,
        toggleMute,
      }}
    >
      {children}
    </VoiceContext.Provider>
  );
}
