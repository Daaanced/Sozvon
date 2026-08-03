//sozvon-client\src\api\http.ts
export const API_URL =
  import.meta.env.VITE_API_URL ?? "http://localhost:8080/api";

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
