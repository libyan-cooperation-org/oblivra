import { Component, For, Show, createSignal, createMemo, onMount, onCleanup } from 'solid-js';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';
import { useApp } from '@core/store';
import { DeployKey } from '../../../wailsjs/go/app/SSHService';
import { WakeHost } from '../../../wailsjs/go/app/HostService';
import { database } from '../../../wailsjs/go/models';

type Host = database.Host;

interface HostTreeProps {
  searchQuery: string;
  onAddHost?: () => void;
  onEditHost?: (hostId: string) => void;
  onDeleteHost?: (hostId: string) => void;
}

export const HostTree: Component<HostTreeProps> = (props) => {
  const [state, actions] = useApp();
  const [contextMenu, setContextMenu] = createSignal<{ x: number; y: number; hostId: string; } | null>(null);
  const [expandedGroups, setExpandedGroups] = createSignal<Set<string>>(new Set(['Uncategorized']));
  const [groupBy, setGroupBy] = createSignal<'folder' | 'tags' | 'status'>('folder');
  const [hostHealth, setHostHealth] = createSignal<Record<string, boolean>>({});

  onMount(() => {
    EventsOn('host-health-update', (data: unknown) => {
      setHostHealth(data as Record<string, boolean>);
    });
  });

  onCleanup(() => {
    EventsOff('host-health-update');
  });

  const toggleGroup = (name: string) => {
    setExpandedGroups(prev => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  };

  const filteredHosts = createMemo(() => {
    const query = props.searchQuery.toLowerCase();
    if (!query) return state.hosts as unknown as Host[];
    return (state.hosts as unknown as Host[]).filter(
      (h) => h.label.toLowerCase().includes(query) ||
        h.hostname.toLowerCase().includes(query) ||
        (h.tags && h.tags.some((t) => t.toLowerCase().includes(query)))
    );
  });

  const favorites = createMemo(() => filteredHosts().filter((h) => h.is_favorite));
  const recents = createMemo(() => [...filteredHosts()].sort((a, b) =>
    (b.connection_count || 0) - (a.connection_count || 0)
  ).slice(0, 5));

  const activeSessionHosts = createMemo(() => {
    const activeIds = state.sessions.filter(s => s.status === 'active').map(s => s.hostId);
    return filteredHosts().filter(h => activeIds.includes(h.id));
  });

  const groupedHosts = createMemo(() => {
    const hosts = filteredHosts();
    switch (groupBy()) {
      case 'folder': {
        const groups: Record<string, Host[]> = { 'Uncategorized': [] };
        hosts.forEach((h) => {
          if (!h.category || h.category.trim() === '') {
            groups['Uncategorized'].push(h);
          } else {
            const cat = h.category.trim();
            if (!groups[cat]) groups[cat] = [];
            groups[cat].push(h);
          }
        });
        return groups;
      }
      case 'status': {
        const activeIds = state.sessions.filter(s => s.status === 'active').map(s => s.hostId);
        return {
          'Online': hosts.filter(h => activeIds.includes(h.id)),
          'Offline': hosts.filter(h => !activeIds.includes(h.id))
        };
      }
      case 'tags': {
        const groups: Record<string, Host[]> = {};
        hosts.forEach((h) => {
          if (!h.tags || h.tags.length === 0) {
            if (!groups['Untagged']) groups['Untagged'] = [];
            groups['Untagged'].push(h);
          } else {
            h.tags.forEach(tag => {
              // Capitalize tag for cleaner visual
              const fTag = tag.charAt(0).toUpperCase() + tag.slice(1);
              if (!groups[fTag]) groups[fTag] = [];
              groups[fTag].push(h);
            });
          }
        });
        return groups;
      }
      default:
        return { 'All Hosts': hosts };
    }
  });

  const handleConnect = (hostId: string) => {
    actions.connectToHost(hostId);
    actions.setActiveNavTab('terminal');
  };

  const handleContextMenu = (e: MouseEvent, hostId: string) => {
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY, hostId });
  };

  const isActive = (hostId: string) => state.sessions.some(s => s.hostId === hostId && s.status === 'active');

  const HostRow = (p: { host: Host }) => (
    <div
      class={`host-item ${isActive(p.host.id) ? 'selected' : ''}`}
      onClick={() => handleConnect(p.host.id)}
      onContextMenu={(e) => handleContextMenu(e, p.host.id)}
    >
      <span class={`status-dot ${isActive(p.host.id) ? 'online' : (hostHealth()[p.host.id] ? 'reachable' : 'offline')}`} />
      <div class="host-info">
        <span class="host-label">{p.host.label || p.host.hostname}</span>
        <span class="host-address">{p.host.username ? `${p.host.username}@` : ''}{p.host.hostname}{p.host.port && p.host.port !== 22 ? `:${p.host.port}` : ''}</span>
      </div>
      <Show when={p.host.tags && p.host.tags.length > 0}>
        <div class="host-tags">
          <For each={(p.host.tags || []).slice(0, 2)}>
            {(tag) => <span class="tag-pill">{tag}</span>}
          </For>
        </div>
      </Show>
    </div>
  );

  const GroupSection = (p: { name: string; icon: string; hosts: Host[]; defaultOpen?: boolean }) => (
    <Show when={p.hosts.length > 0}>
      <div class="host-group">
        <div class="group-header" onClick={() => toggleGroup(p.name)}>
          <span style="font-size: 10px; color: var(--text-muted); width: 14px; text-align: center;">
            {expandedGroups().has(p.name) ? '▼' : '▶'}
          </span>
          <span class="group-icon">{p.icon}</span>
          {p.name}
          <span class="host-count">{p.hosts.length}</span>
        </div>
        <Show when={expandedGroups().has(p.name)}>
          <div class="group-items">
            <For each={p.hosts}>
              {(host) => <HostRow host={host} />}
            </For>
          </div>
        </Show>
      </div>
    </Show>
  );

  return (
    <div class="host-tree" onClick={() => setContextMenu(null)}>
      {/* Host Tree Controls */}
      <div class="host-tree-controls">
        <select value={groupBy()} onChange={(e) => setGroupBy(e.currentTarget.value as any)}>
          <option value="folder">Group by Folder (Category)</option>
          <option value="tags">Group by Smart Tags</option>
          <option value="status">Group by Status</option>
        </select>
      </div>

      {/* Smart Folders */}
      <Show when={favorites().length > 0 || activeSessionHosts().length > 0}>
        <div class="section-label">Smart Folders</div>
        <GroupSection name="Favorites" icon="⭐" hosts={favorites()} />
        <GroupSection name="Live Sessions" icon="🟢" hosts={activeSessionHosts()} />
        <GroupSection name="Recent Connections" icon="🕒" hosts={recents()} />
      </Show>

      {/* Server Groups */}
      <div class="section-label">Servers</div>
      <For each={Object.entries(groupedHosts())}>
        {([groupName, groupHosts]) => (
          <GroupSection name={groupName} icon="📂" hosts={groupHosts} />
        )}
      </For>

      {/* Loading State */}
      <Show when={state.loading}>
        <div class="host-tree-loading">
          <For each={[1, 2, 3, 4]}>
            {() => (
              <div class="skeleton-row">
                <div class="skeleton-dot" />
                <div class="skeleton-text" />
              </div>
            )}
          </For>
        </div>
      </Show>

      {/* Empty State */}
      <Show when={!state.loading && filteredHosts().length === 0}>
        <div class="sidebar-empty-state">
          <div class="sidebar-empty-icon">🖥️</div>
          <div class="sidebar-empty-title">No hosts found</div>
          <div class="sidebar-empty-desc">{props.searchQuery ? 'Try a different search query' : 'Click the + icon in the header to add your first server'}</div>
        </div>
      </Show>

      {/* Context Menu */}
      <Show when={contextMenu()}>
        {(menu) => (
          <div
            class="context-menu"
            style={{
              position: 'fixed',
              top: `${menu().y}px`,
              left: `${menu().x}px`,
            }}
          >
            <div class="menu-item" onClick={() => { handleConnect(menu().hostId); setContextMenu(null); }}>
              ⚡ Connect
            </div>
            <div class="menu-item" onClick={() => setContextMenu(null)}>
              📝 Run Snippet
            </div>
            <div class="menu-item" onClick={async () => {
              const hostId = menu().hostId;
              const password = prompt("Enter SSH Password to deploy key:");
              if (!password) return;
              try {
                await DeployKey(hostId, password);
                alert("SSH Key deployed successfully! Host is now configured for keyed login.");
              } catch (err) {
                alert("Failed to deploy key: " + err);
              }
              setContextMenu(null);
            }}>
              🔑 Deploy SSH Key
            </div>

            <Show when={state.hosts.find(h => h.id === menu().hostId)?.tags?.some(t => t.startsWith('mac:'))}>
              <div class="menu-item" onClick={async () => {
                try {
                  await WakeHost(menu().hostId);
                  alert("WOL packet sent!");
                } catch (err) {
                  alert("Failed to wake host: " + err);
                }
                setContextMenu(null);
              }}>
                📡 Wake-on-LAN
              </div>
            </Show>

            <div class="menu-divider" />
            <div class="menu-item" onClick={() => { props.onEditHost?.(menu().hostId); setContextMenu(null); }}>
              ⚙️ Edit Host
            </div>
            <div class="menu-item" onClick={() => { props.onDeleteHost?.(menu().hostId); setContextMenu(null); }} style="color: var(--error);">
              🗑 Delete
            </div>
          </div>
        )}
      </Show>
    </div>
  );
};