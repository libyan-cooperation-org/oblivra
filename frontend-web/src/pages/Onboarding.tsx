import { createSignal, Show } from 'solid-js';

export default function Onboarding() {
  const [step, setStep] = createSignal(1);
  const [platform, setPlatform] = createSignal('linux');
  const [collectors, setCollectors] = createSignal(['process', 'network']);

  const toggleCollector = (c: string) => {
    if (collectors().includes(c)) {
      setCollectors(collectors().filter(item => item !== c));
    } else {
      setCollectors([...collectors(), c]);
    }
  };

  const getScript = () => {
    const collectorsCsv = collectors().join(',');
    const baseUrl = 'https://oblivra.enterprise.local:8443';
    
    if (platform() === 'linux') {
      return `curl -sSL ${baseUrl}/scripts/install.sh | sudo bash -s -- --collectors ${collectorsCsv}`;
    } else if (platform() === 'windows') {
      return `iex (iwr -UseBasicParsing ${baseUrl}/scripts/install.ps1).Content; Install-Oblivra -Collectors "${collectorsCsv}"`;
    } else {
      return `curl -sSL ${baseUrl}/scripts/install-mac.sh | bash -s -- --collectors ${collectorsCsv}`;
    }
  };

  return (
    <div class="min-h-screen bg-black text-white font-mono p-8">
      <div class="max-w-4xl mx-auto border border-zinc-800 bg-zinc-900/50 p-12 shadow-2xl">
        <header class="mb-12 border-b border-zinc-800 pb-8 flex justify-between items-end">
          <div>
            <h1 class="text-4xl font-black uppercase italic tracking-tighter">Fleet Onboarding</h1>
            <p class="text-zinc-500 mt-2 uppercase tracking-widest text-sm">Deployment Wizard v1.0.0-MVP</p>
          </div>
          <div class="text-right">
            <div class="text-[10px] text-zinc-600 uppercase mb-1">Step</div>
            <div class="text-2xl font-black text-red-600">{step()}/3</div>
          </div>
        </header>

        <Show when={step() === 1}>
          <div class="space-y-8">
            <h2 class="text-xl font-bold uppercase tracking-tight border-l-4 border-red-600 pl-4">01. Select Target Platform</h2>
            <div class="grid grid-cols-3 gap-6">
              {['linux', 'windows', 'darwin'].map(p => (
                <button
                  onClick={() => setPlatform(p)}
                  class={`p-8 border-2 transition-all uppercase font-black text-lg flex flex-col items-center gap-4 ${
                    platform() === p ? 'border-red-600 bg-red-600/10 text-white' : 'border-zinc-800 bg-black text-zinc-600 hover:border-zinc-600 hover:text-zinc-400'
                  }`}
                >
                  <span class="text-4xl">
                    {p === 'linux' ? '🐧' : p === 'windows' ? '🪟' : '🍎'}
                  </span>
                  {p}
                </button>
              ))}
            </div>
            <div class="flex justify-end pt-8">
              <button
                onClick={() => setStep(2)}
                class="bg-white text-black px-12 py-4 font-black uppercase hover:bg-red-600 hover:text-white transition-all shadow-xl"
              >
                Continue to Collectors
              </button>
            </div>
          </div>
        </Show>

        <Show when={step() === 2}>
          <div class="space-y-8">
            <h2 class="text-xl font-bold uppercase tracking-tight border-l-4 border-red-600 pl-4">02. Configure Collectors</h2>
            <div class="space-y-4">
              {[
                { id: 'process', label: 'Process Execution', desc: 'Monitor process creation, termination, and suspicious activity.' },
                { id: 'network', label: 'Network Connections', desc: 'Track inbound/outbound traffic and suspicious DNS queries.' },
                { id: 'file', label: 'File Integrity (FIM)', desc: 'Audit modifications to critical system files and configuration.' },
                { id: 'registry', label: 'Registry Audit', desc: 'Monitor persistence mechanisms in Windows registry.' },
                { id: 'syslog', label: 'Syslog Ingest', desc: 'Forward local system logs to the OBLIVRA pipeline.' }
              ].map(c => (
                <div
                  onClick={() => toggleCollector(c.id)}
                  class={`p-6 border cursor-pointer transition-all flex items-center justify-between ${
                    collectors().includes(c.id) ? 'border-red-600 bg-red-600/5' : 'border-zinc-800 bg-black hover:border-zinc-700'
                  }`}
                >
                  <div>
                    <div class={`font-black uppercase ${collectors().includes(c.id) ? 'text-white' : 'text-zinc-500'}`}>{c.label}</div>
                    <div class="text-xs text-zinc-600 mt-1 uppercase tracking-tight">{c.desc}</div>
                  </div>
                  <div class={`w-6 h-6 border-2 flex items-center justify-center ${collectors().includes(c.id) ? 'border-red-600 bg-red-600 text-black' : 'border-zinc-800 text-transparent'}`}>
                    ✓
                  </div>
                </div>
              ))}
            </div>
            <div class="flex justify-between pt-8">
              <button onClick={() => setStep(1)} class="text-zinc-500 uppercase font-bold hover:text-white transition-all underline underline-offset-8">Go Back</button>
              <button
                onClick={() => setStep(3)}
                class="bg-white text-black px-12 py-4 font-black uppercase hover:bg-red-600 hover:text-white transition-all shadow-xl"
              >
                Generate Deployment Script
              </button>
            </div>
          </div>
        </Show>

        <Show when={step() === 3}>
          <div class="space-y-8">
            <h2 class="text-xl font-bold uppercase tracking-tight border-l-4 border-red-600 pl-4">03. Finalize Deployment</h2>
            <div class="bg-black p-8 border border-zinc-800 relative group">
              <div class="text-[10px] text-zinc-600 uppercase mb-4 tracking-widest flex justify-between">
                <span>Deployment One-Liner</span>
                <span class="text-red-900 font-bold tracking-widest">Privileged Execution Required</span>
              </div>
              <code class="text-red-500 break-all text-sm block leading-relaxed selection:bg-red-600 selection:text-white">
                {getScript()}
              </code>
              <button
                onClick={() => {
                  navigator.clipboard.writeText(getScript());
                  alert('Copied to clipboard');
                }}
                class="absolute right-4 bottom-4 bg-zinc-800 text-zinc-400 text-[10px] uppercase font-bold px-3 py-1 hover:bg-zinc-700 hover:text-white transition-all border border-zinc-700"
              >
                Copy to Clipboard
              </button>
            </div>

            <div class="bg-zinc-900/50 p-6 border-l-4 border-zinc-700 text-xs text-zinc-400 leading-relaxed uppercase">
              <span class="text-white font-bold block mb-2 underline tracking-widest">Pre-Flight Checklist:</span>
              1. Ensure port 8443 (Ingest) and 8080 (REST) are accessible from the target host.<br />
              2. Validate that the target host satisfies minimum hardware requirements (2 vCPU, 4GB RAM).<br />
              3. Run the script with root/administrator privileges.
            </div>

            <div class="flex justify-between pt-8">
               <button onClick={() => setStep(2)} class="text-zinc-500 uppercase font-bold hover:text-white transition-all underline underline-offset-8">Adjust Config</button>
              <button
                onClick={() => window.location.href = '/'}
                class="border-2 border-white text-white px-12 py-4 font-black uppercase hover:bg-white hover:text-black transition-all shadow-xl"
              >
                Complete Onboarding
              </button>
            </div>
          </div>
        </Show>
      </div>
    </div>
  );
}
