import { Component, createSignal, onMount, JSX, Show } from 'solid-js';
import { IsSetup, IsUnlocked } from '../../../wailsjs/go/app/VaultService';
import { VaultSetup } from './VaultSetup';
import { VaultUnlock } from './VaultUnlock';
import '../../styles/vault-gate.css';

interface VaultGuardProps {
    children: JSX.Element;
}

export const VaultGuard: Component<VaultGuardProps> = (props) => {
    const [setup, setSetup] = createSignal<boolean | null>(null);
    const [unlocked, setUnlocked] = createSignal(false);

    const checkState = async () => {
        const isSetup = await IsSetup();
        const isUnlocked = await IsUnlocked();
        setSetup(isSetup);
        setUnlocked(isUnlocked);
    };

    onMount(() => {
        checkState();
    });

    return (
        <Show when={setup() !== null} fallback={<div class="loading-screen"><div class="spinner" /></div>}>
            <Show when={unlocked()} fallback={
                <Show when={setup()} fallback={<VaultSetup onComplete={checkState} />}>
                    <VaultUnlock onUnlock={checkState} />
                </Show>
            }>
                {props.children}
            </Show>
        </Show>
    );
};
