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
}

/**
 * Handle login (Browser mode only)
 */
export async function login(email: string, password: string): Promise<User> {
  // CS-02: Authenticate and receive HttpOnly cookies
  const response = await fetch('/api/v1/auth/login', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ email, password }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Authentication failed' }));
    throw new Error(error.message || 'Authentication failed');
  }

  const { user } = await response.json() as AuthResponse;

  // We only store the user metadata, not the token.
  // The token is now in a secure HttpOnly cookie.
  localStorage.setItem('oblivra_user', JSON.stringify(user));
  
  return user;
}

/**
 * Handle logout (Browser mode only)
 */
export async function logout() {
  if (IS_BROWSER) {
    await fetch('/api/v1/auth/logout', { 
      method: 'POST',
      credentials: 'include' // Ensure cookies are sent and cleared
    }).catch(() => {});
    
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
  // We check for the user metadata existence. The actual session
  // validity is enforced by the backend using HttpOnly cookies.
  return localStorage.getItem('oblivra_user') !== null;
}
