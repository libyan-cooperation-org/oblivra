import { For, createSignal, onMount } from 'solid-js';

interface FleetNode {
  id: string;
  name: string;
  tenant: string;
  platform: 'linux' | 'windows' | 'darwin';
  status: 'online' | 'degraded' | 'offline';
  ip: string;
}

export default function FleetOverview() {
  const [nodes, setNodes] = createSignal<FleetNode[]>([]);

  onMount(() => {
    const mockNodes: FleetNode[] = [
      { id: '1', name: 'PRD-LNX-01', tenant: 'GLOBAL_CORP', platform: 'linux', status: 'online', ip: '192.168.1.10' },
      { id: '2', name: 'PRD-LNX-02', tenant: 'GLOBAL_CORP', platform: 'linux', status: 'online', ip: '192.168.1.11' },
      { id: '3', name: 'PRD-WIN-01', tenant: 'GLOBAL_CORP', platform: 'windows', status: 'degraded', ip: '192.168.1.12' },
      { id: '4', name: 'DEV-DAR-01', tenant: 'INNOVATION', platform: 'darwin', status: 'offline', ip: '10.0.4.45' },
      { id: '5', name: 'STG-LNX-01', tenant: 'GLOBAL_CORP', platform: 'linux', status: 'online', ip: '192.168.5.20' },
    ];
    setNodes(mockNodes);
  });

  const getPlatformIcon = (p: string) => {
    switch (p) {
      case 'windows': return '🪟';
      case 'linux': return '🐧';
      case 'darwin': return '🍎';
      default: return '🖥️';
    }
  };

  const getStatusColor = (s: string) => {
    switch (s) {
      case 'online': return 'bg-[var(--status-online)]';
      case 'degraded': return 'bg-[var(--status-degraded)]';
      case 'offline': return 'bg-[var(--status-offline)]';
      default: return 'bg-zinc-500';
    }
  };

  return (
    <section class="space-y-4 font-mono">
      <div class="flex justify-between items-end border-b border-[var(--border-bold)] pb-2">
        <h3 class="text-xs font-black uppercase tracking-[0.2em] text-[var(--accent-primary)]">
          Fleet Infrastructure Overview
        </h3>
        <div class="flex gap-4 text-[9px] uppercase tracking-widest text-[var(--text-muted)]">
          <div class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-[var(--status-online)]"></span> 03 Online</div>
          <div class="flex items-center gap-1"><span class="w-1.5 h-1.5 rounded-full bg-[var(--status-degraded)]"></span> 01 Warning</div>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        <For each={nodes()}>
          {(node) => (
            <div class="border border-[var(--border-subtle)] bg-black/10 p-3 flex justify-between items-center hover:border-[var(--accent-primary)] transition-all cursor-pointer group">
              <div class="flex items-center gap-3">
                <div class={`w-1 h-8 ${getStatusColor(node.status)}`}></div>
                <div class="flex flex-col">
                  <div class="flex items-center gap-1.5">
                    <span class="text-[10px]">{getPlatformIcon(node.platform)}</span>
                    <span class="text-xs font-black uppercase tracking-tight text-zinc-200">{node.name}</span>
                  </div>
                  <span class="text-[9px] text-[var(--text-muted)] opacity-60 uppercase">{node.tenant} // {node.ip}</span>
                </div>
              </div>
              <div class="opacity-0 group-hover:opacity-100 transition-opacity">
                <span class="text-[10px] text-[var(--accent-primary)]">DRILL_DOWN →</span>
              </div>
            </div>
          )}
        </For>
      </div>
    </section>
  );
}
