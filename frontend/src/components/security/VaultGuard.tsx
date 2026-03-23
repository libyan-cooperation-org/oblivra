import { Component, createSignal, onMount, JSX, Show } from 'solid-js';
import { IsSetup, IsUnlocked } from '../../../wailsjs/go/services/VaultService';
import { useApp } from '../../core/store';
import { IS_BROWSER } from '../../core/context';
import { VaultSetup } from './VaultSetup';
import { VaultUnlock } from './VaultUnlock';
import '../../styles/vault-gate.css';

interface VaultGuardProps {
    children: JSX.Element;
}

export const VaultGuard: Component<VaultGuardProps> = (props) => {
    // ── Browser mode ──────────────────────────────────────────────────────────
    // The local vault is a desktop-only concept (AES-256 encrypted local store).
    // In browser mode there is no Wails runtime so window.go is undefined —
    // calling any Wails binding throws immediately. Skip the vault gate entirely;
    // server-side auth (OIDC / local login) handles authentication instead.
    if (IS_BROWSER) {
        return <>{props.children}</>;
    }

    // ── Desktop / Hybrid mode ─────────────────────────────────────────────────
    const [state, actions] = useApp();
    const [setup, setSetup] = createSignal<boolean>(true);

    const checkState = async () => {
        try {
            const [isSetup, isUnlocked] = await Promise.all([IsSetup(), IsUnlocked()]);
            setSetup(isSetup);
            actions.setVaultUnlocked(isUnlocked);
        } catch (_) {
            // Vault service unavailable — show unlock screen
        }
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
