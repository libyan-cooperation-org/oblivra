/**
 * investigation.svelte.ts — global investigation context.
 *
 * Phase 31 SOC redesign. The "investigation panel" is a slide-out
 * drawer that opens whenever an operator clicks an entity (host,
 * user, IP, process, hash, domain). It is the platform's
 * single context-expansion surface.
 *
 * Why a global store rather than per-page panels:
 *   - The panel needs to be reachable from every page (drill-down
 *     anywhere). A single instance mounted at the App root + a global
 *     store of the active entity is cheaper than re-mounting one in
 *     every route.
 *   - History stack: opening entity B from inside entity A's panel
 *     pushes B onto a history list so the operator can step back
 *     ("breadcrumbs of investigations").
 *   - Reactive: any component dispatching `investigationStore.open(...)`
 *     immediately surfaces the panel without prop-threading.
 */

export type EntityType = 'host' | 'user' | 'ip' | 'process' | 'hash' | 'domain' | 'alert';

export interface EntityRef {
  type: EntityType;
  id: string;
  label?: string;        // optional display string (defaults to id)
  context?: Record<string, any>; // free-form metadata (severity, host, etc.)
}

class InvestigationStore {
  /** The entity currently shown in the panel. null = panel closed. */
  active = $state<EntityRef | null>(null);

  /**
   * Investigation history — operators dig from one entity to another
   * (alert → host → user → ip), and we want to give them a back
   * button. Most recent at the END of the array (it's a stack).
   */
  history = $state<EntityRef[]>([]);

  /** Whether the side panel is rendered. Mirror of `active != null`
   *  so consumers can drive a transition without losing the entity. */
  open = $state(false);

  /**
   * Open the panel on the given entity. If a different entity is
   * already showing, the previous one is pushed onto the history
   * stack first so Back works.
   */
  openEntity(ref: EntityRef) {
    if (!ref || !ref.id) return;
    // If clicking the same entity that's already open, just bring
    // the panel forward.
    if (this.active && this.active.type === ref.type && this.active.id === ref.id) {
      this.open = true;
      return;
    }
    if (this.active) {
      this.history = [...this.history, this.active];
      // Cap history depth so a long chain doesn't grow unbounded.
      if (this.history.length > 20) {
        this.history = this.history.slice(-20);
      }
    }
    this.active = ref;
    this.open = true;
  }

  /**
   * Pop the most recent entity off the history stack and re-open it
   * as the active investigation.
   */
  back(): boolean {
    if (this.history.length === 0) {
      this.close();
      return false;
    }
    const prev = this.history[this.history.length - 1];
    this.history = this.history.slice(0, -1);
    this.active = prev;
    this.open = true;
    return true;
  }

  close() {
    this.open = false;
    // Defer wiping `active` until the panel's transition completes
    // so the slide-out animation has something to render. Callers
    // that don't care about animation can call clear() directly.
    setTimeout(() => {
      if (!this.open) {
        this.active = null;
        this.history = [];
      }
    }, 250);
  }

  clear() {
    this.active = null;
    this.history = [];
    this.open = false;
  }
}

export const investigationStore = new InvestigationStore();
