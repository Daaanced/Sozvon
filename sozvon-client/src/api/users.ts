// sozvon-client/src/api/users.ts
import { request, requestForm } from "./http";

export interface User {
  id: number;
  login: string;
  name: string;
  email: string;
  info: string;
  picture: string;
  created_at: string;
  updated_at: string;
}

export interface UpdateUserPayload {
  name?: string;
  email?: string;
  info?: string;
}

export function searchUser(login: string): Promise<User> {
  return request(`/users/${encodeURIComponent(login)}`);
}

export function getUserById(id: number): Promise<User> {
  return request(`/users/id/${id}`);
}

export function updateUser(login: string, data: UpdateUserPayload) {
  return request(`/users/${encodeURIComponent(login)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
}

export function uploadAvatar(
  login: string,
  file: File,
): Promise<{ avatar_url: string }> {
  const formData = new FormData();
  formData.append("avatar", file);
  return requestForm(`/users/${login}/avatar`, {
    method: "POST",
    body: formData,
  });
}

export function deleteAvatar(login: string) {
  return request(`/users/${encodeURIComponent(login)}/avatar`, {
    method: "DELETE",
  });
}

export function deleteUser(login: string) {
  return request(`/auth/users/${encodeURIComponent(login)}`, {
    method: "DELETE",
  });
}
