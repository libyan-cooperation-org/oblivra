import type { JSX } from 'solid-js';

export type Severity = 'info' | 'low' | 'medium' | 'high' | 'critical';

interface SeverityIconProps {
  severity: Severity;
  size?: number;
  class?: string;
}

export default function SeverityIcon(props: SeverityIconProps): JSX.Element {
  const size = props.size || 16;
  
  const getPath = () => {
    switch (props.severity) {
      case 'info':
        // Circle with 'i'
        return (
          <g fill="currentColor">
            <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/>
            <rect x="11" y="10" width="2" height="7"/>
            <circle cx="12" cy="7" r="1.5"/>
          </g>
        );
      case 'low':
        // Downward Triangle / inverted triangle
        return <path d="M12 21L2 5H22L12 21Z" fill="currentColor" />;
      case 'medium':
        // Square / Diamond
        return <rect x="4" y="4" width="16" height="16" fill="currentColor" />;
      case 'high':
        // Upward Triangle / Caution
        return <path d="M12 3L2 21H22L12 3Z" fill="currentColor" />;
      case 'critical':
        // Octagon / Hexagon / Diamond shape for high alert
        return <path d="M12 2L22 12L12 22L2 12L12 2Z" fill="currentColor" />;
      default:
        return null;
    }
  };

  const getColorClass = () => {
    switch (props.severity) {
      case 'info':     return 'text-[var(--alert-info)]';
      case 'low':      return 'text-[var(--alert-low)]';
      case 'medium':   return 'text-[var(--alert-medium)]';
      case 'high':     return 'text-[var(--alert-high)]';
      case 'critical': return 'text-[var(--alert-critical)]';
      default:         return 'text-zinc-500';
    }
  };

  return (
    <svg 
      viewBox="0 0 24 24" 
      width={size} 
      height={size} 
      class={`${getColorClass()} ${props.class || ''}`}
      aria-hidden="true"
    >
      {getPath()}
    </svg>
  );
}
