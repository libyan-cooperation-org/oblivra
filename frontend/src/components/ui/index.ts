/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Shared Component Library — Barrel Export
   ═══════════════════════════════════════════════════════════════
   
   Import any design-system component from a single path:
   
     import { Table, Badge, KPI, PageLayout, Panel, SearchBar } from '@components/ui';
   
   Every component wraps the .ob-* CSS classes from components.css,
   ensuring design system consistency across all pages.
   ═══════════════════════════════════════════════════════════════ */

// ── Data Display ──────────────────────────────────────────────
export { Table } from './Table';
export type { Column, SortDir } from './Table';

export { Badge, StatusDot, normalizeSeverity } from './Badge';

export { KPI, KPIGrid } from './KPI';

export { Sparkline } from './Sparkline';
export { Histogram } from './Histogram';

// ── Layout ────────────────────────────────────────────────────
export { PageLayout, Notice, CodeBlock, Progress } from './PageLayout';

export { Panel, SectionHeader } from './Panel';

export { SearchBar, Toolbar, ToolbarSpacer, TabBar } from './SearchBar';

// ── Overlays ──────────────────────────────────────────────────
export { Modal, Dropdown, FormField, FormRow } from './Modal';
export { ModalSystem, showModal } from './ModalSystem';
export type { ModalAction, ModalOptions } from './ModalSystem';

// ── State ─────────────────────────────────────────────────────
export { EmptyState } from './EmptyState';

export {
    Skeleton,
    LoadingState,
    ErrorState,
    KeyValue,
    CountBadge,
    formatRelativeTime,
    formatTimestamp,
} from './Utilities';

// ── Legacy (existing components, re-exported for convenience) ─
export { Card, Button as TacticalButton } from './TacticalComponents';

export { CommandPalette } from './CommandPalette';
export { QuickSwitcher } from './QuickSwitcher';
export { ToastContainer } from './Toast';
export { Button } from './Button';
export { Input, Select, TextArea } from './Input';
