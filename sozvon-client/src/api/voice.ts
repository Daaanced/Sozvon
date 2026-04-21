// sozvon-client/src/api/voice.ts
//
// Клиентская сторона голосового чата.
//
// Архитектура:
//   - Сигнализация идёт через существующий WS (ws.ts) — sendWS/onWSMessage
//   - RTCPeerConnection устанавливается с Voice Service напрямую
//   - Медиапоток (SRTP/UDP) течёт напрямую клиент ↔ Voice Service
//
// Жизненный цикл:
//   1. joinRoom()       — отправить {"type":"join","payload":{"room_id":"..."}}
//   2. startCall()      — getUserMedia → createOffer → отправить offer
//   3. Voice Service    → answer → setRemoteDescription
//   4. ICE exchange     — кандидаты в обе стороны
//   5. Медиапоток       — аудио течёт по UDP
//   6. leaveRoom()      — закрыть PeerConnection, отправить leave

import { sendWS, onWSMessage } from "../services/ws";
import { requestAuth } from "../api/http";
import { RoomInfo, VoiceEvents } from "../components/voiceRooms/voice.types.ts";

// ── REST API ───────────────────────────────────────────────────────────────

export function getRooms(): Promise<{ rooms: RoomInfo[]; total: number }> {
  return requestAuth("/voice/rooms");
}

export function createRoom(
  name: string,
): Promise<{ room_id: string; name: string }> {
  return requestAuth("/voice/rooms", {
    method: "POST",
    body: JSON.stringify({ name }),
  });
}

export function getRoom(roomId: string): Promise<RoomInfo> {
  return requestAuth(`/voice/rooms/${roomId}`);
}

export function deleteRoom(roomId: string): Promise<void> {
  return requestAuth(`/voice/rooms/${roomId}`, { method: "DELETE" });
}

// ── VoiceClient ────────────────────────────────────────────────────────────

const STUN_SERVERS: RTCIceServer[] = [{ urls: "stun:stun.l.google.com:19302" }];

class VoiceClient {
  private pc: RTCPeerConnection | null = null;
  private localStream: MediaStream | null = null;
  private currentRoomId: string | null = null;
  private events: VoiceEvents = {};
  private unsubscribe: (() => void) | null = null;
  private isNegotiating = false;
  private pendingOffer: string | null = null;
  // peer_id → MediaStream — треки от других участников
  private remoteStreams: Map<string, MediaStream> = new Map();

  // ── Инициализация ────────────────────────────────────────────────────────

  setEvents(events: VoiceEvents) {
    this.events = events;
  }

  // ── Комнаты ───────────────────────────────────────────────────────────────

  async joinRoom(roomId: string) {
    if (this.currentRoomId) {
      await this.leaveRoom();
    }

    this.currentRoomId = roomId;

    // Подписываемся на входящие WS сообщения от Voice Service
    this.unsubscribe = onWSMessage((msg) => this.handleSignal(msg));

    // Инициализируем PeerConnection
    this.initPC();

    // Отправляем join
    sendWS({
      type: "join",
      payload: { room_id: roomId },
    });
  }

  async leaveRoom() {
    sendWS({ type: "leave" });

    this.cleanup();
  }

  // ── Медиа ─────────────────────────────────────────────────────────────────

  // startCall — захватить микрофон и отправить offer
  async startCall() {
    if (!this.pc) {
      console.error(
        "[voice] PeerConnection не инициализирован, сначала joinRoom()",
      );
      return;
    }

    try {
      this.localStream = await navigator.mediaDevices.getUserMedia({
        audio: true,
        video: false,
      });

      // Добавляем аудио трек в PeerConnection
      this.localStream.getAudioTracks().forEach((track) => {
        this.pc!.addTrack(track, this.localStream!);
      });

      await this.sendOffer();

      this.events.onConnected?.();
    } catch (err) {
      console.error("[voice] getUserMedia error:", err);
      this.events.onError?.(
        "media_error",
        "Не удалось получить доступ к микрофону",
      );
    }
  }

  // setMuted — замутить/размутить свой микрофон
  setMuted(muted: boolean) {
    if (this.localStream) {
      this.localStream.getAudioTracks().forEach((t) => {
        t.enabled = !muted;
      });
    }

    sendWS({
      type: "mute",
      payload: { muted },
    });
  }

  isMuted(): boolean {
    if (!this.localStream) return true;
    return this.localStream.getAudioTracks().every((t) => !t.enabled);
  }

  // setPreferredLayer — выбрать simulcast слой (для будущего видео)
  setPreferredLayer(peerId: string, layer: "low" | "medium" | "high") {
    sendWS({
      type: "set_layer",
      payload: { peer_id: peerId, layer },
    });
  }

  getRemoteStream(peerId: string): MediaStream | undefined {
    return this.remoteStreams.get(peerId);
  }

  getCurrentRoomId(): string | null {
    return this.currentRoomId;
  }

  // ── Внутренняя логика ─────────────────────────────────────────────────────

  private initPC() {
    this.pc = new RTCPeerConnection({ iceServers: STUN_SERVERS });

    // ICE кандидат найден — отправляем на сервер
    this.pc.onicecandidate = ({ candidate }) => {
      if (!candidate) return;
      sendWS({
        type: "ice_candidate",
        payload: {
          candidate: candidate.candidate,
          sdp_mid: candidate.sdpMid ?? "",
          sdp_mline_index: candidate.sdpMLineIndex ?? 0,
        },
      });
    };

    // Входящий трек от другого участника
    // Pion шлёт треки через re-offer, поэтому они приходят после renegotiation
    this.pc.ontrack = (event) => {
      const stream = event.streams[0] ?? new MediaStream([event.track]);

      // Определяем peerId по streamId — сервер называет стримы "voice-{peerId}"
      const streamId = event.streams[0]?.id ?? "";
      const peerId = streamId.startsWith("voice-")
        ? streamId.slice(6)
        : streamId;

      this.remoteStreams.set(peerId, stream);
      this.events.onTrack?.(peerId, stream);
    };

    this.pc.onconnectionstatechange = () => {
      const state = this.pc?.connectionState;
      console.log("[voice] connection state:", state);

      if (state === "failed" || state === "closed") {
        this.events.onDisconnected?.();
        this.cleanup();
      }
    };

    this.pc.onnegotiationneeded = async () => {
      // Браузер просит re-offer (например при добавлении трека)
      // В нашем случае сервер инициирует renegotiation сам через Pion,
      // поэтому здесь обрабатываем только первичный offer
      if (this.isNegotiating) return;
      if (this.pc?.signalingState !== "stable") return;
      if (!this.localStream) return;
      await this.sendOffer();
    };
  }

  private async sendOffer() {
    if (!this.pc) return;
    if (this.isNegotiating) return; // ← guard

    this.isNegotiating = true; // ← lock
    try {
      const offer = await this.pc.createOffer();
      await this.pc.setLocalDescription(offer);
      sendWS({ type: "offer", payload: { sdp: offer.sdp } });
    } finally {
      // Снимаем lock только когда получим answer
      // (см. handleRemoteAnswer ниже)
    }
  }

  // handleSignal — роутинг входящих сигнальных сообщений
  private async handleSignal(msg: any) {
    // Пропускаем сообщения не от Voice Service
    // Voice Service шлёт только известные типы — фильтруем остальные
    const voiceTypes = new Set([
      "room_state",
      "peer_joined",
      "peer_left",
      "peer_muted",
      "offer",
      "answer",
      "ice_candidate",
      "error",
    ]);

    if (!voiceTypes.has(msg.type)) return;

    switch (msg.type) {
      case "room_state":
        this.events.onRoomState?.(msg.payload.peers ?? []);
        break;

      case "peer_joined":
        this.events.onPeerJoined?.(msg.payload.peer);
        break;

      case "peer_left":
        this.remoteStreams.delete(msg.payload.peer_id);
        this.events.onPeerLeft?.(msg.payload.peer_id);
        break;

      case "peer_muted":
        this.events.onPeerMuted?.(msg.payload.peer_id, msg.payload.muted);
        break;

      // Re-offer от сервера (когда добавился новый участник)
      case "offer":
        console.log(
          "[Voice ←] re-offer received, signalingState:",
          this.pc?.signalingState,
        );
        await this.handleRemoteOffer(msg.payload.sdp);
        break;

      // Answer от сервера на наш первичный offer
      case "answer":
        await this.handleRemoteAnswer(msg.payload.sdp);
        break;

      case "ice_candidate":
        await this.handleRemoteICE(msg.payload);
        break;

      case "error":
        console.error(
          "[voice] server error:",
          msg.payload.code,
          msg.payload.message,
        );
        this.events.onError?.(msg.payload.code, msg.payload.message);
        break;
    }
  }

  private async handleRemoteOffer(sdp: string) {
    if (!this.pc) return;

    // Если идёт наш offer — делаем rollback и принимаем серверный
    if (this.pc.signalingState === "have-local-offer") {
      console.log("[Voice] glare: rolling back local offer");
      try {
        await this.pc.setLocalDescription({ type: "rollback" });
        this.isNegotiating = false;
        this.pendingOffer = null;
      } catch (err) {
        console.error("[voice] rollback failed:", err);
        this.pendingOffer = sdp;
        return;
      }
    }

    await this.applyRemoteOffer(sdp);
  }

  private async applyRemoteOffer(sdp: string) {
    if (!this.pc) return;
    try {
      await this.pc.setRemoteDescription({ type: "offer", sdp });
      const answer = await this.pc.createAnswer();
      await this.pc.setLocalDescription(answer);

      sendWS({
        type: "answer",
        payload: { sdp: answer.sdp },
      });
    } catch (err) {
      console.error("[voice] handle remote offer:", err);
    }
  }

  private async handleRemoteAnswer(sdp: string) {
    if (!this.pc) return;
    try {
      await this.pc.setRemoteDescription({ type: "answer", sdp });
      this.isNegotiating = false;

      if (this.pendingOffer) {
        const offer = this.pendingOffer;
        this.pendingOffer = null;
        console.log("[Voice] applying pending offer after answer");
        await this.applyRemoteOffer(offer);
      }
    } catch (err) {
      console.error("[voice] handle remote answer:", err);
      this.isNegotiating = false;
    }
  }

  private async handleRemoteICE(payload: {
    candidate: string;
    sdp_mid: string;
    sdp_mline_index: number;
  }) {
    if (!this.pc) return;
    try {
      await this.pc.addIceCandidate({
        candidate: payload.candidate,
        sdpMid: payload.sdp_mid,
        sdpMLineIndex: payload.sdp_mline_index,
      });
    } catch (err) {
      console.error("[voice] add ICE candidate:", err);
    }
  }

  private cleanup() {
    this.unsubscribe?.();
    this.unsubscribe = null;

    this.localStream?.getTracks().forEach((t) => t.stop());
    this.localStream = null;

    this.pc?.close();
    this.pc = null;

    this.remoteStreams.clear();
    this.currentRoomId = null;

    this.events.onDisconnected?.();
    this.isNegotiating = false;
    this.pendingOffer = null;
  }
}

// Синглтон — один голосовой клиент на всё приложение
export const voiceClient = new VoiceClient();
