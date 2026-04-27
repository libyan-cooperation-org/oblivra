<!--
  MetricsPanel — load average, memory, disk for the selected host.
  Pulls everything in one round-trip via a small inline shell script
  to avoid 4× the SSH overhead.
-->
<script lang="ts">
  import { Activity } from 'lucide-svelte';
  import RemoteExecPanel from './RemoteExecPanel.svelte';

  // Sentinel-delimited so the parser can split each section deterministically.
  const CMD = `printf '###LOAD###\\n' && cat /proc/loadavg 2>/dev/null && \\
printf '###UPTIME###\\n' && uptime 2>/dev/null && \\
printf '###MEM###\\n' && free -m 2>/dev/null && \\
printf '###DISK###\\n' && df -h --output=source,size,used,avail,pcent,target 2>/dev/null | head -20`;

  type Section = { load: string; uptime: string; mem: string; disk: string };

  function parse(out: string): Section {
    const sec: Section = { load: '', uptime: '', mem: '', disk: '' };
    let curr: keyof Section | null = null;
    const buf: Record<string, string[]> = { load: [], uptime: [], mem: [], disk: [] };
    for (const line of out.split('\n')) {
      const t = line.trim();
      if (t === '###LOAD###') { curr = 'load'; continue; }
      if (t === '###UPTIME###') { curr = 'uptime'; continue; }
      if (t === '###MEM###') { curr = 'mem'; continue; }
      if (t === '###DISK###') { curr = 'disk'; continue; }
      if (curr) buf[curr].push(line);
    }
    sec.load = buf.load.join('\n').trim();
    sec.uptime = buf.uptime.join('\n').trim();
    sec.mem = buf.mem.join('\n').trim();
    sec.disk = buf.disk.join('\n').trim();
    return sec;
  }

  function loadColor(loadAvg: string): string {
    const first = parseFloat(loadAvg.split(/\s+/)[0] ?? '0');
    if (first > 5) return 'text-rose-300';
    if (first > 2) return 'text-amber-300';
    return 'text-emerald-300';
  }
</script>

<RemoteExecPanel title="Metrics" icon={Activity} command={CMD} pollIntervalMs={5000}>
  {#snippet children({ output })}
    {@const s = parse(output)}
    <div class="grid grid-cols-1 gap-3 p-3 lg:grid-cols-2">
      <section class="rounded-md border border-[var(--b1)] bg-[var(--s1)] p-3">
        <div class="text-[10px] uppercase tracking-wider text-[var(--tx3)]">Load average</div>
        <div class="mt-1 font-mono text-base {loadColor(s.load)}">{s.load || '—'}</div>
        {#if s.uptime}
          <div class="mt-2 text-[10px] text-[var(--tx3)]">{s.uptime}</div>
        {/if}
      </section>

      <section class="rounded-md border border-[var(--b1)] bg-[var(--s1)] p-3">
        <div class="text-[10px] uppercase tracking-wider text-[var(--tx3)]">Memory</div>
        <pre class="mt-1 whitespace-pre overflow-x-auto font-mono text-[10px] text-[var(--tx2)]">{s.mem || '—'}</pre>
      </section>

      <section class="rounded-md border border-[var(--b1)] bg-[var(--s1)] p-3 lg:col-span-2">
        <div class="text-[10px] uppercase tracking-wider text-[var(--tx3)]">Disk</div>
        <pre class="mt-1 whitespace-pre overflow-x-auto font-mono text-[10px] text-[var(--tx2)]">{s.disk || '—'}</pre>
      </section>
    </div>
  {/snippet}
</RemoteExecPanel>
