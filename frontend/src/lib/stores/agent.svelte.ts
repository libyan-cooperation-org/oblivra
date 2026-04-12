import { IS_BROWSER } from '@lib/context';

export interface AgentDTO {
  id: string;
  hostname: string;
  version: string;
  last_seen: string;
  remote_address: string;
  status: 'online' | 'offline';
}

export class AgentStore {
  agents = $state<AgentDTO[]>([]);
  loading = $state(false);
  error = $state<string | null>(null);

  constructor() {
    if (!IS_BROWSER) {
      this.refresh();
      setInterval(() => this.refresh(), 10000); // 10s heartbeat check
    }
  }

  async refresh() {
    if (IS_BROWSER) return;
    this.loading = true;
    try {
      const { ListAgents } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/agentservice');
      const list = await ListAgents();
      this.agents = list || [];
      this.error = null;
    } catch (err: any) {
      console.error('[AgentStore] Refresh failed:', err);
      this.error = 'Failed to sync with agent fleet';
    } finally {
      this.loading = false;
    }
  }

  async killProcess(agentID: string, pid: number) {
    if (IS_BROWSER) return;
    try {
      const { KillProcess } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/agentservice');
      await KillProcess(agentID, pid);
      // We don't wait for success here since it's an async C2 operation
      return true;
    } catch (err: any) {
      console.error('[AgentStore] KillProcess failed:', err);
      throw err;
    }
  }

  async toggleQuarantine(agentID: string, enabled: boolean) {
    if (IS_BROWSER) return;
    try {
      const { ToggleQuarantine } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/agentservice');
      await ToggleQuarantine(agentID, enabled);
      return true;
    } catch (err: any) {
      console.error('[AgentStore] ToggleQuarantine failed:', err);
      throw err;
    }
  }
}

export const agentStore = new AgentStore();
