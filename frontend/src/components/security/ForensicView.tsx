import { Component, createResource, For, Show } from 'solid-js';
import { AnalyzeFile, GetEvidence } from '../../../wailsjs/go/services/ForensicsService';

interface ForensicViewProps {
    evidenceId: string;
    onClose: () => void;
}

export const ForensicView: Component<ForensicViewProps> = (props) => {
    const [] = createResource(() => GetEvidence(props.evidenceId));
    const [analysis] = createResource(() => AnalyzeFile(props.evidenceId));

    const getEntropyColor = (val: number) => {
        if (val > 7.5) return 'var(--status-offline)';
        if (val > 6.0) return 'var(--status-degraded)';
        if (val > 4.0) return 'var(--status-online)';
        return 'var(--text-muted)';
    };

    return (
        <div style="position: fixed; top: 0; left: 0; right: 0; bottom: 0; z-index: 50; background: rgba(0,0,0,0.9); backdrop-filter: blur(12px); display: flex; align-items: center; justify-content: center; padding: 32px;" class="page-enter">
            <div class="ob-card" style="width: 100%; max-width: 1024px; height: 80vh; display: flex; flex-direction: column; padding: 0; overflow: hidden; box-shadow: 0 24px 64px rgba(0,0,0,0.8);">
                <header style="padding: 24px; border-bottom: 1px solid var(--border-primary); display: flex; justify-content: space-between; align-items: center; background: rgba(255,255,255,0.02);">
                    <div>
                        <h2 style="font-size: 24px; font-weight: 800; color: var(--text-primary); letter-spacing: -1px; text-transform: uppercase; font-style: italic; margin: 0 0 4px 0;">Forensic Deep-Dive</h2>
                        <p style="color: var(--text-muted); font-size: 11px; font-family: var(--font-mono); margin: 0;">Entity: {props.evidenceId}</p>
                    </div>
                    <button onClick={props.onClose} style="background: none; border: none; color: var(--text-muted); cursor: pointer; padding: 8px; transition: color 120ms ease;" onMouseEnter={(e) => e.currentTarget.style.color = 'var(--text-primary)'} onMouseLeave={(e) => e.currentTarget.style.color = 'var(--text-muted)'}>
                        <svg style="width: 24px; height: 24px;" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </header>

                <div style="flex: 1; overflow-y: auto; padding: 32px;">
                    <Show when={!analysis.loading} fallback={
                        <div style="display: flex; align-items: center; justify-content: center; height: 256px; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; text-transform: uppercase; letter-spacing: 2px;">Running Deep Entropy Analysis...</div>
                    }>
                        <div style="display: grid; grid-template-columns: 1fr 3fr; gap: 32px;">
                            <div style="display: flex; flex-direction: column; gap: 24px;">
                                <div style="padding: 16px; background: rgba(0,0,0,0.2); border-radius: 8px; border: 1px solid var(--border-primary);">
                                    <div style="font-size: 10px; color: var(--text-muted); font-weight: 800; text-transform: uppercase; margin-bottom: 4px;">Overall Entropy</div>
                                    <div style={`font-size: 36px; font-weight: 800; color: ${(analysis()?.risk_score ?? 0) >= 80 ? 'var(--status-offline)' : 'var(--status-online)'};`}>
                                        {analysis()?.overall_entropy ?? 0}
                                    </div>
                                    <div style="font-size: 10px; color: var(--text-muted); margin-top: 8px; font-family: var(--font-mono);">Max possible: 8.0</div>
                                </div>
                                <div style="padding: 16px; background: rgba(0,0,0,0.2); border-radius: 8px; border: 1px solid var(--border-primary);">
                                    <div style="font-size: 10px; color: var(--text-muted); font-weight: 800; text-transform: uppercase; margin-bottom: 4px;">Risk Score</div>
                                    <div style="font-size: 36px; font-weight: 800; color: var(--text-primary);">{analysis()?.risk_score}%</div>
                                    <div style="height: 6px; width: 100%; background: rgba(255,255,255,0.05); border-radius: 3px; margin-top: 12px; overflow: hidden;">
                                        <div style={`height: 100%; background: var(--status-offline); transition: all 300ms ease; width: ${analysis()?.risk_score}%;`} />
                                    </div>
                                </div>
                                <div style="padding: 16px; background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.2); border-radius: 8px; color: var(--status-offline); font-size: 12px; line-height: 1.6; font-style: italic;">
                                    {analysis()?.mitigation}
                                </div>
                            </div>

                            <div style="display: flex; flex-direction: column; gap: 32px;">
                                <section>
                                    <h3 style="font-size: 12px; font-weight: 800; color: var(--text-muted); text-transform: uppercase; margin: 0 0 16px 0; letter-spacing: 1px;">Entropy Distribution Profile</h3>
                                    <div style="display: flex; align-items: flex-end; gap: 4px; height: 128px; background: rgba(0,0,0,0.4); padding: 16px; border-radius: 8px; border: 1px solid var(--border-primary);">
                                        <For each={analysis()?.segments}>
                                            {(seg) => (
                                                <div
                                                    style={`flex: 1; min-width: 2px; border-radius: 2px 2px 0 0; background: ${getEntropyColor(seg.entropy)}; height: ${(seg.entropy / 8) * 100}%; transition: all 120ms ease; cursor: help; opacity: 0.8;`}
                                                    title={`Offset ${seg.offset}: Entropy ${seg.entropy}`}
                                                    onMouseEnter={(e) => { e.currentTarget.style.opacity = '1'; e.currentTarget.style.transform = 'scaleY(1.1)'; e.currentTarget.style.zIndex = '10'; }}
                                                    onMouseLeave={(e) => { e.currentTarget.style.opacity = '0.8'; e.currentTarget.style.transform = 'scaleY(1)'; e.currentTarget.style.zIndex = '1'; }}
                                                />
                                            )}
                                        </For>
                                    </div>
                                    <div style="display: flex; justify-content: space-between; margin-top: 8px; font-size: 8px; color: var(--text-muted); font-family: var(--font-mono);">
                                        <span>OFFSET_START</span>
                                        <span>DATA_STREAM_PROFILE (1KB_CHUNKS)</span>
                                        <span>OFFSET_END</span>
                                    </div>
                                </section>

                                <section style="background: rgba(255,255,255,0.02); padding: 24px; border-radius: 8px; border: 1px solid var(--border-primary);">
                                    <h3 style="font-size: 12px; font-weight: 800; color: var(--text-muted); text-transform: uppercase; margin: 0 0 16px 0; letter-spacing: 1px;">File Metadata</h3>
                                    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px; font-size: 12px; font-family: var(--font-mono);">
                                        <div style="display: flex; justify-content: space-between; border-bottom: 1px solid var(--border-primary); padding-bottom: 8px;">
                                            <span style="color: var(--text-muted);">PATH:</span>
                                            <span style="color: var(--text-primary);">{analysis()?.path}</span>
                                        </div>
                                        <div style="display: flex; justify-content: space-between; border-bottom: 1px solid var(--border-primary); padding-bottom: 8px;">
                                            <span style="color: var(--text-muted);">SIZE:</span>
                                            <span style="color: var(--text-primary);">{analysis()?.total_size} bytes</span>
                                        </div>
                                    </div>
                                </section>
                            </div>
                        </div>
                    </Show>
                </div>
            </div>
        </div>
    );
};
