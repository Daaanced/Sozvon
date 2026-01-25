//sozvon-client\src\api\users.ts
import { request } from './http'

export interface User {
  id: string
  login: string
}

export function searchUser(login: string): Promise<User> {
  return request(`/users/${encodeURIComponent(login)}`)
}

