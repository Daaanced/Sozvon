//sozvon-client\src\components\chat\chat.types.ts

export type Attachment = {
  id: string;
  fileName: string;
  mimeType: string;
  size: number;
  url: string;
};

export type ReplyPreview = {
  id: string;
  senderId: number;
  text: string;
};

export type ForwardedMeta = {
  originalMessageId?: string;
  senderId: number;
  text: string;
  attachments?: Attachment[];
};

export type Message = {
  id: string;
  senderId: number;
  text: string;
  replyToId?: string;
  replyToMessage?: ReplyPreview;
  forwardedFrom?: ForwardedMeta;
  editedAt?: string;
  deletedAt?: string;
  attachments?: Attachment[];
  createdAt: string;
};

export type PendingFile = {
  file: File;
  previewUrl: string | null;
};
