//sozvon-client\src\components\chat\chat.types.ts

export type Attachment = {
  id: string
  fileName: string
  mimeType: string
  size: number
  url: string
}

export type Message = {
  id: string
  senderId: number
  text: string
  attachments?: Attachment[]
  createdAt: string
}

export type PendingFile = {
  file: File
  previewUrl: string | null
}