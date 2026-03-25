import { Component, onMount, onCleanup, createEffect } from 'solid-js';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { subscribe } from '@core/bridge';
import '@xterm/xterm/css/xterm.css';

interface TerminalProps {
    sessionId: string;
    onData?: (data: string) => void;
    onResize?: (cols: number, rows: number) => void;
    /** When true the terminal is the visible/active tab — triggers fit on show */
    active?: boolean;
}

const THEME = {
    background: '#0d0e10',
    foreground: '#d4d5d8',
    cursor: '#0099e0',
    cursorAccent: '#0d0e10',
    selectionBackground: 'rgba(0,153,224,0.25)',
    selectionForeground: '#ffffff',
    black: '#3a3d44',
    red: '#e04040',
    green: '#5cc05c',
    yellow: '#f5c518',
    blue: '#0099e0',
    magenta: '#b87fff',
    cyan: '#00c8d4',
    white: '#d4d5d8',
    brightBlack: '#6b6e76',
    brightRed: '#ff6b6b',
    brightGreen: '#7dda7d',
    brightYellow: '#ffd24d',
    brightBlue: '#33b8ff',
    brightMagenta: '#cc99ff',
    brightCyan: '#4de8f0',
    brightWhite: '#f0f1f3',
};

export const TerminalView: Component<TerminalProps> = (props) => {
    let containerRef: HTMLDivElement | undefined;
    let terminal: XTerm | undefined;
    let fitAddon: FitAddon | undefined;
    let resizeObserver: ResizeObserver | undefined;
    let initialized = false;

    const tryFit = () => {
        if (!containerRef || !fitAddon) return;
        if (containerRef.clientWidth > 0 && containerRef.clientHeight > 0) {
            try { fitAddon.fit(); } catch (_) {}
        }
    };

    // Re-fit whenever this terminal becomes the active one
    createEffect(() => {
        if (props.active) {
            // Defer a frame so the container is fully visible
            requestAnimationFrame(() => requestAnimationFrame(() => tryFit()));
        }
    });

    onMount(() => {
        if (!containerRef || initialized) return;
        initialized = true;

        terminal = new XTerm({
            fontSize: 13,
            fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
            theme: THEME,
            cursorBlink: true,
            cursorStyle: 'block',
            scrollback: 10000,
            convertEol: true,
            lineHeight: 1.25,
            allowProposedApi: false,
            // P2: Clipboard OSC 52 integration
            rightClickSelectsWord: true,
        });

        fitAddon = new FitAddon();
        terminal.loadAddon(fitAddon);
        terminal.loadAddon(new WebLinksAddon((_e, uri) => window.open(uri, '_blank')));

        terminal.open(containerRef);

        // Fit immediately and again after a short delay to handle layout settling
        tryFit();
        setTimeout(tryFit, 50);
        setTimeout(tryFit, 200);

        terminal.onData((data) => props.onData?.(data));
        terminal.onResize(({ cols, rows }) => props.onResize?.(cols, rows));

        // P2: Clipboard — auto-copy selection to clipboard (Termius-style)
        terminal.onSelectionChange(() => {
            const sel = terminal?.getSelection();
            if (sel && sel.length > 0) {
                navigator.clipboard.writeText(sel).catch(() => {});
            }
        });

        // P2: Clipboard — right-click paste from clipboard
        containerRef?.addEventListener('contextmenu', async (e) => {
            e.preventDefault();
            try {
                const text = await navigator.clipboard.readText();
                if (text && terminal) {
                    props.onData?.(text);
                }
            } catch {}
        });

        // Local PTY output
        const unsubPty = subscribe(`terminal-output-${props.sessionId}`, (data: string) => {
            if (!terminal || typeof data !== 'string') return;
            try {
                const bin = atob(data);
                const bytes = new Uint8Array(bin.length);
                for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
                terminal.write(bytes);
            } catch { terminal.write(data); }
        });

        // SSH session output
        const unsubSsh = subscribe(`session.output.${props.sessionId}`, (data: string) => {
            if (!terminal || typeof data !== 'string') return;
            try {
                const bin = atob(data);
                const bytes = new Uint8Array(bin.length);
                for (let i = 0; i < bin.length; i++) bytes[i] = bin.charCodeAt(i);
                terminal.write(bytes);
            } catch { terminal.write(data); }
        });
        onCleanup(() => { unsubPty(); unsubSsh(); });

        // Watch container for size changes
        resizeObserver = new ResizeObserver(() => {
            requestAnimationFrame(tryFit);
        });
        resizeObserver.observe(containerRef);

        // DraggablePanel resize event
        const onPanelResize = () => requestAnimationFrame(tryFit);
        window.addEventListener('sov:panel:resize', onPanelResize);
        onCleanup(() => window.removeEventListener('sov:panel:resize', onPanelResize));

        terminal.focus();
    });

    onCleanup(() => {
        EventsOff(`terminal-output-${props.sessionId}`);
        EventsOff(`session.output.${props.sessionId}`);
        resizeObserver?.disconnect();
        terminal?.dispose();
        initialized = false;
    });

    return (
        <div
            ref={containerRef}
            style={{ width: '100%', height: '100%', padding: '0', background: '#0d0e10' }}
        />
    );
};
