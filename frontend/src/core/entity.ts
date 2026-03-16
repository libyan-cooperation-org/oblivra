/**
 * Entity Navigation Helpers
 *
 * Any component can pivot to an entity investigation page with:
 *
 *   import { openEntity } from '@core/entity';
 *   openEntity('host', 'web-01');
 *   openEntity('user', 'root');
 *   openEntity('ip', '10.0.0.1');
 *
 * The URL format is: /#/entity/:type/:id
 * Works with both HashRouter navigation and direct link opening.
 */

export type EntityType = 'host' | 'user' | 'ip';

/**
 * Navigate to the entity investigation page for the given entity.
 * Use this everywhere in the app to pivot to an entity — alerts, SIEM events,
 * topology nodes, UEBA profiles, etc.
 */
export function openEntity(type: EntityType, id: string): void {
    if (!id) return;
    window.location.hash = `#/entity/${type}/${encodeURIComponent(id)}`;
}

/**
 * Returns the entity page URL for use in <a href> links.
 */
export function entityHref(type: EntityType, id: string): string {
    return `#/entity/${type}/${encodeURIComponent(id)}`;
}

/**
 * A clickable entity chip — renders the entity id as a styled link.
 * Clicking navigates to the entity investigation page.
 *
 * Usage:
 *   <EntityChip type="host" id="web-01" />
 *   <EntityChip type="ip" id="10.0.0.1" />
 *   <EntityChip type="user" id="root" />
 */
import { Component, Show } from 'solid-js';

const typeIcon: Record<EntityType, string> = {
    host: '🖥',
    user: '👤',
    ip:   '🌐',
};

const typeColor: Record<EntityType, string> = {
    host: '#0099e0',
    user: '#b87fff',
    ip:   '#f58b00',
};

export const EntityChip: Component<{
    type: EntityType;
    id: string;
    label?: string;
    style?: Record<string, string>;
}> = (props) => {
    if (!props.id) return null;

    return (
        <a
            href={entityHref(props.type, props.id)}
            title={`Investigate ${props.type}: ${props.id}`}
            style={{
                display: 'inline-flex', 'align-items': 'center', gap: '4px',
                padding: '1px 7px', 'border-radius': '10px',
                'font-family': 'var(--font-mono)', 'font-size': '11px', 'font-weight': '600',
                color: typeColor[props.type],
                background: `${typeColor[props.type]}14`,
                border: `1px solid ${typeColor[props.type]}30`,
                'text-decoration': 'none', cursor: 'pointer',
                transition: 'all 0.15s',
                ...(props.style ?? {}),
            }}
            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = `${typeColor[props.type]}28`; }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = `${typeColor[props.type]}14`; }}
        >
            <span style={{ 'font-size': '10px' }}>{typeIcon[props.type]}</span>
            {props.label ?? props.id}
        </a>
    );
};
