import type { Component } from 'solid-js';

interface AppLogoProps {
    size?: number;
    class?: string;
}

export const AppLogo: Component<AppLogoProps> = (props) => {
    const size = props.size || 32;

    return (
        <svg
            width={size}
            height={size}
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            class={props.class}
        >
            <rect x="12" y="2" width="10" height="10" rx="2" fill="var(--accent-primary)" />
            <rect x="2" y="12" width="10" height="10" rx="2" fill="#FFFFFF" fill-opacity="0.8" />
            <path d="M7 7L17 17" stroke="var(--accent-primary)" stroke-width="2.5" stroke-linecap="round" />
        </svg>
    );
};
