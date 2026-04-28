/**
 * OBLIVRA — Svelte 5 Entry Point
 *
 * Mounts the root App.svelte component.
 * Replaces the SolidJS render() + HashRouter from index.tsx.
 */
import './app.css';
import { mount } from 'svelte';
import App from './App.svelte';

const target = document.getElementById('app');
if (!target) throw new Error('Mount target #app not found');

try {
    // We don't need the mount return value — Svelte 5 attaches the
    // component to `target` directly. The previous `const app = ...`
    // tripped a no-unused-vars warning under tsc --noEmit.
    mount(App, { target });
} catch (e: any) {
    console.error("Mount error:", e);
    document.body.innerHTML = `<div style="color:red; padding:20px; font-family:sans-serif;">
        <h2>Frontend Initialization Error</h2>
        <pre>${e.message || e}</pre>
        <pre>${e.stack || ''}</pre>
    </div>`;
}

window.addEventListener('error', (e) => {
    document.body.innerHTML = `<div style="color:red; padding:20px; font-family:sans-serif;">
        <h2>Global Error</h2>
        <pre>${e.message || e}</pre>
        <pre>${e.error?.stack || ''}</pre>
    </div>`;
});
window.addEventListener('unhandledrejection', (e) => {
    document.body.innerHTML = `<div style="color:red; padding:20px; font-family:sans-serif;">
        <h2>Unhandled Promise Rejection</h2>
        <pre>${e.reason?.message || e.reason || e}</pre>
        <pre>${e.reason?.stack || ''}</pre>
    </div>`;
});
