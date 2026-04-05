//sozvon-client\src\api\http.ts
const API_URL = "http://92.127.169.188:8080";

export async function request(path: string, options: RequestInit = {}) {
  const res = await fetch(API_URL + path, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

// Запрос с автоматической подстановкой токена
export async function requestAuth(path: string, options: RequestInit = {}) {
  const token = localStorage.getItem("token");
  const res = await fetch(API_URL + path, {
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...options,
  });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}

export async function requestForm(path: string, options: RequestInit = {}) {
  const res = await fetch(API_URL + path, { ...options });
  if (!res.ok) throw new Error(await res.text());
  return res.json();
}
