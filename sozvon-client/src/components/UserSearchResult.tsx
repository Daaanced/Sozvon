//sozvon-client\src\components\UserSearchResult.tsx
type Props = {
  login: string
  picture: string
  onChat: () => void
  onCall: () => void
}

export default function UserSearchResult({ login, picture, onChat, onCall }: Props) {
  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        padding: 12,
        border: '1px solid #ccc',
        borderRadius: 8,
        marginTop: 12,
        width: 400
      }}
    >
      {/* Avatar */}
      <div
        style={{
          width: 48,
          height: 48,
          borderRadius: '50%',
          overflow: 'hidden',
          background: '#ddd',
          flexShrink: 0
        }}
      >
        <img
          src={picture}
          alt={login}
          style={{
            width: '100%',
            height: '100%',
            objectFit: 'cover' // ✅
          }}
          onError={(e) => {
  			e.currentTarget.onerror = null
  			e.currentTarget.src = 'http://176.51.121.88:8080/static/avatars/default.png'
          }}
        />
      </div>

      <div style={{ flex: 1 }}>
        <div><b>{login}</b></div>
      </div>

      <button onClick={onChat}>Chat</button>
      <button onClick={onCall}>Call</button>
    </div>
  )
}