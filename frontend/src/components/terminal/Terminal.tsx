import {
    Component,
    onMount,
    onCleanup,
} from 'solid-js';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';
import '@xterm/xterm/css/xterm.css';

interface TerminalProps {
    sessionId: string;
    onData?: (data: string) => void;
    onResize?: (cols: number, rows: number) => void;
}

const THEME = {
    background: '#0d1117',
    foreground: '#c9d1d9',
    cursor: '#58a6ff',
    cursorAccent: '#0d1117',
    selectionBackground: '#264f78',
    selectionForeground: '#ffffff',
    black: '#484f58',
    red: '#ff7b72',
    green: '#3fb950',
    yellow: '#d29922',
    blue: '#58a6ff',
    magenta: '#bc8cff',
    cyan: '#39c5cf',
    white: '#b1bac4',
    brightBlack: '#6e7681',
    brightRed: '#ffa198',
    brightGreen: '#56d364',
    brightYellow: '#e3b341',
    brightBlue: '#79c0ff',
    brightMagenta: '#d2a8ff',
    brightCyan: '#56d4dd',
    brightWhite: '#f0f6fc',
};

export const TerminalView: Component<TerminalProps> = (props) => {
    let containerRef: HTMLDivElement | undefined;
    let terminal: XTerm | undefined;
    let fitAddon: FitAddon | undefined;
    let resizeObserver: ResizeObserver | undefined;

    onMount(() => {
        if (!containerRef) return;

        // Create xterm instance
        terminal = new XTerm({
            fontSize: 14,
            fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
            theme: THEME,
            cursorBlink: true,
            cursorStyle: 'block',
            scrollback: 10000,
            convertEol: true,
            lineHeight: 1.2,
        });

        fitAddon = new FitAddon();
        terminal.loadAddon(fitAddon);
        terminal.loadAddon(new WebLinksAddon((_e, uri) => window.open(uri, '_blank')));

        terminal.open(containerRef);

        try {
            fitAddon.fit();
        } catch (_) { /* container may not be visible yet */ }

        // User typing → send to backend
        terminal.onData((data) => {
            props.onData?.(data);
        });

        // Terminal resize → send to backend
        terminal.onResize(({ cols, rows }) => {
            props.onResize?.(cols, rows);
        });

        // Listen for output from backend (base64 encoded)
        EventsOn(`terminal - output - ${props.sessionId}`, (data: string) => {
            if (terminal && typeof data === 'string') {
                try {
                    const binaryString = atob(data);
                    const bytes = new Uint8Array(binaryString.length);
                    for (let i = 0; i < binaryString.length; i++) {
                        bytes[i] = binaryString.charCodeAt(i);
                    }
                    terminal.write(bytes);
                } catch {
                    terminal.write(data);
                }
            }
        });

        // Also listen for SSH session output (different event name format)
        EventsOn(`session.output.${props.sessionId}`, (data: string) => {
            if (terminal && typeof data === 'string') {
                try {
                    const binaryString = atob(data);
                    const bytes = new Uint8Array(binaryString.length);
                    for (let i = 0; i < binaryString.length; i++) {
                        bytes[i] = binaryString.charCodeAt(i);
                    }
                    terminal.write(bytes);
                } catch {
                    terminal.write(data);
                }
            }
        });

        // Auto-fit when container resizes
        resizeObserver = new ResizeObserver(() => {
            requestAnimationFrame(() => {
                if (containerRef && containerRef.clientWidth > 0 && containerRef.clientHeight > 0) {
                    try { fitAddon?.fit(); } catch (_) { }
                }
            });
        });
        resizeObserver.observe(containerRef);

        terminal.focus();
    });

    onCleanup(() => {
        if (props.sessionId) {
            EventsOff(`terminal - output - ${props.sessionId}`);
            EventsOff(`session.output.${props.sessionId}`);
        }
        resizeObserver?.disconnect();
        terminal?.dispose();
    });

    return (
        <div
            ref={containerRef}
            class="terminal-container"
            style={{ width: '100%', height: '100%' }}
        />
    );
};
