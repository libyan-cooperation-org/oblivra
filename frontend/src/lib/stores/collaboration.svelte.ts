/**
 * OBLIVRA — Collaboration Store (Svelte 5 runes)
 *
 * Manages real-time analyst presence and tactical communication in the War Room.
 */
import { subscribe, emitLocal } from '@lib/bridge';

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
  analysts = $state<Analyst[]>([
    { id: 'A1', name: 'Maverick', role: 'Lead Responder', status: 'active', color: '#5aaef0', lastActive: new Date().toISOString() },
    { id: 'A2', name: 'Phoenix', role: 'Forensic Analyst', status: 'active', color: '#9878e0', lastActive: new Date().toISOString() },
    { id: 'A3', name: 'Iceman', role: 'Network Security', status: 'idle', color: '#1aaa60', lastActive: new Date().toISOString() }
  ]);

  messages = $state<TacticalMessage[]>([
    { id: 'M1', analystId: 'A2', text: 'Analyzing memory dump for FIN-SRV-07', timestamp: new Date().toISOString(), type: 'action' },
    { id: 'M2', analystId: 'A3', text: 'Egress traffic to 185.x.x.x blocked', timestamp: new Date().toISOString(), type: 'chat' }
  ]);

  constructor() {
    this.init();
  }

  init() {
    subscribe('presence.update', (data: Analyst) => {
        const idx = this.analysts.findIndex(a => a.id === data.id);
        if (idx === -1) {
            this.analysts = [...this.analysts, data];
        } else {
            this.analysts[idx] = { ...this.analysts[idx], ...data };
        }
    });

    subscribe('collab.message', (msg: TacticalMessage) => {
        this.messages = [...this.messages, msg];
    });
  }

  sendMessage(text: string, type: TacticalMessage['type'] = 'chat') {
    const msg: TacticalMessage = {
        id: `msg-${Date.now()}`,
        analystId: 'A1', // Self
        text,
        timestamp: new Date().toISOString(),
        type
    };
    this.messages = [...this.messages, msg];
    // Emit to backend for broadcast
    emitLocal('collab.message', msg);
  }

  updateStatus(status: Analyst['status']) {
    const self = this.analysts.find(a => a.id === 'A1');
    if (self) self.status = status;
  }
}

export const collabStore = new CollaborationStore();
