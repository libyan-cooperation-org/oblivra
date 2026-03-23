import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { IS_BROWSER } from '@core/context';
import { detection } from '../../../wailsjs/go/models';

export const PolicyVerifier: Component = () => {
    const [verdicts, setVerdicts] = createSignal<detection.ValidationResult[]>([]);
    const [loading, setLoading] = createSignal(true);

    onMount(() => {
        loadVerdicts();
    });

    const loadVerdicts = async () => {
        if (IS_BROWSER) { setLoading(false); return; }
        try {
            const { GetRuleVerifications } = await import('../../../wailsjs/go/services/AlertingService');
            setVerdicts(await GetRuleVerifications() || []);
        } catch (error) { console.error('Failed to load rule verifications', error); }
        finally { setLoading(false); }
    };

    return (
        <div class="policy-verifier p-6 bg-hacker-black text-cyber-gray" style="flex: 1; overflow-y: auto;">
            <div class="flex justify-between items-center mb-6 border-b border-neon-blue pb-4">
                <div>
                    <h2 class="text-2xl font-bold text-neon-teal uppercase tracking-widest" style="margin: 0;">
                        Formal Verification Layer
                    </h2>
                    <p class="text-sm mt-1">Mathematical AST validation ensuring safe policy execution bounds.</p>
                </div>
                <div class="flex gap-4">
                    <div class="bg-terminal-bg border border-neon-teal px-4 py-2 text-center rounded">
                        <div class="text-2xl font-bold text-neon-teal">{verdicts().filter(v => v.is_valid).length}</div>
                        <div class="text-xs uppercase">Secured Rules</div>
                    </div>
                    <div class="bg-terminal-bg border border-alert-red px-4 py-2 text-center rounded">
                        <div class="text-2xl font-bold text-alert-red">{verdicts().filter(v => !v.is_valid).length}</div>
                        <div class="text-xs uppercase">Rejected Rules</div>
                    </div>
                </div>
            </div>

            <Show when={loading()}>
                <div class="p-4 text-neon-teal">Initializing Formal Verification Engine...</div>
            </Show>

            <Show when={!loading()}>
                <div class="space-y-4">
                    <For each={verdicts()}>
                        {(verdict) => (
                            <div
                                class={`border border-l-4 p-4 rounded bg-terminal-bg relative overflow-hidden ` +
                                    (verdict.is_valid ? 'border-neon-teal border-l-neon-teal' : 'border-alert-red border-l-alert-red')
                                }
                            >
                                <div class="absolute top-0 right-0 w-16 h-1 bg-gradient-to-l from-current opacity-50"></div>

                                <div class="flex justify-between items-start">
                                    <div>
                                        <h3 class="font-bold text-lg text-white font-mono flex items-center gap-2 m-0">
                                            {verdict.rule_name}
                                            <Show when={verdict.is_valid}>
                                                <span class="text-xs border border-neon-teal text-neon-teal px-1 rounded-sm uppercase bg-neon-teal/10">Passed</span>
                                            </Show>
                                            <Show when={!verdict.is_valid}>
                                                <span class="text-xs border border-alert-red text-alert-red px-1 rounded-sm uppercase bg-alert-red/10">Failed</span>
                                            </Show>
                                            <Show when={verdict.is_secured}>
                                                <span class="text-xs border border-purple-500 text-purple-400 px-1 rounded-sm uppercase bg-purple-500/10" style="margin-left: 8px;">Anti-ReDoS Shielded</span>
                                            </Show>
                                        </h3>
                                        <p class="text-xs font-mono opacity-70 mt-1 uppercase" style="margin: 4px 0;">ID: {verdict.rule_id}</p>
                                    </div>
                                </div>

                                <Show when={!verdict.is_valid && verdict.errors && verdict.errors.length > 0}>
                                    <div class="mt-4 bg-hacker-black border border-alert-red/30 p-3 rounded" style="margin-top: 16px;">
                                        <h4 class="text-xs uppercase font-bold text-alert-red mb-2 tracking-wider" style="margin: 0 0 8px 0;">AST Constraint Violations</h4>
                                        <ul class="list-disc pl-5 text-sm font-mono text-red-300 space-y-1" style="margin: 0; padding-left: 20px;">
                                            <For each={verdict.errors}>
                                                {(err) => <li>{err}</li>}
                                            </For>
                                        </ul>
                                    </div>
                                </Show>
                            </div>
                        )}
                    </For>
                    <Show when={verdicts().length === 0 && !loading()}>
                        <div class="text-center p-8 border border-dashed border-neon-blue text-neon-blue bg-neon-blue/5 rounded">
                            No detection rules found. Deploy .yaml policies to the data directory to initiate verification.
                        </div>
                    </Show>
                </div>
            </Show>
        </div>
    );
};
