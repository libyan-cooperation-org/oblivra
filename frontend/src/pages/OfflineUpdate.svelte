<!-- Offline Update — bound to UpdaterService. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, PopOutButton } from '@components/ui';
  import { Download, Upload, RefreshCw, CheckCircle2 } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let info = $state<any>(null);
  let busy = $state<string | null>(null);

  async function checkUpdate() {
    busy = 'check';
    try {
      if (IS_BROWSER) return;
      const { CheckForUpdate } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/updaterservice');
      info = await CheckForUpdate();
      appStore.notify((info as any)?.has_update ? `Update ${(info as any).version} available` : 'Up to date', (info as any)?.has_update ? 'info' : 'success');
    } catch (e: any) { appStore.notify(`Check failed: ${e?.message ?? e}`, 'error'); }
    finally { busy = null; }
  }
  async function applyUpdate() {
    if (!confirm('Apply downloaded update? Service will restart.')) return;
    busy = 'apply';
    try {
      const { ApplyUpdate } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/updaterservice');
      await ApplyUpdate();
      appStore.notify('Update applied', 'success');
    } catch (e: any) { appStore.notify(`Apply failed: ${e?.message ?? e}`, 'error'); }
    finally { busy = null; }
  }
  async function exportBundle() {
    const dir = prompt('Output directory for offline bundle:', './oblivra-offline'); if (!dir) return;
    busy = 'export';
    try {
      const { CreateOfflineBundle } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/updaterservice');
      await CreateOfflineBundle(dir);
      appStore.notify(`Bundle written to ${dir}`, 'success');
    } catch (e: any) { appStore.notify(`Export failed: ${e?.message ?? e}`, 'error'); }
    finally { busy = null; }
  }
  async function importBundle() {
    const path = prompt('Path to offline-update bundle:'); if (!path) return;
    busy = 'import';
    try {
      const { ImportOfflineBundle } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/updaterservice');
      await ImportOfflineBundle(path);
      appStore.notify('Bundle imported', 'success');
    } catch (e: any) { appStore.notify(`Import failed: ${e?.message ?? e}`, 'error'); }
    finally { busy = null; }
  }
  onMount(checkUpdate);
</script>

<PageLayout title="Offline Update" subtitle="Air-gapped platform updates and signature verification">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={checkUpdate} disabled={busy === 'check'}>{busy === 'check' ? 'Checking…' : 'Check'}</Button>
    <PopOutButton route="/offline-update" title="Offline Update" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Current" value={(info?.current_version ?? '—').toString()} variant="muted" />
      <KPI label="Latest" value={(info?.latest_version ?? info?.version ?? '—').toString()} variant={info?.has_update ? 'warning' : 'muted'} />
      <KPI label="Status" value={info?.has_update ? 'Update available' : info ? 'Up to date' : '—'} variant={info?.has_update ? 'warning' : 'success'} />
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-2">
        <div class="flex items-center gap-2"><Download size={14} class="text-accent" /><span class="text-xs font-bold uppercase">Apply Update</span></div>
        <p class="text-[11px] text-text-muted">Apply the latest verified update bundle. Service will restart.</p>
        <Button variant="cta" size="sm" onclick={applyUpdate} disabled={!info?.has_update || busy === 'apply'}>{busy === 'apply' ? 'Applying…' : 'Apply Update'}</Button>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-2">
        <div class="flex items-center gap-2"><Upload size={14} class="text-accent" /><span class="text-xs font-bold uppercase">Offline Bundle</span></div>
        <p class="text-[11px] text-text-muted">Build a bundle for air-gapped deployment, or import one received offline.</p>
        <div class="flex gap-2">
          <Button variant="secondary" size="sm" onclick={exportBundle} disabled={busy === 'export'}>{busy === 'export' ? '…' : 'Build bundle'}</Button>
          <Button variant="secondary" size="sm" onclick={importBundle} disabled={busy === 'import'}>{busy === 'import' ? '…' : 'Import bundle'}</Button>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
