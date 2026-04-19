<!-- OBLIVRA Web — MITRE Heatmap (Svelte 5) -->
<script lang="ts">
   import { onMount } from "svelte";
   import { Badge, Button, PageLayout, Spinner } from "@components/ui";
   import { Zap, History, ExternalLink, Activity } from "lucide-svelte";
   import { request } from "../services/api";

   // -- Types --
   interface TechniqueCell {
      id: string;
      name: string;
      hits: number;
   }
   interface TacticRow {
      id: string;
      name: string;
      techniques: TechniqueCell[];
   }
   interface HeatmapData {
      tactics: TacticRow[];
      total_hits: number;
      last_updated: string;
   }

   // -- Constants --
   const TACTIC_ORDER = [
      "TA0001",
      "TA0002",
      "TA0003",
      "TA0004",
      "TA0005",
      "TA0006",
      "TA0007",
      "TA0008",
      "TA0009",
      "TA0011",
      "TA0010",
      "TA0040",
   ];

   // -- State --
   let data = $state<HeatmapData | null>(null);
   let loading = $state(true);
   let selected = $state<TechniqueCell | null>(null);
   let showZero = $state(false);

   // -- Helpers --
   const maxHits = $derived.by(() => {
      let m = 0;
      data?.tactics.forEach((t) =>
         t.techniques.forEach((tc) => {
            if (tc.hits > m) m = tc.hits;
         }),
      );
      return m;
   });

   const heatColor = (hits: number) => {
      if (hits === 0)
         return {
            bg: "var(--surface-0)",
            border: "var(--border-subtle)",
            text: "var(--text-muted)",
         };
      const ratio = Math.min(hits / Math.max(maxHits, 1), 1);
      if (ratio > 0.75)
         return {
            bg: "rgba(200,44,44,0.1)",
            border: "var(--alert-critical)",
            text: "var(--alert-critical)",
         };
      if (ratio > 0.5)
         return {
            bg: "rgba(200,80,0,0.1)",
            border: "var(--alert-high)",
            text: "var(--alert-high)",
         };
      if (ratio > 0.25)
         return {
            bg: "rgba(200,140,0,0.1)",
            border: "var(--alert-medium)",
            text: "var(--alert-medium)",
         };
      return {
         bg: "rgba(0,200,100,0.1)",
         border: "var(--status-online)",
         text: "var(--status-online)",
      };
   };

   const highIntensityTactics = $derived(
      data
         ? data.tactics
              .filter((t) => t.techniques.some((tc) => tc.hits > maxHits * 0.5))
              .slice(0, 2)
         : [],
   );

   // -- Actions --
   async function fetchData() {
      loading = true;
      try {
         const res = await request<HeatmapData>("/mitre/heatmap");
         // Sort tactics
         res.tactics = [...res.tactics].sort(
            (a, b) => TACTIC_ORDER.indexOf(a.id) - TACTIC_ORDER.indexOf(b.id),
         );
         data = res;
      } catch (e) {
         console.error("MITRE Heatmap fetch failed", e);
      } finally {
         loading = false;
      }
   }

   onMount(() => {
      fetchData();
   });
</script>

<PageLayout
   title="MITRE ATT&CK® Navigator"
   subtitle="Adversary techniques and coverage mapping: Real-time hit frequency across the offensive substrate"
>
   {#snippet toolbar()}
      <div class="flex items-center gap-2">
         <Button
            variant="secondary"
            size="sm"
            onclick={() => (showZero = !showZero)}
         >
            {showZero ? "HIDE UNUSED" : "SHOW ALL"}
         </Button>
         <Button variant="primary" size="sm" onclick={fetchData}>
            <History size={14} class="mr-2" />
            RE-SYNC
         </Button>
      </div>
   {/snippet}

   <div class="flex flex-col h-full gap-0 -m-6 overflow-hidden">
      <!-- METRIC STRIP -->
      <div
         class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0"
      >
         <div class="bg-surface-2 p-3">
            <div
               class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1"
            >
               Total Hits (24h)
            </div>
            <div class="text-xl font-mono font-bold text-text-heading">
               {data?.total_hits.toLocaleString() ?? 0}
            </div>
            <div class="text-[9px] text-text-muted mt-1">
               Aggregated detections
            </div>
         </div>
         <div class="bg-surface-2 p-3">
            <div
               class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1"
            >
               Triggered Techniques
            </div>
            <div class="text-xl font-mono font-bold text-accent-primary">
               {data?.tactics.reduce(
                  (acc, t) =>
                     acc + t.techniques.filter((tc) => tc.hits > 0).length,
                  0,
               ) ?? 0}
               <span class="text-[10px] text-text-muted font-normal">
                  / {data?.tactics.reduce(
                     (acc, t) => acc + t.techniques.length,
                     0,
                  ) ?? 0}</span
               >
            </div>
            <div class="text-[9px] text-text-muted mt-1">
               Unique V15 identifiers
            </div>
         </div>
         <div class="bg-surface-2 p-3">
            <div
               class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1"
            >
               High Intensity Tactics
            </div>
            <div class="flex gap-1.5 mt-1">
               {#each highIntensityTactics as t}
                  <Badge variant="danger" size="xs"
                     >{t.name.toUpperCase()}</Badge
                  >
               {:else}
                  <Badge variant="success" size="xs">NOMINAL</Badge>
               {/each}
            </div>
         </div>
         <div class="bg-surface-2 p-3">
            <div
               class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1"
            >
               Last Matrix Sync
            </div>
            <div class="text-xl font-mono font-bold text-status-online">
               {data?.last_updated
                  ? new Date(data.last_updated).toLocaleTimeString()
                  : "WAITING"}
            </div>
            <div class="text-[9px] text-status-online mt-1">
               Baseline Synchronized
            </div>
         </div>
      </div>

      <!-- MAIN BODY -->
      <div class="flex-1 flex flex-col min-h-0 bg-surface-0 overflow-hidden">
         <!-- LEGEND BAR -->
         <div
            class="bg-surface-1 border-b border-border-primary px-6 py-2 flex items-center gap-6 shrink-0"
         >
            <span
               class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest"
               >Heatmap Intensity:</span
            >
            <div class="flex items-center gap-4">
               {#each [{ label: "NONE", bg: "var(--surface-0)", border: "var(--border-subtle)", text: "var(--text-muted)" }, { label: "LOW", bg: "rgba(0,200,100,0.1)", border: "var(--status-online)", text: "var(--status-online)" }, { label: "MED", bg: "rgba(200,140,0,0.1)", border: "var(--alert-medium)", text: "var(--alert-medium)" }, { label: "HIGH", bg: "rgba(200,80,0,0.1)", border: "var(--alert-high)", text: "var(--alert-high)" }, { label: "CRIT", bg: "rgba(200,44,44,0.1)", border: "var(--alert-critical)", text: "var(--alert-critical)" }] as l}
                  <div class="flex items-center gap-2">
                     <div
                        class="w-3 h-3 rounded-xs border"
                        style="background: {l.bg}; border-color: {l.border}"
                     ></div>
                     <span
                        class="text-[8px] font-mono font-bold"
                        style="color: {l.text}">{l.label}</span
                     >
                  </div>
               {/each}
            </div>
            <div class="ml-auto flex items-center gap-2">
               <Zap size={12} class="text-accent-primary animate-pulse" />
               <span
                  class="text-[9px] font-mono text-text-muted uppercase tracking-tighter"
                  >Live adversary substrate mapping active</span
               >
            </div>
         </div>

         <div class="flex-1 overflow-auto p-6">
            {#if loading}
               <div
                  class="h-full flex flex-col items-center justify-center gap-4 text-text-muted"
               >
                  <Spinner size="lg" />
                  <p
                     class="font-mono text-[10px] uppercase tracking-widest animate-pulse"
                  >
                     Initializing ATT&CK Matrix Shards...
                  </p>
               </div>
            {:else if data}
               <div
                  class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-4"
               >
                  {#each data.tactics as tactic}
                     <div
                        class="bg-surface-1 border border-border-primary rounded-sm overflow-hidden flex flex-col shadow-premium"
                     >
                        <!-- Tactic Header -->
                        <div
                           class="bg-surface-2 border-b border-border-primary p-3 space-y-1"
                        >
                           <div class="flex justify-between items-start">
                              <span
                                 class="text-[8px] font-mono text-text-muted uppercase tracking-widest"
                                 >{tactic.id}</span
                              >
                              <Badge variant="secondary" size="xs"
                                 >{tactic.techniques.filter((tc) => tc.hits > 0)
                                    .length}</Badge
                              >
                           </div>
                           <div
                              class="text-[11px] font-black text-text-heading uppercase tracking-tighter leading-tight"
                           >
                              {tactic.name}
                           </div>
                        </div>

                        <!-- Techniques -->
                        <div class="p-1 space-y-1 overflow-auto">
                           {#each showZero ? tactic.techniques : tactic.techniques.filter((tc) => tc.hits > 0) as tech}
                              {@const style = heatColor(tech.hits)}
                              <button
                                 class="w-full text-left p-2 rounded-xs border transition-all group relative overflow-hidden
                                {selected?.id === tech.id
                                    ? 'border-accent-primary ring-1 ring-accent-primary ring-offset-1 ring-offset-surface-1'
                                    : ''}"
                                 style="background: {style.bg}; border-color: {style.border}"
                                 onclick={() =>
                                    (selected =
                                       selected?.id === tech.id ? null : tech)}
                              >
                                 <div
                                    class="flex justify-between items-start relative z-10"
                                 >
                                    <div class="flex flex-col min-w-0">
                                       <span
                                          class="text-[9px] font-black uppercase tracking-tighter italic"
                                          style="color: {style.text}"
                                          >{tech.id}</span
                                       >
                                       <span
                                          class="text-[10px] font-bold text-text-secondary truncate block group-hover:text-text-heading transition-colors"
                                          title={tech.name}>{tech.name}</span
                                       >
                                    </div>
                                    {#if tech.hits > 0}
                                       <span
                                          class="text-[9px] font-black italic"
                                          style="color: {style.text}"
                                          >{tech.hits}</span
                                       >
                                    {/if}
                                 </div>
                                 {#if selected?.id === tech.id}
                                    <div
                                       class="absolute inset-y-0 left-0 w-1 bg-accent-primary"
                                    ></div>
                                 {/if}
                              </button>
                           {/each}
                        </div>
                     </div>
                  {/each}
               </div>
            {/if}
         </div>
      </div>

      <!-- DETAIL PANEL (Overlay) -->
      {#if selected}
         {@const style = heatColor(selected.hits)}
         <div
            class="fixed right-8 bottom-12 w-80 bg-surface-2 border border-border-primary rounded-md shadow-premium z-50 overflow-hidden animate-in fade-in slide-in-from-bottom-4 duration-300"
         >
            <div
               class="p-4 bg-surface-3 border-b border-border-primary flex justify-between items-start"
            >
               <div class="flex flex-col gap-1">
                  <span
                     class="text-[10px] font-black uppercase tracking-widest italic"
                     style="color: {style.text}">{selected.id}</span
                  >
                  <h3
                     class="text-sm font-black text-text-heading leading-tight uppercase"
                  >
                     {selected.name}
                  </h3>
               </div>
               <button
                  class="text-text-muted hover:text-text-primary"
                  onclick={() => (selected = null)}>✕</button
               >
            </div>
            <div class="p-5 space-y-4">
               <div
                  class="flex justify-between items-end border-b border-border-subtle pb-4"
               >
                  <div class="flex flex-col">
                     <span
                        class="text-[9px] font-mono text-text-muted uppercase tracking-widest"
                        >Observed Hits</span
                     >
                     <span
                        class="text-3xl font-mono font-black italic"
                        style="color: {style.text}"
                        >{selected.hits.toLocaleString()}</span
                     >
                  </div>
                  <div class="flex flex-col items-end">
                     <span
                        class="text-[9px] font-mono text-text-muted uppercase tracking-widest"
                        >Intensity</span
                     >
                     <Badge
                        variant={selected.hits > maxHits * 0.75
                           ? "danger"
                           : selected.hits > maxHits * 0.25
                             ? "warning"
                             : "success"}
                        size="xs"
                        class="font-bold"
                     >
                        {Math.round(
                           (selected.hits / Math.max(maxHits, 1)) * 100,
                        )}% REL
                     </Badge>
                  </div>
               </div>

               <div class="space-y-2">
                  <span
                     class="text-[9px] font-mono text-text-muted uppercase tracking-widest"
                     >Resource Linkage</span
                  >
                  <a
                     href={`https://attack.mitre.org/techniques/${selected.id}/`}
                     target="_blank"
                     class="flex items-center justify-between p-3 bg-surface-1 border border-border-primary rounded-sm hover:border-accent-primary transition-colors group"
                  >
                     <span
                        class="text-[10px] font-bold text-text-secondary group-hover:text-text-heading uppercase"
                        >MITRE Navigator Profile</span
                     >
                     <ExternalLink
                        size={14}
                        class="text-text-muted group-hover:text-accent-primary transition-colors"
                     />
                  </a>
               </div>

               <Button
                  variant="primary"
                  size="sm"
                  class="w-full"
                  icon={Activity}>PIVOT TO TELEMETRY</Button
               >
            </div>
         </div>
      {/if}

      <!-- STATUS BAR -->
      <div
         class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest"
      >
         <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>MATRIX_CORE:</span>
            <span class="text-status-online font-bold italic">V15_SYNCED</span>
         </div>
         <span class="text-border-primary opacity-30">|</span>
         <div class="flex items-center gap-1.5">
            <span>MAPPING_DEPTH:</span>
            <span class="text-status-online font-bold italic"
               >FULL_TECHNIQUE_SET</span
            >
         </div>
         <span class="text-border-primary opacity-30">|</span>
         <div class="flex items-center gap-1.5">
            <span>ADVERSARY_RELATION:</span>
            <span class="text-accent-primary font-bold italic">ENABLED</span>
         </div>
         <div class="ml-auto opacity-40">OBLIVRA_MITRE_NAVIGATOR v4.2.1</div>
      </div>
   </div>
</PageLayout>

<style>
   :global(.flex-1::-webkit-scrollbar) {
      width: 6px;
      height: 6px;
   }
   :global(.flex-1::-webkit-scrollbar-track) {
      background: var(--surface-0);
   }
   :global(.flex-1::-webkit-scrollbar-thumb) {
      background: var(--border-primary);
      border-radius: 3px;
   }
</style>
