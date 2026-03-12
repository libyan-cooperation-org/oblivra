import { Component, createSignal, onMount, For, Show, createEffect } from 'solid-js';
import * as AnalyticsService from '../../../wailsjs/go/app/AnalyticsService';

interface PlaybackProps {
    sessionId: string;
    onClose: () => void;
}

export const SessionPlayback: Component<PlaybackProps> = (props) => {
    const [frames, setFrames] = createSignal<any[]>([]);
    const [currentIndex, setCurrentIndex] = createSignal(0);
    const [isPlaying, setIsPlaying] = createSignal(false);
    const [speed, setSpeed] = createSignal(1);
    const [terminalContent, setTerminalContent] = createSignal('');
    const [loading, setLoading] = createSignal(true);

    onMount(async () => {
        try {
            const f = await (AnalyticsService as any).GetRecordingFrames(props.sessionId);
            setFrames(f || []);
        } catch (e) {
            console.error('Failed to load playback frames:', e);
        } finally {
            setLoading(false);
        }
    });

    // Handle Playback Loop
    createEffect(() => {
        if (isPlaying() && currentIndex() < frames().length - 1) {
            const nextFrame = frames()[currentIndex() + 1];
            const currentFrame = frames()[currentIndex()];
            const delay = (nextFrame.timestamp - currentFrame.timestamp) * 1000 / speed();
            
            const timer = setTimeout(() => {
                setCurrentIndex(prev => prev + 1);
            }, Math.max(10, delay));

            return () => clearTimeout(timer);
        } else if (currentIndex() >= frames().length - 1) {
            setIsPlaying(false);
        }
    });

    // Update Terminal Content when Index changes
    createEffect(() => {
        let content = '';
        const allFrames = frames();
        for (let i = 0; i <= currentIndex(); i++) {
            if (allFrames[i]) {
                content += allFrames[i].data;
            }
        }
        setTerminalContent(content);
        
        // Auto-scroll terminal
        const el = document.getElementById('playback-terminal');
        if (el) el.scrollTop = el.scrollHeight;
    });

    return (
        <div class="playback-overlay">
            <div class="playback-header">
                <div class="session-info">
                    <span class="rec-dot"></span>
                    <h3>SESSION REPLAY: {props.sessionId.substring(0, 12)}...</h3>
                </div>
                <button class="btn btn-secondary" onClick={props.onClose}>✕ CLOSE REPLAY</button>
            </div>

            <div class="playback-body">
                <Show when={loading()}>
                    <div class="playback-loading">DECRYPTING RECORDING SLICES...</div>
                </Show>
                
                <Show when={!loading() && frames().length === 0}>
                    <div class="playback-empty">NO RECORDING DATA FOUND FOR THIS SESSION.</div>
                </Show>

                <div id="playback-terminal" class="playback-terminal">
                    <pre>{terminalContent()}</pre>
                </div>
            </div>

            <div class="playback-footer">
                <div class="playback-controls">
                    <button class="ctrl-btn" onClick={() => setIsPlaying(!isPlaying())}>
                        {isPlaying() ? '⏸ PAUSE' : '▶ PLAY'}
                    </button>
                    
                    <div class="scrubber-container">
                        <input 
                            type="range" 
                            min="0" 
                            max={Math.max(0, frames().length - 1)} 
                            value={currentIndex()} 
                            onInput={(e) => {
                                setIsPlaying(false);
                                setCurrentIndex(parseInt(e.currentTarget.value));
                            }}
                            class="scrubber"
                        />
                        <div class="scrubber-labels">
                            <span>{currentIndex() + 1} / {frames().length} FRAMES</span>
                            <span>{frames()[currentIndex()]?.timestamp.toFixed(2)}s</span>
                        </div>
                    </div>

                    <select 
                        class="speed-select"
                        value={speed()} 
                        onChange={(e) => setSpeed(parseFloat(e.currentTarget.value))}
                    >
                        <option value="0.5">0.5x</option>
                        <option value="1">1x</option>
                        <option value="2">2x</option>
                        <option value="4">4x</option>
                    </select>
                </div>
            </div>

            <style>{`
                .playback-overlay {
                    position: fixed;
                    top: 0; left: 0; right: 0; bottom: 0;
                    background: var(--bg-primary);
                    z-index: 1000;
                    display: flex;
                    flex-direction: column;
                    border: 1px solid var(--glass-border);
                }
                .playback-header {
                    height: 60px;
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    padding: 0 2rem;
                    background: var(--bg-secondary);
                    border-bottom: 1px solid var(--glass-border);
                }
                .session-info { display: flex; align-items: center; gap: 1rem; }
                .rec-dot { width: 10px; height: 10px; border-radius: 50%; background: #ff4444; animation: blink 1s infinite; }
                @keyframes blink { 0% { opacity: 1; } 50% { opacity: 0.3; } 100% { opacity: 1; } }
                
                .playback-body { flex: 1; position: relative; overflow: hidden; background: #000; }
                .playback-terminal {
                    height: 100%;
                    overflow-y: auto;
                    padding: 2rem;
                    font-family: 'JetBrains Mono', 'Fira Code', monospace;
                    font-size: 14px;
                    line-height: 1.5;
                    color: #fff;
                }
                .playback-terminal pre { margin: 0; white-space: pre-wrap; word-wrap: break-word; }
                
                .playback-loading, .playback-empty {
                    position: absolute;
                    top: 50%; left: 50%;
                    transform: translate(-50%, -50%);
                    color: var(--accent-primary);
                    font-family: var(--font-mono);
                    letter-spacing: 2px;
                }

                .playback-footer {
                    height: 100px;
                    background: var(--bg-secondary);
                    border-top: 1px solid var(--glass-border);
                    display: flex;
                    align-items: center;
                    padding: 0 2rem;
                }
                .playback-controls { 
                    display: flex; 
                    align-items: center; 
                    gap: 2rem; 
                    width: 100%; 
                }
                .ctrl-btn {
                    background: var(--accent-primary);
                    color: #000;
                    border: none;
                    padding: 0.75rem 1.5rem;
                    font-weight: 700;
                    border-radius: 4px;
                    cursor: pointer;
                    min-width: 120px;
                }
                .scrubber-container { flex: 1; display: flex; flex-direction: column; gap: 0.5rem; }
                .scrubber {
                    width: 100%;
                    accent-color: var(--accent-primary);
                    cursor: pointer;
                }
                .scrubber-labels {
                    display: flex;
                    justify-content: space-between;
                    font-size: 10px;
                    color: var(--text-muted);
                    font-family: var(--font-mono);
                }
                .speed-select {
                    background: #111;
                    color: #fff;
                    border: 1px solid #333;
                    padding: 0.5rem;
                    border-radius: 4px;
                }
            `}</style>
        </div>
    );
};
