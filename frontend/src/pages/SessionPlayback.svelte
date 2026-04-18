<!--
  OBLIVRA — Session Playback (Svelte 5)
  Forensic replay of recorded terminal telemetry.
-->
<script lang="ts">
  import { KPI, PageLayout, Button, Badge } from '@components/ui';
  import { Play, Pause, SkipBack, SkipForward, Clock, Shield } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  let playing = $state(false);
  let progress = $state(45);
  let speed = $state(1);

  const eventLog = [
    { time: '00:12', type: 'input', content: 'sudo su -' },
    { time: '00:15', type: 'output', content: '[sudo] password for operator:' },
    { time: '00:22', type: 'input', content: 'ls -la /root' },
    { time: '00:25', type: 'warning', content: 'UNAUTHORIZED ACCESS ATTEMPT DETECTED' },
  ];
</script>

<PageLayout title="Forensic Replay" subtitle="Auditing terminal session ID: TS-9921">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Badge variant="warning">HI-RES TELEMETRY</Badge>
      <Button variant="secondary" size="sm">Export ASCINEMA</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <!-- Playback Engine -->
    <div class="flex-1 bg-black border border-border-primary rounded-md relative flex flex-col overflow-hidden shadow-2xl">
      <div class="flex-1 p-6 font-mono text-[13px] text-success/90 leading-relaxed overflow-y-auto">
        <div class="opacity-40 mb-4"># OBLIVRA Forensic Playback Engine v2.0</div>
        <div class="mb-1">operator@prod-gateway:~$ sudo su -</div>
        <div class="mb-1">[sudo] password for operator: **********</div>
        <div class="mb-1">root@prod-gateway:~# ls -la /root</div>
        <div class="mb-4">total 42</div>
        <div class="text-error font-bold bg-error/10 p-2 border border-error/20 inline-block mb-4">
          [!] SECURITY ALERT: Access to /root/secrets.gpg monitored
        </div>
        <div class="animate-pulse">_</div>
      </div>

      <!-- Controls -->
      <div class="p-4 bg-surface-2 border-t border-border-primary">
        <div class="flex flex-col gap-4">
          <!-- Seek bar -->
          <div class="relative w-full h-1.5 bg-surface-3 rounded-full cursor-pointer">
            <div class="absolute h-full bg-accent rounded-full transition-all" style="width: {progress}%"></div>
            <div class="absolute w-3 h-3 bg-white border-2 border-accent rounded-full -top-0.5 shadow-md" style="left: {progress}%"></div>
          </div>

          <div class="flex items-center justify-between">
            <div class="flex items-center gap-4">
              <button class="text-text-muted hover:text-text-primary"><SkipBack size={18} /></button>
              <button 
                class="w-10 h-10 rounded-full bg-accent text-white flex items-center justify-center hover:bg-accent/80 transition-colors"
                onclick={() => playing = !playing}
              >
                {#if playing}<Pause size={20} />{:else}<Play size={20} />{/if}
              </button>
              <button class="text-text-muted hover:text-text-primary"><SkipForward size={18} /></button>
            </div>

            <div class="flex items-center gap-6">
              <div class="flex items-center gap-2 text-[11px] font-mono text-text-muted">
                <Clock size={12} />
                <span>02:45 / 05:00</span>
              </div>
              <div class="flex bg-surface-3 rounded-sm p-0.5">
                {#each [1, 2, 4] as s}
                  <button 
                    class="px-2 py-1 text-[9px] font-bold rounded-sm {speed === s ? 'bg-accent text-white' : 'text-text-muted hover:bg-surface-0'}"
                    onclick={() => speed = s}
                  >{s}x</button>
                {/each}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Side Analysis -->
    <div class="h-48 flex gap-5">
      <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-3">
        <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Event Timeline</div>
        <div class="flex-1 overflow-y-auto space-y-2">
          {#each eventLog as event}
            <div class="flex items-center gap-3 text-[10px]">
              <span class="text-text-muted font-mono">{event.time}</span>
              <Badge variant={event.type === 'warning' ? 'error' : 'info'} size="xs">{event.type}</Badge>
              <span class="text-text-secondary truncate">{event.content}</span>
            </div>
          {/each}
        </div>
      </div>

      <div class="w-72 bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-4">
        <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Metadata</div>
        <div class="space-y-3">
          <div class="flex flex-col">
            <span class="text-[9px] text-text-muted uppercase">Origin Host</span>
            <span class="text-xs font-bold text-text-heading">10.0.4.15 (prod-gateway)</span>
          </div>
          <div class="flex flex-col">
            <span class="text-[9px] text-text-muted uppercase">Operator Identity</span>
            <span class="text-xs font-bold text-accent">maverick (UID: 1000)</span>
          </div>
          <div class="flex flex-col">
            <span class="text-[9px] text-text-muted uppercase">Risk Profile</span>
            <Badge variant="error" class="w-fit mt-1">SENSITIVE ACCESS</Badge>
          </div>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
