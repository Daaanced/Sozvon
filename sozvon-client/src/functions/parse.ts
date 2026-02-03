// sozvon-client\src\functions\parse.tsx

export function parseToken(token: string): string | null {
  try {
    const payload = token.split('.')[1]
    const decoded = atob(payload.replace(/-/g, '+').replace(/_/g, '/'))
    const obj = JSON.parse(decoded)
    return obj.login || null
  } catch {
    return null
  }
}