<!--
  ContainersPanel — list Docker / podman containers on the selected host.
  We try docker first, then podman as a fallback. Tab-separated format
  parses cleanly without needing real JSON support on the remote.
-->
<script lang="ts">
  import { Boxes } from 'lucide-svelte';
  import RemoteExecPanel from './RemoteExecPanel.svelte';

  // Try docker, then podman. \\t between fields, \\n between rows.
  const FMT = "{{.ID}}\\t{{.Names}}\\t{{.Image}}\\t{{.Status}}\\t{{.Ports}}";
  let includeStopped = $state(false);
  let cmd = $derived(
    `(command -v docker >/dev/null && docker ps ${includeStopped ? '-a ' : ''}--format '${FMT}') || ` +
    `(command -v podman >/dev/null && podman ps ${includeStopped ? '-a ' : ''}--format '${FMT}') || ` +
    `echo "no docker / podman available"`,
  );

  type Row = { id: string; name: string; image: string; status: string; ports: string };

  function parse(out: string): Row[] {
    return out
      .split('\n')
      .map((line) => line.trim())
      .filter(Boolean)
      .filter((line) => !line.startsWith('no docker'))
      .map<Row>((line) => {
        const [id = '', name = '', image = '', status = '', ports = ''] = line.split('\t');
        return { id, name, image, status, ports };
      });
  }
</script>

<RemoteExecPanel title="Containers" icon={Boxes} command={cmd} pollIntervalMs={10_000}>
  {#snippet controls({ refresh: _refresh })}
    <label class="flex items-center gap-1 text-[11px] text-[var(--tx2)]">
      <input type="checkbox" bind:checked={includeStopped} class="accent-cyan-400" />
      include stopped
    </label>
  {/snippet}
  {#snippet children({ output })}
    {@const rows = parse(output)}
    {#if rows.length === 0}
      <div class="px-6 py-12 text-center text-sm text-[var(--tx3)]">
        {output.includes('no docker') ? 'No docker / podman on this host.' : 'No containers running.'}
      </div>
    {:else}
      <table class="w-full text-xs">
        <thead class="sticky top-0 bg-[var(--s1)] text-[10px] uppercase tracking-wider text-[var(--tx3)]">
          <tr>
            <th class="px-2 py-2 text-left">ID</th>
            <th class="px-2 py-2 text-left">Name</th>
            <th class="px-2 py-2 text-left">Image</th>
            <th class="px-2 py-2 text-left">Status</th>
            <th class="px-2 py-2 text-left">Ports</th>
          </tr>
        </thead>
        <tbody>
          {#each rows as r (r.id)}
            <tr class="border-t border-[var(--b1)] hover:bg-[var(--s1)]">
              <td class="px-2 py-1 font-mono text-[10px] text-[var(--tx2)]">{r.id.slice(0, 12)}</td>
              <td class="px-2 py-1">{r.name}</td>
              <td class="px-2 py-1 font-mono text-[10px] text-[var(--tx2)]">{r.image}</td>
              <td class="px-2 py-1">
                <span class="rounded-sm border px-1.5 py-0.5 text-[9px] uppercase tracking-wider {r.status.toLowerCase().startsWith('up')
                  ? 'border-emerald-400/30 bg-emerald-400/10 text-emerald-300'
                  : 'border-[var(--b1)] bg-[var(--s2)] text-[var(--tx3)]'}">{r.status}</span>
              </td>
              <td class="px-2 py-1 font-mono text-[10px] text-[var(--tx2)]">{r.ports}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  {/snippet}
</RemoteExecPanel>
