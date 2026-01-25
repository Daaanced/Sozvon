//sozvon-client\src\api\auth.ts
import { request } from './http'

export function login(login: string, password: string) {
  return request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ login, password })
  })
}

export function register(login: string, password: string) {
  return request('/auth/register', {
    method: 'POST',
    body: JSON.stringify({ login, password })
  })
}
