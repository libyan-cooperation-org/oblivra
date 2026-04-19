<!-- OBLIVRA Web — Onboarding (Svelte 5) -->
<script lang="ts">
  import { push } from '../core/router.svelte';
  import { Button, Badge } from '@components/ui';
  import { 
    Monitor, 
    Terminal, 
    ShieldAlert, 
    Cpu, 
    Network, 
    Database, 
    FileText, 
    CheckCircle,
    Copy,
    ArrowRight,
    Zap,
    Activity
  } from 'lucide-svelte';

  let step      = $state(1);
  let platform  = $state('linux');
  let collectors = $state<string[]>(['process', 'network']);

  const collectorOptions = [
    { id:'process',  label:'Process Execution',    desc:'Monitor process creation, termination, and suspicious activity.', icon: Cpu },
    { id:'network',  label:'Network Connections',  desc:'Track inbound/outbound traffic and suspicious DNS queries.', icon: Network },
    { id:'file',     label:'File Integrity (FIM)', desc:'Audit modifications to critical system files and configuration.', icon: FileText },
    { id:'registry', label:'Registry Audit',       desc:'Monitor persistence mechanisms in Windows registry.', icon: Database },
    { id:'syslog',   label:'Syslog Ingest',         desc:'Forward local system logs to the OBLIVRA pipeline.', icon: Activity },
  ];

  function toggleCollector(id: string) {
    collectors = collectors.includes(id)
      ? collectors.filter(c => c !== id)
      : [...collectors, id];
  }

  const script = $derived.by(() => {
    const csv = collectors.join(',');
    const base = 'https://oblivra.enterprise.local:8443';
    if (platform === 'linux')   return `curl -sSL ${base}/scripts/install.sh | sudo bash -s -- --collectors ${csv}`;
    if (platform === 'windows') return `iex (iwr -UseBasicParsing ${base}/scripts/install.ps1).Content; Install-Oblivra -Collectors "${csv}"`;
    return `curl -sSL ${base}/scripts/install-mac.sh | bash -s -- --collectors ${csv}`;
  });

  function copyScript() {
    navigator.clipboard.writeText(script);
  }
</script>

<div class="min-h-screen bg-surface-0 flex flex-col items-center justify-center p-6 font-mono selection:bg-accent-primary selection:text-black">
  <div class="w-full max-w-3xl bg-surface-1 border border-border-primary p-12 shadow-premium relative overflow-hidden">
    <!-- Decorative background element -->
    <div class="absolute -right-20 -top-20 w-80 h-80 bg-accent-primary/5 rounded-full blur-3xl pointer-events-none"></div>

    <header class="mb-12 border-b border-border-primary pb-8 flex justify-between items-end">
      <div class="space-y-2">
        <h1 class="text-4xl font-black italic uppercase tracking-tighter text-text-heading">Fleet Onboarding</h1>
        <p class="text-[10px] font-mono text-text-muted uppercase tracking-[0.3em]">Deployment Wizard v1.1.0 // OrbitCA Enabled</p>
      </div>
      <div class="flex flex-col items-end gap-1">
        <span class="text-[9px] font-black text-text-muted uppercase tracking-widest">Phase</span>
        <span class="text-3xl font-black italic text-accent-primary">{step}<span class="text-text-muted text-sm ml-1">/ 03</span></span>
      </div>
    </header>

    {#if step === 1}
      <div class="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
        <div class="flex items-center gap-3">
          <div class="w-1.5 h-6 bg-accent-primary"></div>
          <h2 class="text-lg font-black uppercase tracking-widest text-text-heading italic">01. Select Target Substrate</h2>
        </div>

        <div class="grid grid-cols-3 gap-4">
          {#each [
            { id: 'linux', label: 'Linux', icon: Terminal, desc: 'Ubuntu, RHEL, Debian' },
            { id: 'windows', label: 'Windows', icon: Monitor, desc: 'Server 2019+, Win 10+' },
            { id: 'darwin', label: 'macOS', icon: Zap, desc: 'Intel & Apple Silicon' }
          ] as p}
            <button 
              class="flex flex-col items-center gap-4 p-8 border-2 transition-all group
                {platform === p.id 
                  ? 'bg-accent-primary/10 border-accent-primary' 
                  : 'bg-surface-2 border-border-subtle hover:border-text-muted'}"
              onclick={() => platform = p.id}
            >
              <p.icon size={32} class={platform === p.id ? 'text-accent-primary' : 'text-text-muted group-hover:text-text-secondary'} />
              <div class="text-center">
                <div class="text-xs font-black uppercase tracking-widest {platform === p.id ? 'text-accent-primary' : 'text-text-muted'}">{p.label}</div>
                <div class="text-[9px] font-mono text-text-muted opacity-60 uppercase mt-1">{p.desc}</div>
              </div>
            </button>
          {/each}
        </div>

        <div class="pt-8 flex justify-end">
          <Button variant="primary" class="font-black italic tracking-tighter px-10 py-4 text-lg" onclick={() => step = 2}>
            CONTINUE_TO_COLLECTORS <ArrowRight size={18} class="ml-2" />
          </Button>
        </div>
      </div>

    {:else if step === 2}
      <div class="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
        <div class="flex items-center gap-3">
          <div class="w-1.5 h-6 bg-accent-primary"></div>
          <h2 class="text-lg font-black uppercase tracking-widest text-text-heading italic">02. Configure Telemetry Shards</h2>
        </div>

        <div class="space-y-3">
          {#each collectorOptions as c}
            {@const active = collectors.includes(c.id)}
            <button 
              class="w-full flex items-center justify-between p-5 border transition-all text-left
                {active ? 'bg-accent-primary/10 border-accent-primary' : 'bg-surface-2 border-border-subtle hover:border-text-muted'}"
              onclick={() => toggleCollector(c.id)}
            >
              <div class="flex items-center gap-5">
                <div class="p-2 bg-surface-1 border border-border-subtle rounded-xs">
                  <c.icon size={18} class={active ? 'text-accent-primary' : 'text-text-muted'} />
                </div>
                <div>
                  <div class="text-[11px] font-black uppercase tracking-widest {active ? 'text-text-heading' : 'text-text-muted'}">{c.label}</div>
                  <div class="text-[9px] font-mono text-text-muted opacity-60 uppercase mt-1">{c.desc}</div>
                </div>
              </div>
              <div class="w-6 h-6 border-2 flex items-center justify-center transition-all
                {active ? 'border-accent-primary bg-accent-primary text-black' : 'border-border-subtle text-transparent'}">
                <CheckCircle size={14} />
              </div>
            </button>
          {/each}
        </div>

        <div class="pt-8 flex justify-between items-center">
          <button class="text-xs font-bold uppercase tracking-widest text-text-muted hover:text-text-heading underline underline-offset-4" onclick={() => step = 1}>
            ← Adjust Platform
          </button>
          <Button variant="primary" class="font-black italic tracking-tighter px-10 py-4 text-lg" onclick={() => step = 3}>
            GENERATE_DEPLOYMENT_SCRIPT <ArrowRight size={18} class="ml-2" />
          </Button>
        </div>
      </div>

    {:else}
      <div class="space-y-8 animate-in fade-in slide-in-from-bottom-4 duration-500">
        <div class="flex items-center gap-3">
          <div class="w-1.5 h-6 bg-accent-primary"></div>
          <h2 class="text-lg font-black uppercase tracking-widest text-text-heading italic">03. Finalize Deployment</h2>
        </div>

        <div class="bg-surface-2 border border-border-primary p-6 relative group overflow-hidden">
           <div class="absolute -right-2 -top-2 opacity-[0.03] grayscale">
              <Terminal size={120} />
           </div>
           
           <div class="flex justify-between items-center mb-4 relative z-10">
              <span class="text-[10px] font-black text-text-muted uppercase tracking-[0.2em]">One-Liner Execution Shard</span>
              <Badge variant="danger" size="xs" dot>PRIVILEGED_EXECUTION_REQUIRED</Badge>
           </div>
           
           <code class="block bg-surface-0 border border-border-subtle p-6 text-[13px] font-mono text-accent-primary leading-relaxed break-all mb-4 relative z-10 shadow-inner">
             {script}
           </code>
           
           <div class="flex justify-end relative z-10">
             <Button variant="secondary" size="sm" icon={Copy} onclick={copyScript}>COPY_TO_CLIPBOARD</Button>
           </div>
        </div>

        <div class="p-6 bg-surface-2 border-l-4 border-border-primary space-y-3">
           <div class="flex items-center gap-2 text-[10px] font-black text-text-heading uppercase tracking-widest">
              <ShieldAlert size={14} class="text-accent-primary" />
              Pre-Flight Checklist
           </div>
           <div class="space-y-1.5 text-[10px] font-mono text-text-muted uppercase tracking-tighter leading-relaxed">
             <div>1. VERIFY_ACCESS: PORTS 8443 (INGEST) AND 8080 (REST) MUST BE OPEN.</div>
             <div>2. RESOURCE_ALLOCATION: MINIMUM 2 VCPU, 4GB RAM REQUIRED PER NODE.</div>
             <div>3. AUTH_CHECK: RUN SCRIPT WITH SUDO OR ADMINISTRATOR PERMISSIONS.</div>
           </div>
        </div>

        <div class="pt-8 flex justify-between items-center">
          <button class="text-xs font-bold uppercase tracking-widest text-text-muted hover:text-text-heading underline underline-offset-4" onclick={() => step = 2}>
            ← Adjust Collectors
          </button>
          <Button variant="primary" class="font-black italic tracking-tighter px-10 py-4 text-lg" onclick={() => push('/')}>
            COMPLETE_ONBOARDING <CheckCircle size={18} class="ml-2" />
          </Button>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  :global(.animate-in) {
    animation: animate-in 0.5s ease-out;
  }
  @keyframes animate-in {
    from { opacity: 0; transform: translateY(10px); }
    to { opacity: 1; transform: translateY(0); }
  }
</style>
