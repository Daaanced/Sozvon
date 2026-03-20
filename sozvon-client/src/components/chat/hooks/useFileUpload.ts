// sozvon-client/src/components/chat/hooks/useFileUpload.ts
import { useState, useCallback } from "react";
import type { PendingFile } from "../chat.types";

export function useFileUpload() {
  const [pendingFiles, setPendingFiles] = useState<PendingFile[]>([]);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);

  const handleFilesAdded = useCallback((newFiles: PendingFile[]) => {
    setPendingFiles((prev) => [...prev, ...newFiles]);
  }, []);

  const handleFileRemove = useCallback((index: number) => {
    setPendingFiles((prev) => {
      const updated = [...prev];
      if (updated[index].previewUrl)
        URL.revokeObjectURL(updated[index].previewUrl!);
      updated.splice(index, 1);
      return updated;
    });
  }, []);

  const clearFiles = useCallback(() => {
    setPendingFiles((prev) => {
      prev.forEach((pf) => {
        if (pf.previewUrl) URL.revokeObjectURL(pf.previewUrl!);
      });
      return [];
    });
  }, []);

  return {
    pendingFiles,
    uploading,
    uploadProgress,
    setUploading,
    setUploadProgress,
    handleFilesAdded,
    handleFileRemove,
    clearFiles,
  };
}
