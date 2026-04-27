<!--
  ProcessesPanel — top-style snapshot of remote processes.
  Runs `ps -eo pid,user,pcpu,pmem,rss,comm,args --sort=-pcpu | head -100`.
  Polls every 5s by default.
-->
<script lang="ts">
  import { Cpu } from 'lucide-svelte';
  import RemoteExecPanel from './RemoteExecPanel.svelte';

  // Wide column set covering the most-used `top`-equivalents.
  const CMD = "ps -eo pid,user,pcpu,pmem,rss,comm,args --sort=-pcpu --no-headers | head -100";

  type Row = { pid: string; user: string; cpu: string; mem: string; rss: string; comm: string; args: string };

  function parse(out: string): Row[] {
    return out
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
      .map<Row>((line) => {
        const cols = line.split(/\s+/);
        const pid = cols[0] ?? '';
        const user = cols[1] ?? '';
        const cpu = cols[2] ?? '';
        const mem = cols[3] ?? '';
        const rss = cols[4] ?? '';
        const comm = cols[5] ?? '';
        const args = cols.slice(6).join(' ');
        return { pid, user, cpu, mem, rss, comm, args };
      });
  }
</script>

<RemoteExecPanel title="Processes" icon={Cpu} command={CMD} pollIntervalMs={5000}>
  {#snippet children({ output })}
    {@const rows = parse(output)}
    {#if rows.length === 0}
      <div class="px-6 py-12 text-center text-sm text-[var(--tx3)]">No processes returned.</div>
    {:else}
      <table class="w-full text-xs">
        <thead class="sticky top-0 bg-[var(--s1)] text-[10px] uppercase tracking-wider text-[var(--tx3)]">
          <tr>
            <th class="px-2 py-2 text-right">PID</th>
            <th class="px-2 py-2 text-left">USER</th>
            <th class="px-2 py-2 text-right">%CPU</th>
            <th class="px-2 py-2 text-right">%MEM</th>
            <th class="px-2 py-2 text-right">RSS</th>
            <th class="px-2 py-2 text-left">COMMAND</th>
          </tr>
        </thead>
        <tbody>
          {#each rows as r (r.pid)}
            <tr class="border-t border-[var(--b1)] hover:bg-[var(--s1)]">
              <td class="px-2 py-1 text-right font-mono text-[10px]">{r.pid}</td>
              <td class="px-2 py-1 font-mono text-[10px] text-[var(--tx2)]">{r.user}</td>
              <td class="px-2 py-1 text-right font-mono text-[10px] {parseFloat(r.cpu) > 50 ? 'text-rose-300' : parseFloat(r.cpu) > 10 ? 'text-amber-300' : 'text-[var(--tx2)]'}">{r.cpu}</td>
              <td class="px-2 py-1 text-right font-mono text-[10px] {parseFloat(r.mem) > 50 ? 'text-rose-300' : 'text-[var(--tx2)]'}">{r.mem}</td>
              <td class="px-2 py-1 text-right font-mono text-[10px] text-[var(--tx2)]">{r.rss}</td>
              <td class="px-2 py-1 font-mono text-[11px] text-[var(--tx)]">{r.args || r.comm}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  {/snippet}
</RemoteExecPanel>
