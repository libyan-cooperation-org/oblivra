<!--
  OBLIVRA — War Room Mode (Svelte 5)
  The tactical red-alert interface for high-intensity incident response.
-->
<script lang="ts">
  import { PageLayout, Button, Badge } from '@components/ui';
  import { appStore } from '@lib/stores/app.svelte';
  import { Shield, AlertCircle, Radio, Zap, Lock, Power } from 'lucide-svelte';

  let countdown = $state(3600); // 1 hour containment window
  let containmentActive = $state(false);

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
    }
  }
</script>

<PageLayout title="War Room Mode" subtitle="Tactical command interface for active threat containment and protocol execution">
  <div class="flex flex-col h-full -m-6 bg-error/5 animate-pulse-slow">
    <!-- CRITICAL ALERT STRIP -->
    <div class="bg-error px-6 py-2 flex items-center justify-between shadow-[0_0_30px_rgba(200,44,44,0.3)]">
        <div class="flex items-center gap-3">
            <Shield size={20} class="text-white animate-bounce" />
            <span class="text-sm font-bold text-white uppercase tracking-widest">Global Protocol Level: Red</span>
        </div>
        <div class="flex items-center gap-6 text-white/80 font-mono text-[10px]">
            <span>THREAT_ACTOR: APT-28 (ESTIMATED)</span>
            <span>BLAST_RADIUS: 14 NODES</span>
            <span>PROTOCOL: OBLIVRA_CONTAIN_V4</span>
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
        <div class="col-span-4 bg-surface-2 flex flex-col border-l border-error/20">
            <div class="p-4 border-b border-error/20">
                <div class="flex items-center gap-2 mb-4">
                    <Radio size={14} class="text-error animate-pulse" />
                    <span class="text-[10px] font-mono font-bold uppercase tracking-widest">Active Protocols</span>
                </div>
                <div class="space-y-2">
                    <div class="flex items-center justify-between p-2 bg-surface-1 border border-border-primary rounded-sm group hover:border-error transition-colors">
                        <span class="text-[9px] font-mono font-bold uppercase">Network Isolation</span>
                        <Badge variant="critical">ACTIVE</Badge>
                    </div>
                    <div class="flex items-center justify-between p-2 bg-surface-1 border border-border-primary rounded-sm group hover:border-error transition-colors">
                        <span class="text-[9px] font-mono font-bold uppercase">Data Encryption Sharding</span>
                        <Badge variant="success">READY</Badge>
                    </div>
                    <div class="flex items-center justify-between p-2 bg-surface-1 border border-border-primary rounded-sm group hover:border-error transition-colors">
                        <span class="text-[9px] font-mono font-bold uppercase">Hardware Key Rotation</span>
                        <Badge variant="warning">PENDING</Badge>
                    </div>
                </div>
            </div>

            <div class="p-6 flex flex-col gap-4 mt-auto">
                <div class="flex items-center gap-2 mb-2">
                    <Zap size={16} class="text-error" />
                    <span class="text-[10px] font-mono font-bold uppercase tracking-widest">Orchestration Actions</span>
                </div>
                <Button variant="danger" class="w-full h-12 uppercase font-black tracking-tighter text-md" onclick={toggleContainment}>
                    {containmentActive ? 'DISARM CONTAINMENT' : 'ARM GLOBAL CONTAINMENT'}
                </Button>
                <div class="grid grid-cols-2 gap-2">
                    <Button variant="secondary" size="sm" class="font-bold">
                        <Lock size={14} class="mr-2" /> LOCK VAULTS
                    </Button>
                    <Button variant="secondary" size="sm" class="font-bold">
                        <Power size={14} class="mr-2" /> REBOOT MESH
                    </Button>
                </div>
                <p class="text-[8px] font-mono text-text-muted italic text-center opacity-60">
                    Executing these actions will impact global fleet performance. Use with extreme caution.
                </p>
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
