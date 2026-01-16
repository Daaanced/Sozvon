import { request } from './http'

export function login(login: string, password: string) {
  return request('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ login, password })
  })
}
