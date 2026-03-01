//sozvon-client\src\api\http.ts
const API_URL = 'http://92.127.177.190:8080'

export async function request(
  path: string,
  options: RequestInit = {}
) {
  const res = await fetch(API_URL + path, {
    headers: {
      'Content-Type': 'application/json'
    },
    ...options
  })

  if (!res.ok) {
    throw new Error(await res.text())
  }

  return res.json()
}

export async function requestForm(
  path: string,
  options: RequestInit = {}
) {
  const res = await fetch(API_URL + path, {
    ...options
  })

  if (!res.ok) {
    throw new Error(await res.text())
  }

  return res.json()
}