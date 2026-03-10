import { Component, onMount, onCleanup, createSignal, createResource, For, Show } from 'solid-js';
import { render } from 'solid-js/web';
import { GoldenLayout, LayoutConfig } from 'golden-layout';
import 'golden-layout/dist/css/goldenlayout-base.css';
import 'golden-layout/dist/css/themes/goldenlayout-dark-theme.css';
import { TerminalView } from '../terminal/Terminal';
import { WindowFrame } from './WindowFrame';
import { AlertInvestigationPanel } from './panels/AlertInvestigationPanel';
import { LogSearchPanel } from './panels/LogSearchPanel';
import { TimelineInvestigationPanel } from './panels/TimelineInvestigationPanel';
import { MitreAttackPanel } from './panels/MitreAttackPanel';
import { HardwareTrustPanel } from './panels/HardwareTrustPanel';
import { ThreatGraphPanel } from './panels/ThreatGraphPanel';
import { GetAllMetrics } from '../../../wailsjs/go/app/MetricsService';
import { GetAllHealth } from '../../../wailsjs/go/app/HealthService';
import {
    ActivateAirGapMode,
    ActivateKillSwitch,
    DeactivateKillSwitch,
    TriggerNuclearDestruction,
    GetMode
} from '../../../wailsjs/go/app/DisasterService';

// Golden Layout integration for SOC Workspace

const DEFAULT_LAYOUT: LayoutConfig = {
    settings: {
        showPopoutIcon: true,
        showMaximiseIcon: true,
        showCloseIcon: true,
    },
    root: {
        type: 'row',
        content: [
            {
                type: 'column',
                width: 20,
                content: [
                    {
                        type: 'component',
                        componentType: 'Alerts',
                        title: 'ALERT INVESTIGATION',
                    },
                    {
                        type: 'component',
                        componentType: 'Mitre',
                        title: 'MITRE ATT&CK',
                    }
                ]
            },
            {
                type: 'column',
                width: 50,
                content: [
                    {
                        type: 'component',
                        componentType: 'Terminal',
                        title: 'SSH TERMINAL',
                    },
                    {
                        type: 'component',
                        componentType: 'Logs',
                        title: 'SIEM LOG SEARCH',
                    }
                ]
            },
            {
                type: 'column',
                width: 30,
                content: [
                    {
                        type: 'component',
                        componentType: 'Trust',
                        title: 'PLATFORM INTEGRITY',
                    },
                    {
                        type: 'component',
                        componentType: 'Graph',
                        title: 'THREAT GRAPH',
                    },
                    {
                        type: 'component',
                        componentType: 'Timeline',
                        title: 'EVENT TIMELINE',
                    }
                ]
            }
        ]
    }
};

export const SOCWorkspace: Component = () => {
    let containerRef: HTMLDivElement | undefined;
    let layout: GoldenLayout | undefined;
    const [warMode, setWarMode] = createSignal(false);
    const [disasterMode, { refetch: refetchMode }] = createResource(GetMode);

    // Minimize Dock State
    const [minimizedPanels, setMinimizedPanels] = createSignal<{id: string, title: string, componentType: string}[]>([]);

    // Fetch metrics for footer
    const [metrics] = createResource(async () => {
        try {
            const m = await GetAllMetrics();
            const health = await GetAllHealth();
            return {
                latency: m.find((x: any) => x.Name === 'system.latency')?.value || 12,
                eps: m.find((x: any) => x.Name === 'ingest.eps')?.value || 4204,
                nodes: Object.keys(health).length || 1,
                integrity: Object.values(health).every((h: any) => h.CPU < 90) ? 'Optimal' : 'Degraded'
            };
        } catch (e) {
            return { latency: 0, eps: 0, nodes: 0, integrity: 'Offline' };
        }
    });

    onMount(() => {
        if (!containerRef) return;

        const savedLayout = localStorage.getItem('oblivra-soc-layout');
        const config = savedLayout ? JSON.parse(savedLayout) : DEFAULT_LAYOUT;

        layout = new GoldenLayout(containerRef);

        // Register components
        const registerPanel = (name: string, title: string, status: string, component: any) => {
            layout?.registerComponentFactoryFunction(name, (container) => {
                const el = document.createElement('div');
                el.className = 'h-full w-full';
                container.getElement().appendChild(el);
                render(() => (
                    <WindowFrame
                        title={title}
                        status={status}
                        onClose={() => container.close()}
                        onMinimize={() => {
                            // Save to dock state
                            setMinimizedPanels(prev => [...prev, {
                                id: container.parent.id || Math.random().toString(36), // GL generates IDs
                                title,
                                componentType: name
                            }]);
                            // Close it from layout (we'll recreate it on restore)
                            container.close();
                        }}
                        onMaximize={() => (container as any).toggleMaximise()}
                        onPopout={() => (container as any).popout()}
                    >
                        {component}
                    </WindowFrame>
                ), el);
            });
        };

        registerPanel('Alerts', 'Alert Investigation', 'CRITICAL', <AlertInvestigationPanel />);
        registerPanel('Logs', 'SIEM Log Search', 'LIVE', <LogSearchPanel />);
        registerPanel('Timeline', 'Event Timeline', 'ANALYTIC', <TimelineInvestigationPanel />);
        registerPanel('Mitre', 'MITRE ATT&CK Matrix', 'COVERAGE', <MitreAttackPanel />);
        registerPanel('Trust', 'Platform Integrity', 'ATTESTED', <HardwareTrustPanel />);
        registerPanel('Graph', 'Multi-Hop Threat Graph', 'CORRELATED', <ThreatGraphPanel />);

        layout.registerComponentFactoryFunction('Terminal', (container) => {
            const el = document.createElement('div');
            el.className = 'h-full w-full';
            container.getElement().appendChild(el);
            render(() => (
                <WindowFrame
                    title="SSH Response Terminal"
                    status="CONNECTED"
                    onClose={() => container.close()}
                    onMinimize={() => {
                        setMinimizedPanels(prev => [...prev, { id: 'term', title: 'SSH Response Terminal', componentType: 'Terminal' }]);
                        container.close();
                    }}
                    onMaximize={() => (container as any).toggleMaximise()}
                    onPopout={() => (container as any).popout()}
                >
                    <TerminalView sessionId="soc-session" onData={() => { }} onResize={() => { }} />
                </WindowFrame>
            ), el);
        });

        layout.loadLayout(config);

        layout.on('stateChanged', () => {
            const layoutConfig = layout?.saveLayout();
            if (layoutConfig) {
                localStorage.setItem('oblivra-soc-layout', JSON.stringify(layoutConfig));
            }
        });

        // Ensure popout windows inherit all Sovereign tailwind and custom CSS
        layout.on('windowOpened', (newWindow: any) => {
            const externalWindow = newWindow.getGlWindow();
            if (externalWindow && externalWindow.document) {
                // Copy all <style> and <link rel="stylesheet"> tags from the parent document
                const headElements = document.querySelectorAll('style, link[rel="stylesheet"]');
                headElements.forEach(el => {
                    externalWindow.document.head.appendChild(el.cloneNode(true));
                });
                // Initialize the dark mode background explicitly
                externalWindow.document.body.className = "bg-surface-0 text-gray-300 font-sans antialiased overflow-hidden";
                externalWindow.document.body.style.backgroundColor = "#000000"; // Enforce black terminal theme
            }
        });

        const handleResize = () => {
            // @ts-ignore - updateSize signature in GL v2
            layout?.updateSize();
        };
        window.addEventListener('resize', handleResize);

        onCleanup(() => {
            window.removeEventListener('resize', handleResize);
            layout?.destroy();
        });
    });

    const handleNuclearDestruction = async () => {
        if (confirm("WARNING: NUCLEAR DESTRUCTION INITIATED.\n\nTHIS WILL CRYPTOGRAPHICALLY WIPE THE VAULT AND ALL DATA.\n\nPROCEED?")) {
            await TriggerNuclearDestruction();
            window.location.reload();
        }
    };

    const handleAirGap = async () => {
        await ActivateAirGapMode();
        refetchMode();
    };

    const handleKillSwitch = async () => {
        if (disasterMode() === 'KILL_SWITCH') {
            await DeactivateKillSwitch();
        } else {
            await ActivateKillSwitch("Manual tactical override");
        }
        refetchMode();
    };

    const restorePanel = (panel: {id: string, title: string, componentType: string}) => {
        if (!layout) return;
        
        // Add back to the root row
        layout.rootItem?.contentItems[0]?.addChild({
            type: 'component',
            title: panel.title,
            componentType: panel.componentType,
            componentState: {}
        } as any);
        
        setMinimizedPanels(prev => prev.filter(p => p.id !== panel.id));
    };

    return (
        <div class={`flex flex-col h-screen w-screen transition-colors duration-1000 overflow-hidden text-gray-300 ${warMode() ? 'war-mode-active' : 'bg-black'}`}>
            {/* Immersive CRT Layer */}
            <div class="crt-overlay"></div>
            <div class="crt-flicker"></div>

            {/* Workspace Header */}
            <header class={`flex items-center justify-between px-4 py-2 border-b h-10 select-none ${warMode() ? 'bg-red-900/40 border-red-800 war-mode-aberration' : 'bg-gray-900 border-gray-800'}`}>
                <div class="flex items-center gap-4">
                    <span class="text-xs font-black tracking-tighter text-white">OBLIVRA // SOC TERMINAL</span>
                    <div class="flex items-center gap-4 border-l border-gray-800 pl-4">
                        <div class="flex items-center gap-2">
                            <span class={`w-2 h-2 rounded-full ${metrics()?.integrity === 'Optimal' ? 'bg-green-500' : 'bg-red-500'} animate-pulse`}></span>
                            <span class="text-[9px] text-gray-400 font-mono uppercase tracking-widest">{disasterMode() || 'NORMAL'}</span>
                        </div>
                    </div>
                </div>

                <div class="flex items-center gap-2">
                    {/* Disaster Layer Controls */}
                    <button
                        onClick={handleAirGap}
                        class={`text-[9px] px-2 py-0.5 border font-mono font-bold uppercase transition-all ${disasterMode() === 'AIR_GAP' ? 'bg-blue-600 border-blue-500 text-white' : 'border-gray-700 text-gray-500 hover:border-blue-500 hover:text-blue-500'}`}
                    >
                        Air-Gap
                    </button>
                    <button
                        onClick={handleKillSwitch}
                        class={`text-[9px] px-2 py-0.5 border font-mono font-bold uppercase transition-all ${disasterMode() === 'KILL_SWITCH' ? 'bg-orange-600 border-orange-500 text-white' : 'border-gray-700 text-gray-500 hover:border-orange-500 hover:text-orange-500'}`}
                    >
                        Kill-Switch
                    </button>
                    <button
                        onClick={handleNuclearDestruction}
                        class="text-[9px] px-2 py-0.5 border border-red-900 bg-red-900/20 text-red-600 font-mono font-bold uppercase hover:bg-red-600 hover:text-white transition-all shadow-[0_0_15px_rgba(153,27,27,0.3)] hover:shadow-[0_0_20px_rgba(220,38,38,0.5)]"
                    >
                        Nuclear
                    </button>

                    <div class="h-4 w-px bg-gray-800 mx-2"></div>

                    <button
                        onClick={() => setWarMode(!warMode())}
                        class={`text-[10px] px-3 py-1 border font-mono font-bold uppercase tracking-tighter transition-all ${warMode()
                            ? 'bg-red-600 border-red-500 text-white animate-pulse shadow-[0_0_10px_rgba(220,38,38,0.5)]'
                            : 'bg-red-950/20 border-red-900 text-red-500 hover:bg-red-600 hover:text-white'
                            }`}
                    >
                        {warMode() ? 'WAR MODE ACTIVE' : 'WAR MODE'}
                    </button>

                    <button
                        class="p-1 text-gray-600 hover:text-white transition-colors"
                        onClick={() => {
                            localStorage.removeItem('oblivra-soc-layout');
                            window.location.reload();
                        }}
                        title="Reset Layout"
                    >
                        <svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                        </svg>
                    </button>
                </div>
            </header>

            {/* Golden Layout Root */}
            <div
                ref={containerRef}
                id="layout-container"
                class={`flex-1 w-full relative transition-opacity duration-500 ${warMode() ? 'war-mode-aberration' : ''} ${disasterMode() === 'KILL_SWITCH' ? 'opacity-40 grayscale pointer-events-none' : 'opacity-100'}`}
            >
                <div class="absolute inset-0 pointer-events-none border-t border-gray-800/50 shadow-[inset_0_2px_10px_rgba(0,0,0,0.5)] z-10"></div>
                {disasterMode() === 'KILL_SWITCH' && (
                    <div class="absolute inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm">
                        <div class="text-orange-500 font-mono font-black text-4xl animate-pulse tracking-tighter border-4 border-orange-500 p-8">
                            SYSTEM KILL-SWITCH ACTIVE
                        </div>
                    </div>
                )}
            </div>

            {/* Minimized Dock Taskbar */}
            <Show when={minimizedPanels().length > 0}>
                <div class={`h-8 bg-surface-1 border-t border-border-primary flex items-center px-2 gap-2 overflow-x-auto ${warMode() ? 'war-mode-aberration' : ''}`}>
                    <span class="text-[9px] font-mono font-black text-text-muted px-2 border-r border-border-primary">MINIMIZED</span>
                    <For each={minimizedPanels()}>
                        {(panel) => (
                            <button 
                                onClick={() => restorePanel(panel)}
                                class="px-3 py-1 bg-surface-2 hover:bg-surface-3 border border-border-primary rounded-sm text-[10px] font-mono text-text-primary flex items-center gap-2 transition-colors min-w-max truncate max-w-[200px]"
                                title={`Restore ${panel.title}`}
                            >
                                <span class="text-accent-primary">■</span> {panel.title}
                            </button>
                        )}
                    </For>
                </div>
            </Show>

            {/* Tactical Footer */}
            <footer class={`h-7 bg-gray-900 border-t border-gray-800 flex items-center px-4 justify-between text-[9px] font-mono text-gray-500 select-none uppercase ${warMode() ? 'war-mode-aberration' : ''}`}>
                <div class="flex items-center gap-5">
                    <div class="flex gap-1.5"><span class="opacity-50">LATENCY:</span> <span class="text-green-500 font-bold">{metrics()?.latency}ms</span></div>
                    <div class="flex gap-1.5"><span class="opacity-50">EPS:</span> <span class="text-blue-500 font-bold">{metrics()?.eps}</span></div>
                    <div class="flex gap-1.5"><span class="opacity-50">NODES</span> <span class="text-gray-300 font-bold">{metrics()?.nodes}</span></div>
                    <div class="flex gap-1.5"><span class="opacity-50">UPTIME:</span> <span class="text-gray-300">14d 02h 11m</span></div>
                </div>
                <div class="flex items-center gap-5">
                    <div class="flex gap-1.5"><span class="opacity-50">INTEGRITY:</span> <span class={`font-black ${metrics()?.integrity === 'Optimal' ? 'text-green-500' : 'text-red-500'}`}>{metrics()?.integrity === 'Optimal' ? 'VERIFIED' : 'DEGRADED'}</span></div>
                    <div class="flex gap-1.5"><span class="opacity-50">CONSENSUS:</span> <span class="text-blue-400">QUORUM REACHED</span></div>
                    <div class="flex gap-1.5"><span class="opacity-50">VAULT:</span> <span class="text-yellow-500">UNLOCKED</span></div>
                </div>
            </footer>
        </div>
    );
};
