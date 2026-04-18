import { request } from './api';

export interface User {
  id: string;
  email: string;
  name: string;
  role_id: string;
}

interface AuthResponse {
  user: User;
  token: string;
}

export async function login(email: string, password: string): Promise<User> {
  const { user, token } = await request<AuthResponse>('/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });

  localStorage.setItem('oblivra_token', token);
  localStorage.setItem('oblivra_user', JSON.stringify(user));
  return user;
}

export async function logout() {
  await request('/auth/logout', { method: 'POST' }).catch(() => {});
  localStorage.removeItem('oblivra_token');
  localStorage.removeItem('oblivra_user');
}

export function getCurrentUser(): User | null {
  const user = localStorage.getItem('oblivra_user');
  if (!user) return null;
  try {
    return JSON.parse(user);
  } catch {
    return null;
  }
}

export function isAuthenticated(): boolean {
  return localStorage.getItem('oblivra_token') !== null;
}
