import { Component, createSignal, onMount, onCleanup, Show } from 'solid-js';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { GetRecordingFrames, GetRecordingMeta } from '../../../wailsjs/go/services/RecordingService';

import '@xterm/xterm/css/xterm.css';

interface RecordingPlayerProps {
    recordingId: string;
    onClose: () => void;
}

interface TerminalFrame {
    timestamp: number;
    type: string;
    data: string;
}

export const RecordingPlayer: Component<RecordingPlayerProps> = (props) => {
    let terminalElement: HTMLDivElement | undefined;
    const [loading, setLoading] = createSignal(true);
    const [progress, setProgress] = createSignal(0);
    const [isPlaying, setIsPlaying] = createSignal(false);
    const [currentTime, setCurrentTime] = createSignal(0);
    const [duration, setDuration] = createSignal(0);

    let term: Terminal;
    let fitAddon: FitAddon;
    let frames: TerminalFrame[] = [];
    let playbackInterval: ReturnType<typeof setInterval> | undefined;

    onMount(async () => {
        try {
            const meta = await GetRecordingMeta(props.recordingId);
            setDuration(meta.duration || 0);

            term = new Terminal({
                cursorBlink: false,
                fontSize: 13,
                fontFamily: 'JetBrains Mono, monospace',
                theme: {
                    background: '#0a0a0c',
                    foreground: '#e0e0e0',
                },
                cols: meta.cols || 80,
                rows: meta.rows || 24,
            });

            fitAddon = new FitAddon();
            term.loadAddon(fitAddon);
            term.open(terminalElement!);
            fitAddon.fit();

            frames = (await GetRecordingFrames(props.recordingId)) as unknown as TerminalFrame[];
            setLoading(false);

            // Initial render of first frame if available
            if (frames.length > 0) {
                renderUpTo(0);
            }
        } catch (e: unknown) {
            console.error('Failed to load recording:', e);
            setLoading(false);
        }
    });

    onCleanup(() => {
        if (playbackInterval) clearInterval(playbackInterval);
        term?.dispose();
    });

    const renderUpTo = (time: number) => {
        term.reset();
        for (const frame of frames) {
            if (frame.timestamp <= time) {
                if (frame.type === 'o') {
                    term.write(frame.data);
                }
            } else {
                break;
            }
        }
        setCurrentTime(time);
        setProgress((time / duration()) * 100);
    };

    const togglePlayback = () => {
        if (isPlaying()) {
            clearInterval(playbackInterval);
            setIsPlaying(false);
        } else {
            setIsPlaying(true);
            const startTime = Date.now() - (currentTime() * 1000);

            playbackInterval = setInterval(() => {
                const elapsed = (Date.now() - startTime) / 1000;
                if (elapsed >= duration()) {
                    clearInterval(playbackInterval);
                    setIsPlaying(false);
                    renderUpTo(duration());
                } else {
                    renderUpTo(elapsed);
                }
            }, 50);
        }
    };

    const handleSeek = (e: Event & { currentTarget: HTMLInputElement, target: HTMLInputElement }) => {
        const percent = parseFloat(e.currentTarget.value);
        const targetTime = (percent / 100) * duration();
        renderUpTo(targetTime);
        if (isPlaying()) {
            // Restart playback from new time
            clearInterval(playbackInterval);
            setIsPlaying(false);
            togglePlayback();
        }
    };

    return (
        <div class="recording-player-modal">
            <div class="player-container">
                <div class="player-header">
                    <span>🎬 Session Playback: {props.recordingId.substring(0, 8)}...</span>
                    <button class="close-btn" onClick={props.onClose}>✕</button>
                </div>

                <div class="player-body">
                    <Show when={loading()}>
                        <div class="player-loader">
                            <div class="spinner"></div>
                            <p>Fetching encrypted frames...</p>
                        </div>
                    </Show>
                    <div ref={terminalElement} class="player-terminal" />
                </div>

                <div class="player-controls">
                    <button class="play-pause-btn" onClick={togglePlayback}>
                        {isPlaying() ? '⏸' : '▶'}
                    </button>

                    <div class="timeline-container">
                        <input
                            type="range"
                            min="0"
                            max="100"
                            value={progress()}
                            onInput={handleSeek}
                            class="timeline-slider"
                        />
                        <div class="time-display">
                            {currentTime().toFixed(1)}s / {duration().toFixed(1)}s
                        </div>
                    </div>

                    <div class="player-actions">
                        <button class="action-btn-sm" onClick={() => renderUpTo(0)}>⏮</button>
                        <button class="action-btn-sm" onClick={() => renderUpTo(duration())}>⏭</button>
                    </div>
                </div>
            </div>
        </div>
    );
};
