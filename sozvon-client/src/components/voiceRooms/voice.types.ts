//sozvon-client\src\components\voiceRooms\voice.types.ts

export interface RoomInfo {
  id: string;
  name: string;
  created_by: string;
  created_at: string;
  peer_count: number;
  peers: PeerInfo[];
}

export interface PeerInfo {
  peer_id: string;
  user_id: string;
  username: string;
  muted: boolean;
  deafened: boolean;
}

export interface VoiceEvents {
  onRoomState?: (peers: PeerInfo[]) => void;
  onPeerJoined?: (peer: PeerInfo) => void;
  onPeerLeft?: (peerId: string) => void;
  onPeerMuted?: (peerId: string, muted: boolean) => void;
  onPeerDeafened?: (peerId: string, deafened: boolean) => void;
  onTrack?: (peerId: string, stream: MediaStream) => void;
  onError?: (code: string, message: string) => void;
  onConnected?: () => void;
  onDisconnected?: () => void;
}
