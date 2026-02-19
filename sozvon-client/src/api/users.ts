//sozvon-client\src\api\users.ts
import { request } from './http'

export interface User {
  id: number
  login: string
  name: string
  email: string
  info: string
  picture: string
  created_at: string
  updated_at: string
}


export function searchUser(login: string): Promise<User> {
  return request(`/users/${encodeURIComponent(login)}`)
}

// ===== Update text fields =====
export function updateUser(login: string, data: { name?: string, email?: string, info?: string }): Promise<void> {
  return request(`/users/${encodeURIComponent(login)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data)
  })
}

// ===== Upload avatar =====
export function uploadAvatar(login: string, file: File): Promise<{ avatar_url: string }> {
  const formData = new FormData()
  formData.append('avatar', file)

  return fetch(`http://176.51.121.88:8080/users/${encodeURIComponent(login)}/avatar`, {
    method: 'POST',
    body: formData
  }).then(async res => {
    if (!res.ok) throw new Error(await res.text())
    return res.json()
  })
}