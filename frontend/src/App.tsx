import { Component, createSignal, onMount, Show } from 'solid-js';
import { AppLayout } from '@components/layout/AppLayout';
import { initBridge } from '@core/bridge';
import { AppProvider } from '@core/store';
import { VaultGuard } from '@components/security/VaultGuard';
import { LoadingScreen } from '@components/ui/LoadingScreen';
import { ErrorScreen } from '@components/ui/ErrorScreen';
import { ToastContainer } from '@components/layout/ToastContainer';
import { useToast } from '@core/toast';
import { PanelManagerProvider } from '@components/layout/PanelManager';

const App: Component<{ children?: any }> = (props) => {
    const [ready, setReady] = createSignal(false);
    const [error, setError] = createSignal<string | null>(null);
    const { addToast } = useToast();

    onMount(async () => {
        try {
            await initBridge();

            // Hook Global Toasts into Wails Events
            if ((window as any).runtime) {
                (window as any).runtime.EventsOn('system.error', (msg: string) => {
                    addToast({ type: 'error', title: 'System Error', message: msg });
                });
                (window as any).runtime.EventsOn('system.toast', (toast: any) => {
                    addToast(toast);
                });
            }

            setReady(true);
        } catch (err) {
            setError(`${err}`);
        }
    });

    return (
        <Show
            when={ready()}
            fallback={
                <Show
                    when={!error()}
                    fallback={
                        <ErrorScreen message={error()!} />
                    }
                >
                    <LoadingScreen />
                </Show>
            }
        >
            <AppProvider>
                <VaultGuard>
                    <PanelManagerProvider>
                        <div class="app-entry-animation">
                            <AppLayout>{props.children}</AppLayout>

                            <ToastContainer />
                        </div>
                    </PanelManagerProvider>
                </VaultGuard>
            </AppProvider>
        </Show>
    );
};

export default App;
