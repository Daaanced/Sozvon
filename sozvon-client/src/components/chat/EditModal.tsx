// sozvon-client/src/components/chat/EditModal.tsx
import { styles } from "./chat.styles";

type Props = {
  value: string;
  onChange: (v: string) => void;
  onSave: () => void;
  onCancel: () => void;
};

export default function EditModal({
  value,
  onChange,
  onSave,
  onCancel,
}: Props) {
  return (
    <div style={styles.modalOverlay}>
      <div style={styles.modalContent}>
        <div style={styles.modalTitle}>Редактировать сообщение</div>

        <textarea
          value={value}
          onChange={(e) => onChange(e.target.value)}
          rows={4}
          style={styles.modalTextarea}
        />

        <div style={styles.modalActions}>
          <button onClick={onCancel} style={styles.modalCancelBtn}>
            Отмена
          </button>
          <button onClick={onSave} style={styles.modalSaveBtn}>
            Сохранить
          </button>
        </div>
      </div>
    </div>
  );
}
