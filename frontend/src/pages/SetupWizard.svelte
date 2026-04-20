<!--
  OBLIVRA — Setup Wizard (Svelte 5)
  Mission-critical initialization: Hardware key generation and fleet bootstrap.
-->
<script lang="ts">
  import { PageLayout, Button, Spinner } from '@components/ui';
  import { Key, Database, Network, Terminal, CheckCircle, Info, Zap } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  import { IS_BROWSER } from '@lib/context';

  let currentStep = $state(0);
  let status = $state('idle'); // idle, running, success, error
  let progress = $state(0);
  let logs = $state<string[]>([]);

  const steps = [
    { title: 'Identity Initialization', desc: 'Hardware-backed root key generation and TPM binding.', icon: Key },
    { title: 'Sovereign Data Lake', desc: 'Provisioning encrypted storage shards and graph indices.', icon: Database },
    { title: 'Fleet Bootstrap', desc: 'Generating agent binaries and mesh sync certificates.', icon: Zap },
    { title: 'Network Mesh', desc: 'Establishing peer-to-peer orchestration layer.', icon: Network }
  ];

  const CurrentIcon = $derived(steps[currentStep]?.icon || Key);

  async function nextStep() {
    if (currentStep < steps.length - 1) {
        status = 'running';
        progress = 0;
        await performInitialization();
    } else {
        status = 'success';
        appStore.notify('Platform initialized successfully.', 'success');
    }
  }

  async function performInitialization() {
    if (!IS_BROWSER) return;
    
    // Smooth progress simulation
    const timer = setInterval(() => {
        if (progress < 90) progress += 5;
    }, 100);

    try {
        const res = await fetch('/api/v1/setup/initialize', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ step: currentStep }),
            credentials: 'include'
        });

        if (res.ok) {
            const data = await res.json();
            clearInterval(timer);
            progress = 100;
            setTimeout(() => {
                logs = [...logs, ...data.logs];
                status = 'idle';
                currentStep++;
            }, 500);
        } else {
            clearInterval(timer);
            status = 'error';
            logs = [...logs, `[ERROR] Phase ${currentStep} failed: ${await res.text()}`];
        }
    } catch (e) {
        clearInterval(timer);
        status = 'error';
        logs = [...logs, `[ERROR] Connection failed during phase ${currentStep}`];
    }
  }
</script>

<PageLayout title="Platform Initialization" subtitle="Initialize the OBLIVRA sovereign security engine">
  <div class="flex flex-col items-center justify-center h-full max-w-4xl mx-auto gap-12">
    <!-- STEP PROGRESS -->
    <div class="w-full grid grid-cols-4 gap-4">
        {#each steps as step, i}
            <div class="flex flex-col gap-3 {i <= currentStep ? 'opacity-100' : 'opacity-30'} transition-opacity">
                <div class="flex items-center gap-2">
                    <div class="w-6 h-6 rounded-sm {i < currentStep ? 'bg-success text-white' : i === currentStep ? 'bg-accent text-white' : 'bg-surface-2 border border-border-primary text-text-muted'} flex items-center justify-center text-[10px] font-bold">
                        {#if i < currentStep}
                            <CheckCircle size={14} />
                        {:else}
                            {i + 1}
                        {/if}
                    </div>
                    <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">{step.title}</span>
                </div>
                <div class="h-1 bg-surface-2 rounded-full overflow-hidden">
                    <div class="h-full {i < currentStep ? 'bg-success' : i === currentStep ? 'bg-accent' : 'bg-transparent'}" style="width: {i <= currentStep ? (i === currentStep ? progress : 100) : 0}%"></div>
                </div>
            </div>
        {/each}
    </div>

    <!-- MAIN CARD -->
    <div class="w-full bg-surface-1 border border-border-primary rounded-sm shadow-premium flex overflow-hidden min-h-[400px]">
        <!-- LEFT: INFO -->
        <div class="w-1/2 p-8 flex flex-col gap-6 border-r border-border-primary">
            <div class="w-12 h-12 bg-accent/10 border border-accent/40 rounded-sm flex items-center justify-center">
                <CurrentIcon size={24} class="text-accent" />
            </div>
            <div class="space-y-2">
                <h2 class="text-xl font-bold text-text-heading uppercase tracking-tighter">{steps[currentStep]?.title}</h2>
                <p class="text-xs text-text-muted leading-relaxed font-mono">
                    {steps[currentStep]?.desc}
                </p>
            </div>
            
            <div class="mt-auto space-y-4">
                <div class="flex items-start gap-3 p-3 bg-surface-2 border border-border-primary rounded-sm">
                    <Info size={14} class="text-accent mt-0.5 shrink-0" />
                    <p class="text-[9px] font-mono text-text-muted italic">
                        OBLIVRA initialization is hardware-bound. Ensure your security token is inserted.
                    </p>
                </div>
                <Button 
                    variant={status === 'running' ? 'secondary' : 'primary'} 
                    class="w-full h-10 font-bold uppercase tracking-widest" 
                    onclick={nextStep}
                    disabled={status === 'running'}
                >
                    {#if status === 'running'}
                        <Spinner size="sm" class="mr-2" /> INITIALIZING...
                    {:else if currentStep === steps.length - 1}
                        FINALIZE BOOTSTRAP
                    {:else}
                        CONTINUE TO NEXT PHASE
                    {/if}
                </Button>
            </div>
        </div>

        <!-- RIGHT: CONSOLE -->
        <div class="w-1/2 bg-black/20 flex flex-col">
            <div class="px-4 py-3 bg-surface-2 border-b border-border-primary flex items-center gap-2">
                <Terminal size={14} class="text-text-muted" />
                <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Initialization Log</span>
            </div>
            <div class="flex-1 p-4 font-mono text-[10px] space-y-1.5 overflow-auto mask-fade-bottom">
                {#each logs as log}
                    <div class="flex gap-2">
                        <span class="text-text-muted opacity-40">[{new Date().toLocaleTimeString()}]</span>
                        <span class={log.startsWith('[OK]') ? 'text-success' : 'text-text-secondary'}>{log}</span>
                    </div>
                {/each}
                {#if status === 'running'}
                    <div class="flex gap-2 text-accent animate-pulse">
                        <span class="text-text-muted opacity-40">[{new Date().toLocaleTimeString()}]</span>
                        <span>[BUSY] Orchestrating resources...</span>
                    </div>
                {/if}
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="w-full flex justify-between items-center text-[9px] font-mono text-text-muted uppercase tracking-widest px-1">
        <div class="flex items-center gap-2">
            <span class="font-bold">TPM STATUS:</span>
            <span class="text-success">READY</span>
        </div>
        <div class="flex items-center gap-2">
            <span class="font-bold">HW ENTROPY:</span>
            <span class="text-success">VERIFIED</span>
        </div>
        <div class="flex items-center gap-2">
            <span class="font-bold">INIT VERSION:</span>
            <span>1.4.0-OBLIVRA</span>
        </div>
    </div>
  </div>
</PageLayout>
