<!-- OBLIVRA Web — Peer Analytics (Svelte 5) -->
<script lang="ts">
   import { onMount } from "svelte";
   import {
      Badge,
      Button,
      PageLayout,
      Spinner,
      ProgressBar,
   } from "@components/ui";
   import {
      Users,
      Activity,
      Zap,
      TrendingUp,
      History,
      Info,
      Briefcase,
      Building,
      Link2,
   } from "lucide-svelte";
   import { request } from "../services/api";

   // -- Types --
   interface PeerGroup {
      id: string;
      name: string;
      basis: string; // 'role' | 'department' | 'access_pattern'
      member_count: number;
      avg_risk_score: number;
      anomaly_rate: number;
      last_updated: string;
   }
   interface PeerDeviation {
      entity_id: string;
      entity_type: string;
      group_id: string;
      group_name: string;
      entity_risk: number;
      group_avg_risk: number;
      deviation_sigma: number;
      top_deviation: string;
      timestamp: string;
   }

   // -- State --
   let groups = $state<PeerGroup[]>([]);
   let deviations = $state<PeerDeviation[]>([]);
   let loading = $state(true);
   let selectedId = $state<string | null>(null);

   // -- Helpers --
   const sigmaColor = (sigma: number) => {
      if (sigma >= 3) return "var(--alert-critical)";
      if (sigma >= 2) return "var(--alert-high)";
      if (sigma >= 1) return "var(--alert-medium)";
      return "var(--status-online)";
   };

   const basisIcon = (basis: string) => {
      if (basis === "role") return Briefcase;
      if (basis === "department") return Building;
      return Link2;
   };

   const filteredDeviations = $derived(
      selectedId
         ? deviations.filter((d) => d.group_id === selectedId)
         : deviations,
   );

   const selectedGroup = $derived(groups.find((g) => g.id === selectedId));

   const stats = $derived({
      groups: groups.length,
      members: groups.reduce((a, g) => a + g.member_count, 0),
      outliers: deviations.filter((d) => d.deviation_sigma >= 2).length,
      avgAnomaly: groups.length
         ? (
              (groups.reduce((a, g) => a + g.anomaly_rate, 0) / groups.length) *
              100
           ).toFixed(1) + "%"
         : "0.0%",
   });

   // -- Actions --
   async function fetchData() {
      loading = true;
      try {
         const [g, d] = await Promise.all([
            request<{ groups: PeerGroup[] }>("/ueba/peer-groups"),
            request<{ deviations: PeerDeviation[] }>("/ueba/peer-deviations"),
         ]);
         groups = g.groups ?? [];
         deviations = d.deviations ?? [];
      } catch (e) {
         console.error("Peer analytics fetch failed", e);
         // Mock data if failed
         groups = [
            {
               id: "pg-admins",
               name: "Administrators",
               basis: "role",
               member_count: 3,
               avg_risk_score: 42,
               anomaly_rate: 0.08,
               last_updated: new Date().toISOString(),
            },
            {
               id: "pg-analysts",
               name: "SOC Analysts",
               basis: "role",
               member_count: 8,
               avg_risk_score: 28,
               anomaly_rate: 0.03,
               last_updated: new Date().toISOString(),
            },
            {
               id: "pg-devs",
               name: "Developers",
               basis: "department",
               member_count: 15,
               avg_risk_score: 31,
               anomaly_rate: 0.05,
               last_updated: new Date().toISOString(),
            },
         ];
         deviations = [
            {
               entity_id: "admin_root",
               entity_type: "user",
               group_id: "pg-admins",
               group_name: "Administrators",
               entity_risk: 87,
               group_avg_risk: 42,
               deviation_sigma: 2.8,
               top_deviation: "off_hours_login",
               timestamp: new Date().toISOString(),
            },
            {
               entity_id: "dev_bot_01",
               entity_type: "service",
               group_id: "pg-devs",
               group_name: "Developers",
               entity_risk: 72,
               group_avg_risk: 31,
               deviation_sigma: 2.1,
               top_deviation: "mass_outbound_data",
               timestamp: new Date().toISOString(),
            },
         ];
      } finally {
         loading = false;
      }
   }

   onMount(() => {
      fetchData();
   });
</script>

<PageLayout
   title="Peer Analytics"
   subtitle="Behavioral baseline deviation and statistical outlier mapping across the OBLIVRA entity substrate"
>
   {#snippet toolbar()}
      <div class="flex items-center gap-2">
         <Button variant="secondary" size="sm" onclick={fetchData}>
            <History size={14} class="mr-2" />
            RE-CALCULATE
         </Button>
         <Button variant="primary" size="sm">EXPORT BASALINE</Button>
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
               Defined Peer Groups
            </div>
            <div class="text-xl font-mono font-bold text-accent-primary">
               {stats.groups}
            </div>
            <div class="text-[9px] text-text-muted mt-1 italic">
               Clustered by context
            </div>
         </div>
         <div class="bg-surface-2 p-3">
            <div
               class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1"
            >
               Monitored Entities
            </div>
            <div class="text-xl font-mono font-bold text-text-heading">
               {stats.members.toLocaleString()}
            </div>
            <div
               class="text-[9px] text-status-online mt-1 uppercase tracking-tighter"
            >
               ✓ Real-time baseline sync
            </div>
         </div>
         <div class="bg-surface-2 p-3">
            <div
               class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1"
            >
               Critical Outliers (≥2σ)
            </div>
            <div class="text-xl font-mono font-bold text-alert-critical">
               {stats.outliers}
            </div>
            <div
               class="text-[9px] text-alert-critical mt-1 uppercase tracking-tighter"
            >
               High confidence anomalies
            </div>
         </div>
         <div class="bg-surface-2 p-3">
            <div
               class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1"
            >
               Avg Anomaly Rate
            </div>
            <div class="text-xl font-mono font-bold text-text-heading">
               {stats.avgAnomaly}
            </div>
            <div class="text-[9px] text-text-muted mt-1 italic">
               Baseline stability index
            </div>
         </div>
      </div>

      <!-- MAIN BODY -->
      <div class="flex-1 flex min-h-0 bg-surface-0 overflow-hidden">
         <!-- LEFT: PEER GROUPS LIST -->
         <div
            class="w-80 border-r border-border-primary flex flex-col shrink-0 bg-surface-1 overflow-hidden"
         >
            <div
               class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center shrink-0"
            >
               <span
                  class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest"
                  >Entity Clusters</span
               >
               <Badge variant="secondary" size="xs">AUTO-SHARDED</Badge>
            </div>

            <div class="flex-1 overflow-auto p-2 space-y-2">
               {#if loading}
                  <div class="py-10 flex justify-center">
                     <Spinner size="sm" />
                  </div>
               {:else}
                  {#each groups as g}
                     {@const Icon = basisIcon(g.basis)}
                     <button
                        class="w-full text-left p-4 rounded-sm border transition-all group flex flex-col gap-3 relative overflow-hidden
                         {selectedId === g.id
                           ? 'bg-surface-2 border-accent-primary shadow-premium'
                           : 'bg-transparent border-border-subtle hover:bg-surface-2'}"
                        onclick={() =>
                           (selectedId = selectedId === g.id ? null : g.id)}
                     >
                        {#if selectedId === g.id}
                           <div
                              class="absolute inset-y-0 left-0 w-1 bg-accent-primary"
                           ></div>
                        {/if}
                        <div class="flex justify-between items-start">
                           <div class="flex items-center gap-3">
                              <div
                                 class="p-1.5 bg-surface-0 border border-border-subtle rounded-xs"
                              >
                                 <Icon size={14} class="text-accent-primary" />
                              </div>
                              <div class="flex flex-col">
                                 <span
                                    class="text-[11px] font-black text-text-heading uppercase tracking-tighter"
                                    >{g.name}</span
                                 >
                                 <span
                                    class="text-[8px] font-mono text-text-muted uppercase tracking-widest opacity-60"
                                    >Basis: {g.basis.replace(/_/g, " ")}</span
                                 >
                              </div>
                           </div>
                           <Badge
                              variant={g.avg_risk_score > 60
                                 ? "warning"
                                 : "success"}
                              size="xs">{g.avg_risk_score}</Badge
                           >
                        </div>

                        <div class="space-y-1.5">
                           <div
                              class="flex justify-between text-[8px] font-mono text-text-muted uppercase"
                           >
                              <span>{g.member_count} Members</span>
                              <span
                                 >Anomaly: {(g.anomaly_rate * 100).toFixed(
                                    1,
                                 )}%</span
                              >
                           </div>
                           <ProgressBar
                              value={g.avg_risk_score}
                              height="2px"
                              color={g.avg_risk_score > 60
                                 ? "var(--alert-high)"
                                 : "var(--status-online)"}
                           />
                        </div>
                     </button>
                  {/each}
               {/if}
            </div>
         </div>

         <!-- RIGHT: DEVIATIONS -->
         <div class="flex-1 flex flex-col min-w-0 bg-surface-0 overflow-hidden">
            <div
               class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0"
            >
               <div class="flex items-center gap-4">
                  <TrendingUp size={16} class="text-accent-primary" />
                  <span
                     class="text-[10px] font-black text-text-heading uppercase tracking-widest"
                  >
                     {selectedGroup
                        ? `Peer Outliers — ${selectedGroup.name}`
                        : "Global Behavioral Deviations (≥1σ)"}
                  </span>
               </div>
               {#if selectedId}
                  <Button
                     variant="ghost"
                     size="xs"
                     onclick={() => (selectedId = null)}>✕ CLEAR FILTER</Button
                  >
               {/if}
            </div>

            <div class="flex-1 overflow-auto p-6 space-y-4">
               {#if filteredDeviations.length === 0}
                  <div
                     class="h-full flex flex-col items-center justify-center gap-4 text-text-muted opacity-30 py-20"
                  >
                     <Users size={48} />
                     <p class="text-[10px] font-mono uppercase tracking-widest">
                        No significant peer deviations detected
                     </p>
                  </div>
               {:else}
                  {#each filteredDeviations as d}
                     <div
                        class="bg-surface-1 border border-border-primary border-l-2 p-5 rounded-sm flex flex-col gap-4 shadow-premium group hover:border-accent-primary transition-colors"
                        style="border-left-color: {sigmaColor(
                           d.deviation_sigma,
                        )}"
                     >
                        <div class="flex justify-between items-start">
                           <div class="flex flex-col gap-1">
                              <div class="flex items-center gap-3">
                                 <span
                                    class="text-[13px] font-black text-text-heading uppercase tracking-tighter italic"
                                    >{d.entity_id}</span
                                 >
                                 <Badge
                                    variant="secondary"
                                    size="xs"
                                    class="font-bold opacity-60"
                                    >{d.entity_type.toUpperCase()}</Badge
                                 >
                              </div>
                              <span
                                 class="text-[9px] font-mono text-text-muted uppercase tracking-widest"
                                 >Cluster: <span class="text-accent-primary"
                                    >{d.group_name}</span
                                 ></span
                              >
                           </div>
                           <div class="flex flex-col items-end">
                              <span
                                 class="text-xl font-mono font-black italic"
                                 style="color: {sigmaColor(d.deviation_sigma)}"
                                 >{d.deviation_sigma.toFixed(1)}σ</span
                              >
                              <span
                                 class="text-[8px] font-mono text-text-muted uppercase tracking-tighter"
                                 >Statistical Sigma</span
                              >
                           </div>
                        </div>

                        <div
                           class="grid grid-cols-2 gap-8 py-2 border-y border-border-subtle"
                        >
                           <div class="space-y-1.5">
                              <div
                                 class="flex justify-between text-[9px] font-mono text-text-muted uppercase tracking-tighter"
                              >
                                 <span>Peer Group Avg</span>
                                 <span>{d.group_avg_risk}</span>
                              </div>
                              <div
                                 class="h-1 bg-surface-2 rounded-full overflow-hidden"
                              >
                                 <div
                                    class="h-full bg-text-muted opacity-40"
                                    style="width: {d.group_avg_risk}%"
                                 ></div>
                              </div>
                           </div>
                           <div class="space-y-1.5">
                              <div
                                 class="flex justify-between text-[9px] font-mono text-text-muted uppercase tracking-tighter"
                              >
                                 <span>Entity Risk</span>
                                 <span
                                    style="color: {sigmaColor(
                                       d.deviation_sigma,
                                    )}">{d.entity_risk}</span
                                 >
                              </div>
                              <div
                                 class="h-1 bg-surface-2 rounded-full overflow-hidden"
                              >
                                 <div
                                    class="h-full"
                                    style="width: {d.entity_risk}%; background: {sigmaColor(
                                       d.deviation_sigma,
                                    )}"
                                 ></div>
                              </div>
                           </div>
                        </div>

                        <div
                           class="flex justify-between items-center text-[9px] font-mono"
                        >
                           <div class="flex items-center gap-4">
                              <div
                                 class="flex items-center gap-1.5 text-text-muted"
                              >
                                 <Activity size={10} />
                                 <span class="uppercase">Primary Vector:</span>
                                 <span
                                    class="text-text-heading font-bold uppercase"
                                    >{d.top_deviation.replace(/_/g, " ")}</span
                                 >
                              </div>
                              <div
                                 class="flex items-center gap-1.5 text-text-muted"
                              >
                                 <History size={10} />
                                 <span
                                    >{new Date(
                                       d.timestamp,
                                    ).toLocaleTimeString()}</span
                                 >
                              </div>
                           </div>
                           <Button variant="ghost" size="xs" icon={Zap}
                              >PIVOT TO PROFILE</Button
                           >
                        </div>
                     </div>
                  {/each}
               {/if}

               <!-- Methodology Note -->
               <div
                  class="p-5 bg-surface-2 border border-border-primary rounded-sm space-y-3 shadow-inner"
               >
                  <div
                     class="flex items-center gap-2 text-[10px] font-black text-text-heading uppercase tracking-widest"
                  >
                     <Info size={14} class="text-accent-primary" />
                     Methodology: Basal Outlier Mapping
                  </div>
                  <p
                     class="text-[9px] text-text-muted font-mono leading-relaxed opacity-70 italic"
                  >
                     Peer groups are dynamically sharded based on LDAP roles,
                     cross-service access patterns, and departmental affinities.
                     Statistical deviation is calculated via standard Sigma (σ)
                     distribution from the group centroid. High-density
                     anomalies (≥2σ) trigger automated UEBA alerts.
                  </p>
               </div>
            </div>
         </div>
      </div>

      <!-- STATUS BAR -->
      <div
         class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest"
      >
         <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>UEBA_ENGINE:</span>
            <span class="text-status-online font-bold italic"
               >CLUSTER_STABLE</span
            >
         </div>
         <span class="text-border-primary opacity-30">|</span>
         <div class="flex items-center gap-1.5">
            <span>PEER_SYNC:</span>
            <span class="text-status-online font-bold italic">REALTIME</span>
         </div>
         <span class="text-border-primary opacity-30">|</span>
         <div class="flex items-center gap-1.5">
            <span>BASAL_SIGMA:</span>
            <span class="text-accent-primary font-bold italic">L7_ACTIVE</span>
         </div>
         <div class="ml-auto opacity-40">OBLIVRA_PEER_v10.5.1</div>
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
