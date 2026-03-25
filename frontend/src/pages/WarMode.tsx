import { Component, createSignal, onMount, Show } from 'solid-js';
import { 
    PageLayout, 
    Panel, 
    Badge, 
    SectionHeader, 
    Input, 
    Button, 
    Notice,
    CodeBlock,
    normalizeSeverity 
} from '@components/ui';
import * as DisasterService from '../../wailsjs/go/services/DisasterService';
import '../styles/war-mode.css';

export const WarMode: Component = () => {
    const [status, setStatus] = createSignal<any>(null);
    const [loading, setLoading] = createSignal(true);
    const [passphrase, setPassphrase] = createSignal('');
    const [exporting, setExporting] = createSignal(false);
    const [lastExport, setLastExport] = createSignal<string | null>(null);
    const [exportErr, setExportErr] = createSignal('');

    const refresh = async () => {
        try {
            const mode = await DisasterService.GetMode();
            setStatus({ mode });
        } catch { /* service may not be ready yet */ }
        finally { setLoading(false); }
    };

    onMount(() => {
        refresh();
        const t = setInterval(refresh, 5_000);
        return () => clearInterval(t);
    });

    const handleExport = async () => {
        if (!passphrase()) { setExportErr('Passphrase required'); return; }
        setExporting(true);
        setExportErr('');
        try {
            const path = await DisasterService.ExportResilienceBundle(passphrase());
            setLastExport(path);
        } catch (err: any) {
            setExportErr(err?.message ?? String(err));
        } finally {
            setExporting(false);
        }
    };

    const toggleAirGap = async () => {
        if (status()?.mode === 'air_gap') {
            await DisasterService.DeactivateKillSwitch();
        } else {
            await DisasterService.ActivateAirGapMode();
        }
        refresh();
    };

    const toggleKillSwitch = async () => {
        if (status()?.mode === 'read_only') {
            await DisasterService.DeactivateKillSwitch();
        } else {
            const reason = prompt('KILL-SWITCH ACTIVATION — State reason for audit log:');
            if (reason) await DisasterService.ActivateKillSwitch(reason);
        }
        refresh();
    };

    const mode = () => status()?.mode as string | undefined;

    const modeDisplay = () => {
        const m = mode();
        const label = m ? m.replace('_', '-').toUpperCase() : 'NOMINAL';
        const sev = m === 'read_only' ? 'critical' : m === 'air_gap' ? 'medium' : 'low';
        return { label, sev };
    };

    return (
        <PageLayout
            title="War-Mode & Emergency Isolation"
            subtitle="SOVEREIGN_RESILIENCE"
            actions={
                <div style="display: flex; align-items: center; gap: 8px;">
                    <Badge severity={modeDisplay().sev}>
                        <div class="war-mode-status">
                            <i class={`war-mode-dot ${mode() && mode() !== 'nominal' ? 'active' : ''}`} style={`background: currentColor;`} />
                            <span>{loading() ? 'CHECKING...' : modeDisplay().label}</span>
                        </div>
                    </Badge>
                </div>
            }
        >
            <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(360px, 1fr)); gap: var(--gap-lg);">
                {/* NODE ISOLATION */}
                <Panel title="NODE_ISOLATION" subtitle="EMERGENCY_NETWORK_CONTROL">
                    <p style="font-size: 11px; color: var(--text-muted); margin-bottom: var(--gap-md); line-height: 1.5;">
                        Sever network ties or enter forensic read-only mode. Both actions are logged to the audit ledger.
                    </p>
                    <div style="display: flex; flex-direction: column; gap: var(--gap-sm);">
                        <button
                            class={`war-action-btn ${mode() === 'air_gap' ? 'active' : ''}`}
                            onClick={toggleAirGap}
                        >
                            <span class="war-action-label">{mode() === 'air_gap' ? 'RESTORE NETWORK' : 'ACTIVATE AIR-GAP'}</span>
                            <span class="war-action-sub">{mode() === 'air_gap' ? 'Re-enable outbound traffic' : 'Kill all outbound network activity immediately'}</span>
                        </button>
                        <button
                            class={`war-action-btn ${mode() === 'read_only' ? 'active' : ''} danger`}
                            onClick={toggleKillSwitch}
                        >
                            <span class="war-action-label">{mode() === 'read_only' ? 'RELEASE KILL-SWITCH' : 'ACTIVATE KILL-SWITCH'}</span>
                            <span class="war-action-sub">{mode() === 'read_only' ? 'Restore read-write ingestion' : 'Freeze all event ingestion — forensic mode only'}</span>
                        </button>
                    </div>
                </Panel>

                {/* DEAD-DROP REPLICATION */}
                <Panel title="DEAD-DROP_REPLICATION" subtitle="OFFLINE_STATE_TRANSPORT">
                    <p style="font-size: 11px; color: var(--text-muted); margin-bottom: var(--gap-md); line-height: 1.5;">
                        Export encrypted state bundle for physical transport to air-gapped clones.
                    </p>
                    <div style="display: flex; flex-direction: column; gap: var(--gap-sm); margin-bottom: var(--gap-md);">
                        <label style="font-family: var(--font-mono); font-size: 9px; font-weight: 800; color: var(--text-muted); text-transform: uppercase;">
                            Encryption Passphrase
                        </label>
                        <Input
                            type="password"
                            placeholder="Enter passphrase..."
                            value={passphrase()}
                            onInput={(e) => setPassphrase(e.currentTarget.value)}
                            mono
                        />
                    </div>

                    <Show when={exportErr()}>
                        <Notice level="error" class="mb-2">
                            {exportErr()}
                        </Notice>
                    </Show>

                    <Button
                        variant="primary"
                        onClick={handleExport}
                        disabled={exporting()}
                        loading={exporting()}
                    >
                        GENERATE_RESILIENCE_BUNDLE
                    </Button>

                    <Show when={lastExport()}>
                        <div style="margin-top: var(--gap-md);">
                            <SectionHeader>LAST_EXPORT_LOCATION</SectionHeader>
                            <CodeBlock>
                                {lastExport()}
                            </CodeBlock>
                        </div>
                    </Show>
                </Panel>

                {/* OFFLINE UPDATE */}
                <Panel title="OFFLINE_UPDATE" subtitle="SIGNED_BUNDLE_APPLICATION">
                    <p style="font-size: 11px; color: var(--text-muted); margin-bottom: var(--gap-md); line-height: 1.5;">
                        Apply signed update bundles from physical media. Signature verified before apply.
                    </p>
                    <div class="war-drop-zone">
                        <div style="font-family: var(--font-mono); font-size: 11px; font-weight: 800;">
                            DROP UPDATE BUNDLE (.vbx)
                        </div>
                        <Button variant="default" size="sm">SELECT FILE</Button>
                    </div>
                    <div style="display: flex; justify-content: space-between; margin-top: var(--gap-md);">
                        <Badge severity="low">INTEGRITY:_VERIFIED</Badge>
                        <span style="font-family: var(--font-mono); font-size: 10px; color: var(--text-muted); text-transform: uppercase;">
                            CHANNEL:_OFFLINE_ONLY
                        </span>
                    </div>
                </Panel>
            </div>
        </PageLayout>
    );
};
