# OBLI RA — Pha e Tracker (Build Roadmap)

> **What thi  file i **: the  ingle  ource of truth for the chronological build narrati e — e ery pha e, it  goal, and current  tatu .
>
> **What li e  el ewhere**:
> - [`HARDENING.md`](HARDENING.md) —  ecurity finding , fi e , po tmortem , and hardening gate 
> - [`ROADMAP.md`](ROADMAP.md) — long-term  i ion (C PM, K8 ,  uln mgmt)
> - [`RE EARCH.md`](RE EARCH.md) — DARPA/N A-grade re earch item 
> - [`BU INE  .md`](BU INE  .md) — certification , legal, GTM
> - [`FUTURE.md`](FUTURE.md) — cro  -cutting (chao  engineering, deception, i18n)
> - [` TRATEGY.md`]( TRATEGY.md) — Pha e 22  trategic rationale

**La t audited**: 2026-04-29
**Current po itioning**:  o ereign, log-dri en  ecurity and foren ic  platform.

---

##  tatu  Tier 
- `[x]` = **Production-Ready** (hardened, documented,  oak-te ted)
- `[s]` = ** alidated** (functionally correct, te ted under load)
- `[V]` = ** caffolded** (code e i t  and compile )
- `[ ]` = Not  tarted

---

## Platform Architecture — Golden Rule

> **De ktop =  en iti e + Local + Operator Action **
> **Web =  hared +  calable + Multi-u er**

### De ktop (Wail  App) — Mu t li e here
-  ault &  ecret  management (AE -256, O  keychain, FIDO2/YubiKey)
- Local/offline operation  and air-gap mode
- Agent build/ igning and certificate generation
- Local foren ic  e idence handling

### Web (Brow er UI) — Mu t li e here
- Fleet-wide  IEM  earch and ob er ability
- Alerting, e calation, and notification 
- Central detection rule management
- Threat hunting and in e tigation tool 
- Fleet management and agent o er ight
- Multi-tenancy and RBAC

### Hybrid (Both)
-  earch, detection rule , da hboard , and alert  ( coped differently per  urface)

**Ne er on Web**:  ault ma ter key, agent  igning key , local file y tem acce  .

---

## Core Platform Feature  (Current  tate)

###  ecurity &  ault
- [ ] AE -256 encrypted  ault with O  keychain integration
- [ ] FIDO2 / YubiKey  upport
- [ ] TL  certificate generation
- [ ]  ecurity key management UI

###  torage, Inge tion &  earch
- [ ] BadgerDB (hot) + Ble e ( earch) + Parquet (warm)
- [ ]  3-compatible cold tier ( tub interface — full implementation in Pha e 22.3)
- [ ] Cra h- afe Write-Ahead Log (WAL)
- [ ]  y log, J ON, CEF, Window  E T  par er  (nati e inge t); Linu  journald  ia agent backfill
- [ ] High-throughput inge tion pipeline (benchmarked;  u tained-load  oak te t pending)
- [ ] OQL (Obli ra Query Language) with pipe  ynta 
- [ ]  torage tiering migrator (Hot → Warm → Cold)

### Detection & Analytic 
- [ ]  igma rule engine + tran piler (82+ rule )
- [ ] MITRE ATT&CK mapping and heatmap
- [ ] Correlation and multi- tage fu ion engine
- [ ] UEBA (beha ioral ba eline  + peer group analy i )
- [ ] NDR (NetFlow, DN  tunneling, JA3)
- [ ] Ran omware beha ioral detection (entropy-ba ed — detection only; re pon e action  remo ed Pha e 36)
- [ ] Ri k-ba ed alerting

### Foren ic  & Integrity
- [ ] Merkle-tree chained audit logging
- [ ] RFC 3161 time tamping
- [ ] E idence locker with chain-of-cu tody
- [ ] Temporal integrity (clock-drift  alidation, time tamp guard ) — DHCP-aware entity re olution pending
- [ ] Centralized DLP redactor

### Agent Framework
- [ ] Lightweight Go agent with HTTP + mTL  + zlib tran port
- [ ] File tailing, Window  E ent Log, journald, metric , FIM
- [ ] Offline buffering (local WAL) + edge filtering
- [ ] Agentle   collector  (WMI,  NMP, RE T polling)
- [ ] Encrypted config  torage, multi-output routing, watchdog

### Enterpri e
- [ ] Multi-tenancy with  trong data i olation
- [ ] OIDC/ AML + MFA + granular RBAC
- [ ] Audit log e idence pack (E idenceLocker + Merkle audit e i t; formal e port endpoint pending Pha e 38). Pair with e ternal compliance tooling (Drata/ anta/Tugboat) for atte tation.

### U  & Producti ity
- [ ] Hybrid de ktop/web architecture with conte t guard 
- [ ] In e tigation-fir t UI (Ho tDetail, In e tigationPanel, EntityLink)
- [ ] Multi-monitor pop-out window  + work pace  a e/re tore
- [ ] Command palette, notification center, unified time range picker

---

## Pha e Hi tory (Conden ed)

### Pha e 0–0.5: Foundation &  tabilization ✅
Core  er ice regi try, de ktop/web conte t  eparation, web M P, acce  ibility, and architectural hardening.

### Pha e 1: Core  torage + Inge tion +  earch ✅
BadgerDB, Ble e, Parquet archi al, WAL, high-performance inge tion pipeline, and OQL foundation.

### Pha e 2: Alerting + RE T API ✅
Detection rule , alerting with e calation, RE T API with RBAC, real-time  treaming.

### Pha e 3: Threat Intel + Enrichment ✅
 TI /TA II, IOC matching, GeoIP, DN , a  et mapping, ad anced log par er .

### Pha e 4: Detection Engineering + MITRE ✅
 igma rule , correlation engine, MITRE heatmap, rule te ting framework.

### Pha e 5–6:  tability & Audit E idence ✅
Memory bound , incident lifecycle, Merkle audit logging, e idence locker, RFC 3161 time tamping.

### Pha e 7: Agent Framework ✅
Full agent + agentle   collection, encrypted config, multi-output, local detection rule .

### Pha e 10: UEBA & Beha ioral Analytic  ✅
U er/entity ba eline , I olation Fore t, peer group analy i , multi- tage fu ion.

### Pha e 11: Network Detection & Re pon e (NDR) ✅
NetFlow/IPFI , DN  analy i , JA3, lateral mo ement detection.

### Pha e 12: Enterpri e Capabilitie  ✅
Multi-tenancy i olation, HA foundation , identity & RBAC, data lifecycle management.

### Pha e 15:  o ereignty ✅
Offline update ,  ignature  erification, zero-internet mode.

### Pha e 17–21: Detection Quality &  caling ✅
Full  igma  upport, OQL enhancement , partitioned pipeline, query limit , rule routing.

### Pha e 22: Productization & Reliability ✅
Multi-tenant i olation, reliability engineering (chao  te ting, reconnect, degradation handling),  torage tiering foundation.

### Pha e 23: De ktop U  & Windowing ✅
Framele   window chrome, multi-monitor pop-out , work pace  a e/re tore, notification center.

### Pha e 27: Ad anced Platform Mechanic  ✅
OQL `par e` command , temporal entity re olution, centralized DLP, Raft control plane impro ement .

### Pha e 30–31: In e tigation-Fir t UI ✅
Ho tDetail page, global In e tigationPanel, EntityLink, 6-domain na igation ( IEM / IN E T / RE POND / FLEET / GO ERN / ADMIN), Acti ityFeed, foren ic backfill.

### Pha e 32:  hell  ub y tem Remo al ✅
Interacti e terminal/  H/ FTP  ub y tem remo ed from UI (backend librarie  retained for non-terminal u e).

### Pha e 35:  torage Tiering Wiring ✅
Hot/Warm/Cold migrator fully wired, RE T API, da hboard, and ob er ability.

### Pha e 36:  cope Cut — Log-Dri en Foren ic  Platform ✅
**Major repo itioning**. Remo ed:  OAR, incident re pon e playbook , ran omware re pon e action , di k/memory imaging, AI a  i tant, plugin framework, e ternal ob er ability  tack (Prometheu /Grafana/Tempo), and compliance YAML pack  + e aluator + PDF/HTML report generator (36. ).
Focu  narrowed to high-integrity log collection, detection, UEBA, NDR, and foren ic e idence handling.

---

## Current Open Work (Po t-Pha e 36)

### Pha e 37: Log Foren ic  Core (In Progre  )
- [ ] Log gap and anti-foren ic acti ity detection (within OBLI RA'  own e idence chain)
- [ ] Enhanced E T  and journald deep par ing (top 30 E T  e ent ID  + top 20 journald unit ; e panded per-quarter)
- [ ] Unified foren ic timeline  iew with  e erity rail 
- [ ] Ba ic E idence Package e port (e ent  + timeline + Merkle proof)
- [ ] Foren ic  earch template  in OQL
- [ ] Tru t-tier (TE/ E/BE) enforcement in  earch/e port pipeline 

### Pha e 38: Court Admi  ibility Layer
- [ ] Full Foren ic E idence Package (PDF/HTML +  ignature  +  erification in truction )
- [ ] E idence  erification Portal (offline CLI  erifier)
- [ ] WORM mode for warm/cold tier  (Window  ReF  integrity  tream  + Linu  `chattr +i`)
- [ ] Templated narrati e builder for in e tigation report  (no LLM)
- [ ] E panded chain-of-cu tody UI in E idence ault
- [ ] Legal re iew gate before claiming admi  ibility

### Pha e 39: Ad anced Log Foren ic 
- [ ] Proce   lineage and command-line recon truction from log 
- [ ] Authentication/ e  ion recon truction
- [ ] Entity Foren ic Profile tab (Ho t/U er/IP)
- [ ] Tampered/deleted log indicator  (within OBLI RA'  own e idence chain — not ho t-file y tem  canning)
- [ ] E pert witne   e port package

### Pha e 22.3:  torage Tiering Poli h (Carry-o er)
- [ ] Inge t pipeline write  through HotTier interface
- [ ] Cold tier  3  upport (build-tagged)
- [ ] Per-tenant retention o erride 
- [ ] Cro  -tier integrity  erification

### Immediate Hygiene
- [ ] Final Pha e 36 cleanup (dead F M path , Wail  binding  regeneration, doc  refre h)
- [ ] Update `README.md` and `FEATURE .md` with new log foren ic  po itioning
- [ ] Create `doc /operator/log-foren ic .md`
- [ ] **Pha e 36.7** — backend re pon e-action chain cleanup (~250 LOC acro   MCP / agent / agent_ er er / agent_ er ice / api_ er ice / re pon e_action  / re t_fu ion_peer + frontend ToggleQuarantine/KillProce   remo al).  ee `HARDENING.md`.
- [ ] **Pha e 36.8** —  chema migration  35: dropped 4 table  (`tunnel `, `  o_pro ider `, `graph_node `, `graph_edge `) + 5 inde e  + `audit_log .u er_agent` column. 5 table  originally flagged were  erified li e and **kept** (`agent_*`, `dhcp_lea e_log`, `compliance_package `, `login_lockout `, `d r_reque t `, `e idence_time tamp `).
- [ ] **Pha e 36.9** — frontend orphan  weep: deleted 2  tore  (compliance, playbook), 5 page  (Ta k Page, E calationCenter, PurpleTeam, Re pon eReplay, ConfigRi k) + emergency follow-on ( imulationPanel, AlertDa hboard Re ol e),  tripped broken action  from AlertManagement + ThreatGraph, remo ed orphan Wail  binding .
- [ ] **Pha e 36.10** — UI regi try con olidation:  tripped  tale entrie  acro   CommandRail / CommandPalette / na -config.t  / conte t.t . (Architectural fi  — na -config.t  a   ingle  ource of truth + CI lint — deferred.)
- [ ] **Pha e 36.11** — Po t-`wail 3 build` orphan  weep: deleted 8 unreferenced page  + 6 unreferenced component  + barrel-e port cleanup; fi ed 3 a11y warning  (TenantFa t witcher, Modal). Build now produce  zero warning .
- [ ] **Pha e 36.12** — Backend Wail -binding triage: 26  er ice  unregi tered from `main_gui.go` (Wail -dead but Go-li e — kept running in container.go for RE T/e entbu  con umer ). Binding  will  top auto-regenerating.  ee `HARDENING.md` for the per- er ice  erification.
- [ ] Create `doc /operator/log-foren ic .md`

---

##  trategic End  tate

OBLI RA i  not a traditional  IEM.

It i  a **cryptographically  erifiable foren ic log  y tem** capable of recon tructing logged  y tem acti ity acro   time,  torage tier , and organizational boundarie  — with e plicit gap marker  where telemetry wa  una ailable or tampered.

---

## Ne t Mile tone — Beta-1

-  table high-integrity inge tion pipeline (with documented  u tained-load  oak te t)
-  erified detection and correlation engine
- Foren ic timeline recon truction with e plicit gap marker 
- Functional e idence e port with cryptographic proof  and offline  erifier
-  table multi-tenant i olation

---

**La t updated**: 2026-04-29
