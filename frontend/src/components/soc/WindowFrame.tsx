import { Component, JSX } from 'solid-js';

interface WindowFrameProps {
    title: string;
    type?: string;
    onClose?: () => void;
    onMinimize?: () => void;
    onMaximize?: () => void;
    onPopout?: () => void;
    children: JSX.Element;
    status?: string;
}

export const WindowFrame: Component<WindowFrameProps> = (props) => {
    return (
        <div class="h-full w-full flex flex-col bg-surface-0 border border-border-primary relative group overflow-hidden">
            {/* Sovereign Corner Decorators */}
            <div class="absolute top-0 left-0 w-2 h-2 border-t-2 border-l-2 border-accent-primary/40 z-20"></div>
            <div class="absolute top-0 right-0 w-2 h-2 border-t-2 border-r-2 border-accent-primary/40 z-20"></div>
            <div class="absolute bottom-0 left-0 w-2 h-2 border-b-2 border-l-2 border-accent-primary/40 z-20"></div>
            
            {/* Tactical Title Bar */}
            <div class="flex items-center justify-between px-3 h-8 bg-surface-1 border-b border-border-primary select-none">
                <div class="flex items-center gap-3">
                    <div class="flex gap-1">
                        <div class="w-1 h-3 bg-accent-primary/60"></div>
                    </div>
                    <span class="text-[10px] font-black tracking-widest text-text-primary uppercase font-mono truncate max-w-[150px]">
                        {props.title}
                    </span>
                    {props.status && (
                        <span class="text-[8px] px-1.5 py-0.5 bg-surface-2 text-accent-primary font-mono uppercase font-black">
                            {props.status}
                        </span>
                    )}
                </div>

                <div class="flex items-center gap-1 opacity-40 group-hover:opacity-100 transition-opacity">
                    {/* Pop-out (External Window Simulation) */}
                    <button
                        onClick={props.onPopout}
                        class="p-1 hover:bg-surface-2 text-text-secondary hover:text-accent-primary transition-all"
                        title="Pop-out Window"
                    >
                        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6" /><polyline points="15 3 21 3 21 9" /><line x1="10" y1="14" x2="21" y2="3" />
                        </svg>
                    </button>

                    <button
                        onClick={props.onMinimize}
                        class="p-1 hover:bg-surface-2 text-text-secondary hover:text-white transition-all"
                    >
                        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="5" y1="12" x2="19" y2="12" />
                        </svg>
                    </button>

                    <button
                        onClick={props.onMaximize}
                        class="p-1 hover:bg-surface-2 text-text-secondary hover:text-white transition-all"
                    >
                        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <rect x="3" y="3" width="18" height="18" rx="0" ry="0" />
                        </svg>
                    </button>

                    <button
                        onClick={props.onClose}
                        class="p-1 hover:bg-red-950 text-text-secondary hover:text-red-500 transition-all ml-1"
                    >
                        <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                            <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
                        </svg>
                    </button>
                </div>
            </div>

            {/* Content Area */}
            <div class="flex-1 relative overflow-hidden bg-surface-0">
                {props.children}

                {/* Tactical Pixel Markers */}
                <div class="absolute top-2 right-2 w-1 h-1 bg-white/5"></div>
                <div class="absolute bottom-2 left-2 w-1 h-1 bg-white/5"></div>

                {/* Subtle Background Mesh/Grid */}
                <div class="absolute inset-0 pointer-events-none opacity-[0.02] bg-[linear-gradient(rgba(255,255,255,0.05)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.05)_1px,transparent_1px)] bg-[size:24px_24px]"></div>
            </div>

            {/* Frame Status Decorator */}
            <div class="absolute bottom-0 right-0 p-1 opacity-40 pointer-events-none">
                <div class="w-3 h-3 border-r-2 border-b-2 border-accent-primary/30"></div>
            </div>
        </div>
    );
};
