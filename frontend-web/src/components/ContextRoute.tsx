/**
 * ContextRoute.tsx — Context-aware route guard (Phase 0.5)
 *
 * Wraps a route component and ensures it only renders in the correct
 * execution context. If a user navigates to a mismatched context route,
 * they are silently redirected to the dashboard root.
 *
 * Usage:
 *   <ContextRoute context="web" component={FleetManagement} />
 *   <ContextRoute context="desktop" component={Workspace} />
 *   <ContextRoute context="any" component={Dashboard} />
 */

import { type Component, Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { getAppContext, type AppContext } from '../context';

interface ContextRouteProps {
  /** The required context for this route. Use 'any' for hybrid pages. */
  context: AppContext | 'any';
  /** The page component to render if context matches. */
  component: Component;
}

export default function ContextRoute(props: ContextRouteProps) {
  const navigate = useNavigate();
  const currentContext = getAppContext();

  const allowed = props.context === 'any' || props.context === currentContext;

  if (!allowed) {
    // Redirect immediately — this route is not available in this context.
    navigate('/', { replace: true });
  }

  return (
    <Show when={allowed}>
      <props.component />
    </Show>
  );
}
