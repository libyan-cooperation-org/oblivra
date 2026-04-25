import { subscribe, send } from '@lib/bridge';
import { appStore } from './app.svelte';

export interface Analyst {
  id: string;
  name: string;
  role: string;
  status: 'active' | 'idle' | 'away';
  color: string;
  avatar?: string;
  lastActive: string;
}

export interface TacticalMessage {
  id: string;
  analystId: string;
  text: string;
  timestamp: string;
  type: 'chat' | 'action' | 'system';
}

export class CollaborationStore {
  analysts = $state<Analyst[]>([]);

  messages = $state<TacticalMessage[]>([]);

  constructor() {
    this.init();
  }

  init() {
    // If no analysts yet, add self as active
    $effect.root(() => {
      $effect(() => {
          if (appStore.currentUser && this.analysts.length === 0) {
              this.analysts = [{
                  id: appStore.currentUser.id,
                  name: appStore.currentUser.username || appStore.currentUser.name,
                  role: appStore.currentUser.role || 'Operator',
                  status: 'active',
                  color: '#5aaef0',
                  lastActive: new Date().toISOString()
              }];
              // Announce presence
              send('presence.update', this.analysts[0]);
          }
      });
    });

    subscribe('presence.update', (data: Analyst) => {
        const idx = this.analysts.findIndex(a => a.id === data.id);
        if (idx === -1) {
            this.analysts = [...this.analysts, data];
        } else {
            this.analysts[idx] = { ...this.analysts[idx], ...data };
        }
    });

    subscribe('collab.message', (msg: TacticalMessage) => {
        // Prevent duplicate local messages if echoed back by server
        if (this.messages.find(m => m.id === msg.id)) return;
        this.messages = [...this.messages, msg];
    });
  }

  sendMessage(text: string, type: TacticalMessage['type'] = 'chat') {
    if (!appStore.currentUser) return;
    const msg: TacticalMessage = {
        id: `msg-${Date.now()}-${Math.random().toString(36).substr(2, 5)}`,
        analystId: appStore.currentUser.id,
        text,
        timestamp: new Date().toISOString(),
        type
    };
    this.messages = [...this.messages, msg];
    // Send to backend for global broadcast
    send('collab.message', msg);
  }

  updateStatus(status: Analyst['status']) {
    if (!appStore.currentUser) return;
    const self = this.analysts.find(a => a.id === appStore.currentUser.id);
    if (self) {
        self.status = status;
        send('presence.update', self);
    }
  }
}

export const collabStore = new CollaborationStore();
