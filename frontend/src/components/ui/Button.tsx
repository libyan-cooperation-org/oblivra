import { Component, JSX, splitProps } from 'solid-js';

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Button — Standard interaction component
   Wraps .ob-btn and its variants from components.css
   ═══════════════════════════════════════════════════════════════ */

interface ButtonProps extends JSX.ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: 'primary' | 'blue' | 'danger' | 'ghost' | 'default';
    size?: 'sm' | 'md' | 'lg' | 'icon';
    loading?: boolean;
}

export const Button: Component<ButtonProps> = (props) => {
    const [local, rest] = splitProps(props, ['variant', 'size', 'loading', 'class', 'children', 'disabled']);

    const getVariantClass = () => {
        switch (local.variant) {
            case 'primary': return 'ob-btn-primary';
            case 'blue':    return 'ob-btn-blue';
            case 'danger':  return 'ob-btn-danger';
            case 'ghost':   return 'ob-btn-ghost';
            default:        return '';
        }
    };

    const getSizeClass = () => {
        switch (local.size) {
            case 'sm':   return 'ob-btn-sm';
            case 'lg':   return 'ob-btn-lg';
            case 'icon': return 'ob-btn-icon';
            default:     return '';
        }
    };

    return (
        <button
            class={`ob-btn ${getVariantClass()} ${getSizeClass()} ${local.class || ''}`}
            disabled={local.disabled || local.loading}
            {...rest}
        >
            {local.loading ? <span class="spinner" style="width: 12px; height: 12px; border-width: 1px;" /> : local.children}
        </button>
    );
};
