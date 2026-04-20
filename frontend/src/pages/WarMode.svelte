<!--
  OBLIVRA — War Room Mode (Svelte 5)
  The tactical red-alert interface for high-intensity incident response.
-->
<script lang="ts">
  import { PageLayout, Button, Badge } from '@components/ui';
  import { appStore } from '@lib/stores/app.svelte';
  import { collabStore } from '@lib/stores/collaboration.svelte.ts';
  import { Shield, AlertCircle, Radio, Lock, Power, MessageSquare, Send } from 'lucide-svelte';

  let countdown = $state(3600); // 1 hour containment window
  let containmentActive = $state(false);
  let messageText = $state('');

  $effect(() => {
    let timer: any;
    if (containmentActive && countdown > 0) {
      timer = setInterval(() => {
        countdown--;
      }, 1000);
    }
    return () => clearInterval(timer);
  });

  const formattedTime = $derived(() => {
    const mins = Math.floor(countdown / 60);
    const secs = countdown % 60;
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
  });

  function toggleContainment() {
    containmentActive = !containmentActive;
    if (containmentActive) {
        appStore.notify('GLOBAL CONTAINMENT ARMED', 'error', 'All egress traffic restricted to verified nodes.');
        collabStore.sendMessage('GLOBAL CONTAINMENT ARMED', 'action');
    } else {
        collabStore.sendMessage('GLOBAL CONTAINMENT DISARMED', 'action');
    }
  }

  function handleSend(e: Event) {
    e.preventDefault();
    if (!messageText.trim()) return;
    collabStore.sendMessage(messageText);
    messageText = '';
  }
</script>

<PageLayout title="War Room Mode" subtitle="Tactical command interface for active threat containment and protocol execution">
  <div class="flex flex-col h-full -m-6 bg-error/5 animate-pulse-slow">
    <!-- CRITICAL ALERT STRIP -->
    <div class="bg-error px-6 py-2 flex items-center justify-between shadow-[0_0_30px_rgba(200,44,44,0.3)] shrink-0">
        <div class="flex items-center gap-3">
            <Shield size={20} class="text-white animate-bounce" />
            <span class="text-sm font-bold text-white uppercase tracking-widest">Global Protocol Level: Red</span>
        </div>
        <div class="flex items-center gap-4">
            <div class="flex -space-x-2 mr-4">
                {#each collabStore.analysts as analyst}
                    <div 
                        class="w-6 h-6 rounded-full border-2 border-error flex items-center justify-center text-[8px] font-black uppercase text-white shadow-lg"
                        style="background: {analyst.color};"
                        title="{analyst.name} ({analyst.role})"
                    >
                        {analyst.name[0]}
                        {#if analyst.status === 'active'}
                            <div class="absolute bottom-0 right-0 w-1.5 h-1.5 bg-success rounded-full border border-error"></div>
                        {/if}
                    </div>
                {/each}
            </div>
            <div class="flex items-center gap-6 text-white/80 font-mono text-[10px]">
                <span>THREAT_ACTOR: APT-28</span>
                <span>BLAST_RADIUS: 14 NODES</span>
                <span class="bg-white/20 px-2 py-0.5 rounded-sm">PROTOCOL: OBLIVRA_CONTAIN_V4</span>
            </div>
        </div>
    </div>

    <!-- MAIN GRID -->
    <div class="flex-1 grid grid-cols-12 gap-px bg-error/20 overflow-hidden">
        <!-- LEFT: TACTICAL TELEMETRY -->
        <div class="col-span-8 flex flex-col bg-surface-1 min-h-0">
            <div class="p-6 flex flex-col items-center justify-center gap-8 flex-1">
                <!-- RADAR / VISUALIZER PLACEHOLDER -->
                <div class="w-96 h-96 rounded-full border-4 border-error/20 flex items-center justify-center relative">
                    <div class="absolute inset-0 rounded-full border border-error/40 animate-ping"></div>
                    <div class="absolute inset-8 rounded-full border border-error/30 animate-pulse"></div>
                    <div class="w-full h-1 bg-error/20 absolute rotate-45"></div>
                    <div class="w-full h-1 bg-error/20 absolute -rotate-45"></div>
                    <div class="z-10 flex flex-col items-center text-center">
                        <AlertCircle size={48} class="text-error mb-4" />
                        <div class="text-4xl font-mono font-bold text-text-heading tracking-tighter">
                            {formattedTime()}
                        </div>
                        <div class="text-[10px] font-mono text-error uppercase tracking-[0.2em] font-bold mt-2">Containment Window</div>
                    </div>
                </div>

                <div class="grid grid-cols-3 gap-8 w-full max-w-2xl">
                    <div class="bg-surface-2 border border-error/20 p-4 rounded-sm text-center">
                        <div class="text-[9px] font-mono text-text-muted uppercase mb-1">Egress Blocked</div>
                        <div class="text-2xl font-mono font-bold text-error">94.2 GB</div>
                    </div>
                    <div class="bg-surface-2 border border-error/20 p-4 rounded-sm text-center">
                        <div class="text-[9px] font-mono text-text-muted uppercase mb-1">Lateral Moves</div>
                        <div class="text-2xl font-mono font-bold text-warning">0</div>
                    </div>
                    <div class="bg-surface-2 border border-error/20 p-4 rounded-sm text-center">
                        <div class="text-[9px] font-mono text-text-muted uppercase mb-1">Mesh Integrity</div>
                        <div class="text-2xl font-mono font-bold text-success">VERIFIED</div>
                    </div>
                </div>
            </div>

            <!-- LOGS -->
            <div class="h-48 border-t border-error/20 bg-black/40 p-4 font-mono text-[10px] space-y-1 overflow-auto">
                <div class="text-error font-bold mb-2">[CRITICAL AUDIT TRAIL]</div>
                <div class="flex gap-2 text-text-muted"><span class="opacity-40">10:42:15</span> <span class="text-error font-bold">[WARN]</span> Port scanning detected from unknown peer</div>
                <div class="flex gap-2 text-text-muted"><span class="opacity-40">10:42:18</span> <span class="text-error font-bold">[HALT]</span> Execution blocked on SRV-APP-04 (Suspicious Payload)</div>
                <div class="flex gap-2 text-text-muted"><span class="opacity-40">10:42:20</span> <span class="text-success font-bold">[INFO]</span> Ephemeral keys rotated globally</div>
                <div class="flex gap-2 text-text-muted"><span class="opacity-40">10:42:21</span> <span class="text-error font-bold">[WARN]</span> Inbound SSH attempt from 203.0.113.5 blocked</div>
            </div>
        </div>

        <!-- RIGHT: MISSION CONTROL -->
        <div class="col-span-4 bg-surface-2 flex flex-col border-l border-error/20 overflow-hidden">
            <!-- COLLABORATION FEED -->
            <div class="flex-1 flex flex-col min-h-0 bg-black/20">
                <div class="p-3 border-b border-error/20 flex items-center justify-between bg-surface-2">
                    <div class="flex items-center gap-2">
                        <MessageSquare size={14} class="text-error" />
                        <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Tactical Comms</span>
                    </div>
                    <Badge variant="critical" size="xs" class="animate-pulse">{collabStore.analysts.filter(a => a.status === 'active').length} OPS</Badge>
                </div>
                
                <div class="flex-1 overflow-auto p-3 space-y-3 font-mono text-[10px]">
                    {#each collabStore.messages as msg}
                        {@const analyst = collabStore.analysts.find(a => a.id === msg.analystId)}
                        <div class="flex flex-col gap-1">
                            <div class="flex items-center gap-2">
                                <span class="font-bold" style="color: {analyst?.color || 'var(--text-muted)'}">{analyst?.name || 'Unknown'}</span>
                                <span class="opacity-30 text-[8px]">{new Date(msg.timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit', second: '2-digit'})}</span>
                            </div>
                            <div class="p-2 rounded-sm {msg.type === 'action' ? 'bg-error/10 border border-error/20 text-error font-bold italic' : 'bg-surface-3 text-text-secondary'}">
                                {msg.text}
                            </div>
                        </div>
                    {/each}
                </div>

                <!-- CHAT INPUT -->
                <form class="p-2 bg-surface-2 border-t border-error/20 flex gap-2" onsubmit={handleSend}>
                    <input 
                        type="text" 
                        bind:value={messageText}
                        placeholder="SEND TACTICAL MSG..." 
                        class="flex-1 bg-black/40 border border-border-primary rounded-sm px-2 py-1.5 text-[10px] font-mono text-text-heading focus:border-error focus:outline-none transition-colors"
                    />
                    <button type="submit" class="p-1.5 bg-error/20 border border-error/40 text-error hover:bg-error/30 transition-colors rounded-sm">
                        <Send size={14} />
                    </button>
                </form>
            </div>

            <div class="p-4 border-t border-error/20 bg-surface-1">
                <div class="flex items-center gap-2 mb-4">
                    <Radio size={14} class="text-error animate-pulse" />
                    <span class="text-[10px] font-mono font-bold uppercase tracking-widest">Active Protocols</span>
                </div>
                <div class="space-y-2">
                    <div class="flex items-center justify-between p-2 bg-surface-2 border border-border-primary rounded-sm group hover:border-error transition-colors">
                        <span class="text-[9px] font-mono font-bold uppercase">Network Isolation</span>
                        <Badge variant="critical">ACTIVE</Badge>
                    </div>
                    <div class="flex items-center justify-between p-2 bg-surface-2 border border-border-primary rounded-sm group hover:border-error transition-colors">
                        <span class="text-[9px] font-mono font-bold uppercase">Data Sharding</span>
                        <Badge variant="success">READY</Badge>
                    </div>
                </div>
            </div>

            <div class="p-4 flex flex-col gap-3 bg-surface-2">
                <Button variant="danger" class="w-full h-10 uppercase font-black tracking-tighter text-sm" onclick={toggleContainment}>
                    {containmentActive ? 'DISARM CONTAINMENT' : 'ARM GLOBAL CONTAINMENT'}
                </Button>
                <div class="grid grid-cols-2 gap-2">
                    <Button variant="secondary" size="sm" class="text-[9px] font-bold">
                        <Lock size={12} class="mr-2" /> LOCK VAULTS
                    </Button>
                    <Button variant="secondary" size="sm" class="text-[9px] font-bold">
                        <Power size={12} class="mr-2" /> REBOOT MESH
                    </Button>
                </div>
            </div>
        </div>
    </div>
  </div>
</PageLayout>

<style>
  @keyframes pulse-slow {
    0%, 100% { background-color: rgba(200, 44, 44, 0.05); }
    50% { background-color: rgba(200, 44, 44, 0.08); }
  }
  .animate-pulse-slow {
    animation: pulse-slow 4s cubic-bezier(0.4, 0, 0.6, 1) infinite;
  }
</style>
