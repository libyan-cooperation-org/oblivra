import { Component, createSignal, onMount, JSX, Show } from 'solid-js';
import { IsSetup, IsUnlocked } from '../../../wailsjs/go/app/VaultService';
import { useApp } from '../../core/store';
import { VaultSetup } from './VaultSetup';
import { VaultUnlock } from './VaultUnlock';
import '../../styles/vault-gate.css';

interface VaultGuardProps {
    children: JSX.Element;
}

export const VaultGuard: Component<VaultGuardProps> = (props) => {
    const [state, actions] = useApp();
    const [setup, setSetup] = createSignal<boolean>(true);

    const checkState = async () => {
        try {
            const [isSetup, isUnlocked] = await Promise.all([IsSetup(), IsUnlocked()]);
            setSetup(isSetup);
            actions.setVaultUnlocked(isUnlocked);
        } catch (_) {}
    };

    onMount(() => { checkState(); });

    return (
        <Show
            when={state.vaultUnlocked}
            fallback={
                <Show
                    when={setup()}
                    fallback={<VaultSetup onComplete={checkState} />}
                >
                    <VaultUnlock onUnlock={checkState} />
                </Show>
            }
        >
            {props.children}
        </Show>
    );
};
