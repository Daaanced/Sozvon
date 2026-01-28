//sozvon-client\src\api\users.ts
import { request } from './http'

// src/api/users.ts
export interface User {
  login: string
  name: string
  email: string
  info: string
  picture: string
}

export function searchUser(login: string): Promise<User> {
  return request(`/users/${encodeURIComponent(login)}`)
}

