//sozvon-client\src\components\chat\Lightbox.tsx

import { styles } from "./chat.styles";

type Props = {
  url: string;
  onClose: () => void;
};

export default function Lightbox({ url, onClose }: Props) {
  return (
    <div style={styles.lightboxOverlay} onClick={onClose}>
      <img
        src={url}
        style={styles.lightboxImage}
        onClick={(e) => e.stopPropagation()}
      />
      <button style={styles.lightboxClose} onClick={onClose}>
        ✕
      </button>
    </div>
  );
}
