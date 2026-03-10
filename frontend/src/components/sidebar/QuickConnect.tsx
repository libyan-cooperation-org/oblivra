import { Component, createSignal } from 'solid-js';

export const QuickConnect: Component = () => {
  const [value, setValue] = createSignal('');

  const handleConnect = () => {
    const input = value().trim();
    if (!input) return;

    // Parse: user@host:port or host or user@host
    let username = '';
    let hostname = input;
    let port = 22;

    if (input.includes('@')) {
      const parts = input.split('@');
      username = parts[0];
      hostname = parts[1];
    }

    if (hostname.includes(':')) {
      const parts = hostname.split(':');
      hostname = parts[0];
      port = parseInt(parts[1], 10) || 22;
    }

    console.log(`Quick connect: ${username}@${hostname}:${port}`);
    // TODO: Create temporary host and connect
    setValue('');
  };

  return (
    <div class="quick-connect">
      <div class="quick-connect-input">
        <span class="quick-connect-icon">⚡</span>
        <input
          type="text"
          placeholder="user@host:port"
          value={value()}
          onInput={(e) => setValue(e.currentTarget.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') handleConnect();
          }}
        />
        <button
          class="quick-connect-btn"
          onClick={handleConnect}
          disabled={!value().trim()}
        >
          →
        </button>
      </div>
    </div>
  );
};