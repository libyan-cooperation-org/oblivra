import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { 
    PageLayout, 
    Panel, 
    Badge, 
    LoadingState, 
    normalizeSeverity,
    formatRelativeTime
} from '@components/ui';
import { IS_BROWSER } from '@core/context';

interface TrustStatus {
    component: string;
    status: string;
    detail: string;
    last_check: string;
}

export const RuntimeTrust: Component = () => {
    const [score, setScore] = createSignal<number>(0);
    const [components, setComponents] = createSignal<TrustStatus[]>([]);
    const [loading, setLoading] = createSignal(true);

    const loadTrustState = async () => {
        setLoading(true);
        if (IS_BROWSER) {
            setScore(84.2);
            setComponents([
                { component: 'Secure Boot', status: 'TRUSTED', detail: 'UEFI Secure Boot enabled and verified', last_check: new Date().toISOString() },
                { component: 'Memory Protection', status: 'TRUSTED', detail: 'ASLR and DEP active on 1,248 processes', last_check: new Date().toISOString() },
                { component: 'Kernel Integrity', status: 'WARNING', detail: 'Unknown module loaded: kprobe_filter', last_check: new Date().toISOString() },
                { component: 'Audit Framework', status: 'TRUSTED', detail: 'Auditd logging at high verbosity', last_check: new Date().toISOString() }
            ]);
            setLoading(false);
            return;
        }

        try {
            const svc = (window as any).go?.services?.RuntimeTrustService;
            if (!svc) return;

            await svc.VerifyIntegrity();
            const [newScore, newComponents] = await Promise.all([
                svc.CalculateTrustIndex(),
                svc.GetAggregatedStatus()
            ]);

            setScore(newScore);
            setComponents(newComponents || []);
        } catch (err) {
            console.error("Failed to load trust index:", err);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        loadTrustState();
    });

    const getScoreColor = (sc: number) => {
        if (sc >= 90) return 'var(--status-online)';
        if (sc >= 70) return 'var(--status-degraded)';
        return 'var(--alert-critical)';
    };

    return (
        <PageLayout
            title="Adaptive Trust Weighting"
            subtitle="GLOBAL_MACHINE_INTEGRITY_CONSENSUS"
            actions={
                <button 
                    class="ob-btn ob-btn-primary" 
                    onClick={loadTrustState}
                    disabled={loading()}
                >
                    {loading() ? 'CALCULATING...' : 'RE-VERIFY INVARIANTS'}
                </button>
            }
        >
            <div style="display: grid; grid-template-columns: 320px 1fr; gap: var(--gap-lg); flex: 1; min-height: 0;">
                {/* Score Widget */}
                <Panel title="GLOBAL TRUST SCORE" class="trust-score-panel">
                    <div style="display: flex; flex-direction: column; items: center; justify-content: center; padding: var(--gap-xl) 0; text-align: center;">
                        <div style="position: relative; width: 200px; height: 200px; margin: 0 auto;">
                            <svg style="width: 100%; height: 100%; transform: rotate(-90deg);">
                                <circle 
                                    cx="100" cy="100" r="85" 
                                    stroke="var(--surface-3)" stroke-width="8" fill="transparent" 
                                />
                                <circle
                                    cx="100" cy="100" r="85"
                                    stroke={getScoreColor(score())}
                                    stroke-width="8"
                                    fill="transparent"
                                    stroke-dasharray={String(85 * 2 * Math.PI)}
                                    stroke-dashoffset={String(85 * 2 * Math.PI * (1 - (score() / 100)))}
                                    style="transition: stroke-dashoffset 1s ease-out;"
                                />
                            </svg>
                            <div style="position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center;">
                                <div style={{ 
                                    'font-size': '44px', 
                                    'font-weight': '800', 
                                    'font-family': 'var(--font-mono)',
                                    color: getScoreColor(score())
                                }}>
                                    {score().toFixed(1)}
                                </div>
                                <div style="font-size: 11px; color: var(--text-muted); font-weight: 600;">INDEX_VAL</div>
                            </div>
                        </div>

                        <div style="margin-top: var(--gap-lg); font-size: 11px; color: var(--text-muted); line-height: 1.5; padding: 0 var(--gap-md);">
                            Machine runtime integrity calculated using a deterministic 5-factor weighting model.
                        </div>
                    </div>
                </Panel>

                {/* Pillars Grid */}
                <div style="display: flex; flex-direction: column; gap: var(--gap-md); overflow-y: auto;">
                    <Show when={loading()}>
                        <LoadingState message="AUDITING_SYSTEM_INVARIANTS..." />
                    </Show>
                    
                    <Show when={!loading()}>
                        <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: var(--gap-md);">
                            <For each={components().sort((a, b) => a.component.localeCompare(b.component))}>
                                {(comp) => (
                                    <Panel 
                                        title={comp.component} 
                                        actions={
                                            <Badge severity={normalizeSeverity(comp.status)}>
                                                {comp.status}
                                            </Badge>
                                        }
                                    >
                                        <div style="font-size: 12px; color: var(--text-primary); margin-bottom: var(--gap-md); line-height: 1.5;">
                                            {comp.detail}
                                        </div>
                                        <div style="display: flex; justify-content: space-between; align-items: center; border-top: 1px solid var(--border-subtle); padding-top: var(--gap-sm); margin-top: auto;">
                                            <span style="font-size: 10px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px;">Last Verified</span>
                                            <span style="font-size: 11px; color: var(--text-muted); font-family: var(--font-mono);">
                                                {formatRelativeTime(comp.last_check)}
                                            </span>
                                        </div>
                                    </Panel>
                                )}
                            </For>
                        </div>

                        <Show when={components().length === 0}>
                            <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 48px; color: var(--text-muted); border: 1px dashed var(--border-primary); border-radius: var(--radius-md);">
                                <div style="font-size: 14px; font-weight: 600; margin-bottom: 4px;">NO DATA</div>
                                <div style="font-size: 12px;">Waiting for first invariant pass...</div>
                            </div>
                        </Show>
                    </Show>
                </div>
            </div>
        </PageLayout>
    );
};
