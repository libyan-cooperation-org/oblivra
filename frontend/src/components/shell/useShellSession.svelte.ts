// useShellSession — small helper used by every Blacknode-port panel that
// needs an SSH session to the operator-selected host.
//
// Resolution order for a "running" session id:
//   1. shellStore.selectedHostID has an active SSH session in the workspace
//      → reuse that session (no new handshake).
//   2. shellStore.selectedHostID is set but no session is active
//      → call SSHService.Connect(hostID) to open one, return its id.
//   3. selectedHostID is null → return null and let the panel render its
//      "Pick a host" empty state.
//
// All panels share this so the user only does one host pick.

import { shellStore } from '@lib/stores/shell.svelte';

export async function ensureSessionForSelectedHost(): Promise<string | null> {
  const hostID = shellStore.selectedHostID;
  if (!hostID) return null;
  try {
    const ssh = await import(
      '@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice'
    );
    // Reuse an existing session if any (no handshake cost).
    const existing = (await ssh.GetActiveSessionForHost(hostID)) as string;
    if (existing) return existing;
    const fresh = (await ssh.Connect(hostID)) as string;
    return fresh || null;
  } catch (e) {
    console.warn('[shell.session] resolve failed for host', hostID, e);
    return null;
  }
}

/** Run a single command via SSH on the resolved session. Throws on failure. */
export async function execOnHost(cmd: string): Promise<string> {
  const sid = await ensureSessionForSelectedHost();
  if (!sid) throw new Error('No host selected. Pick one in the sidebar.');
  const { Exec } = await import(
    '@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice'
  );
  const out = (await Exec(sid, cmd)) as string;
  return out ?? '';
}
