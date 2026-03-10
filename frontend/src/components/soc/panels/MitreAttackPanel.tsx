import { Component, For, createResource } from 'solid-js';
import { GetCapabilitiesMatrix } from '../../../../wailsjs/go/app/PolicyService';

export const MitreAttackPanel: Component = () => {
    const [capabilities] = createResource(async () => {
        try {
            return await GetCapabilitiesMatrix();
        } catch (e) {
            console.error("Failed to fetch capabilities:", e);
            return {};
        }
    });

    const tactics = [
        { name: 'Initial Access', technique: 'T1190 - Exploit Public-Facing Application' },
        { name: 'Execution', technique: 'T1059 - Command and Scripting Interpreter' },
        { name: 'Persistence', technique: 'T1547 - Boot or Logon Autostart Execution' },
        { name: 'Privilege Escalation', technique: 'T1068 - Exploitation for Privilege Escalation' },
        { name: 'Defense Evasion', technique: 'T1562 - Impair Defenses' },
        { name: 'Credential Access', technique: 'T1003 - OS Credential Dumping' },
        { name: 'Discovery', technique: 'T1046 - Network Service Discovery' },
        { name: 'Lateral Movement', technique: 'T1021 - Remote Services' },
    ];

    return (
        <div class="h-full flex flex-col bg-gray-950 font-mono text-[11px] overflow-hidden">
            <div class="p-3 border-b border-gray-800 flex justify-between items-center bg-gray-900/30">
                <span class="text-gray-400 font-bold tracking-widest uppercase">ATT&CK Coverage Matrix</span>
                <span class="text-blue-500 font-bold text-[9px]">V12.1</span>
            </div>

            <div class="flex-1 overflow-auto p-3">
                <div class="grid grid-cols-2 gap-2">
                    <For each={tactics}>
                        {(tactic) => {
                            const isActive = capabilities()?.[tactic.name] || false;
                            return (
                                <div class={`p-2 border transition-all duration-500 ${isActive ? 'bg-blue-950/20 border-blue-900 shadow-[inset_0_0_10px_rgba(30,58,138,0.2)]' : 'bg-gray-900/40 border-gray-800 opacity-60'}`}>
                                    <div class="text-[9px] uppercase text-gray-500 mb-1 font-bold">{tactic.name}</div>
                                    <div class={`text-[10px] leading-tight ${isActive ? 'text-blue-400 font-bold' : 'text-gray-600 italic'}`}>
                                        {tactic.technique}
                                    </div>
                                    {isActive && (
                                        <div class="mt-2 flex gap-1">
                                            <div class="w-1.5 h-1.5 bg-blue-500 rounded-full animate-pulse"></div>
                                            <span class="text-[8px] text-blue-300 font-black uppercase">Active Shield</span>
                                        </div>
                                    )}
                                </div>
                            );
                        }}
                    </For>
                </div>
            </div>

            <div class="p-2 border-t border-gray-800 bg-gray-900/50 flex justify-between items-center px-4">
                <div class="flex gap-4 items-center">
                    <span class="text-[9px] text-gray-600">COVERAGE: <span class="text-blue-500">84.2%</span></span>
                    <span class="text-[9px] text-gray-600">DETECTIONS: <span class="text-blue-500">142</span></span>
                </div>
                <button class="text-[9px] text-blue-500 font-bold hover:text-white transition-colors">FULL MATRIX</button>
            </div>
        </div>
    );
};
