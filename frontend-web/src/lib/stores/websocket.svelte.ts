/**
 * OBLIVRA — Tactical WebSocket Stream (Svelte 5 Runes)
 * Manages real-time telemetry and global notification events.
 */

import { appStore } from './app.svelte';

export class WebSocketStream {
  url: string;
  socket: WebSocket | null = null;
  status = $state<'CONNECTING' | 'OPEN' | 'CLOSED'>('CLOSED');
  reconnectAttempts = 0;
  maxReconnectAttempts = 5;

  constructor(url: string = 'ws://localhost:8080/api/v1/ws') {
    this.url = url;
  }

  connect() {
    if (this.socket?.readyState === WebSocket.OPEN) return;

    this.status = 'CONNECTING';
    try {
      this.socket = new WebSocket(this.url);

      this.socket.onopen = () => {
        console.log('Tactical WebSocket Stream established');
        this.status = 'OPEN';
        this.reconnectAttempts = 0;
        appStore.notify('Secure telemetry stream established', 'info');
      };

      this.socket.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          this.handleMessage(data);
        } catch (e) {
          console.error('Failed to parse WS message', e);
        }
      };

      this.socket.onclose = () => {
        this.status = 'CLOSED';
        this.attemptReconnect();
      };

      this.socket.onerror = (error) => {
        console.error('WebSocket Error', error);
        this.socket?.close();
      };
    } catch (e) {
      this.status = 'CLOSED';
      this.attemptReconnect();
    }
  }

  private handleMessage(data: any) {
    // Standard tactical message format: { type: string, payload: any }
    switch (data.type) {
      case 'SECURITY_ALERT':
        appStore.notify(`CRITICAL_ALERT: ${data.payload.message}`, 'error');
        break;
      case 'TELEMETRY_UPDATE':
        // Update global telemetry store if needed
        break;
      case 'SYSTEM_HEALTH':
        appStore.setHealth(data.payload.status);
        break;
      default:
        console.debug('Received unhandled WS message type', data.type);
    }
  }

  /**
   * Sends a tactical command to the backend event substrate.
   */
  sendCommand(command: string, payload: any = {}) {
    if (this.socket?.readyState === WebSocket.OPEN) {
      console.log(`Emitting tactical command: ${command}`);
      this.socket.send(JSON.stringify({
        type: 'COMMAND_REQUEST',
        command,
        payload,
        timestamp: new Date().toISOString()
      }));
    } else {
      appStore.notify(`Command failed: Stream offline (${command})`, 'warning');
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      this.reconnectAttempts++;
      const delay = Math.pow(2, this.reconnectAttempts) * 1000;
      console.warn(`Attempting WS reconnection ${this.reconnectAttempts}/${this.maxReconnectAttempts} in ${delay}ms`);
      setTimeout(() => this.connect(), delay);
    } else {
      appStore.notify('Live telemetry stream disconnected. Manual refresh required.', 'warning');
    }
  }
}

// Singleton instance
export const wsStream = new WebSocketStream();
