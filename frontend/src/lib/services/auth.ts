/**
 * OBLIVRA — Auth Service (Svelte/Web)
 */
import { IS_BROWSER } from '../context';

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

/**
 * Handle login (Browser mode only)
 */
export async function login(email: string, password: string): Promise<User> {
  // In the real app, this would be a fetch to the backend.
  // For now, mirroring the SolidJS logic but with fetch.
  const response = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    const error = await response.json();
    throw new Error(error.message || 'Authentication failed');
  }

  const { user, token } = await response.json() as AuthResponse;

  localStorage.setItem('oblivra_token', token);
  localStorage.setItem('oblivra_user', JSON.stringify(user));
  
  return user;
}

/**
 * Handle logout (Browser mode only)
 */
export async function logout() {
  if (IS_BROWSER) {
    await fetch('/api/v1/auth/logout', { method: 'POST' }).catch(() => {});
    localStorage.removeItem('oblivra_token');
    localStorage.removeItem('oblivra_user');
  }
}

/**
 * Get current authenticated user (Browser mode only)
 */
export function getCurrentUser(): User | null {
  if (!IS_BROWSER) return null;
  const user = localStorage.getItem('oblivra_user');
  if (!user) return null;
  try {
    return JSON.parse(user);
  } catch {
    return null;
  }
}

/**
 * Check if authenticated (Browser mode only)
 */
export function isAuthenticated(): boolean {
  if (!IS_BROWSER) return true; // Desktop mode uses VaultLocked barrier
  return localStorage.getItem('oblivra_token') !== null;
}
