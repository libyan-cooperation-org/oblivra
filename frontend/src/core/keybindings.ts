type KeyHandler = () => void;

interface KeyBinding {
    key: string;
    ctrl?: boolean;
    shift?: boolean;
    alt?: boolean;
    meta?: boolean;
    handler: KeyHandler;
    description: string;
}

class KeybindingManager {
    private bindings: KeyBinding[] = [];
    private enabled = true;

    constructor() {
        document.addEventListener('keydown', this.handleKeyDown.bind(this));
    }

    register(binding: KeyBinding): () => void {
        this.bindings.push(binding);
        return () => { this.bindings = this.bindings.filter((b) => b !== binding); };
    }

    private handleKeyDown(e: KeyboardEvent) {
        if (!this.enabled) return;
        for (const binding of this.bindings) {
            const match =
                e.key.toLowerCase() === binding.key.toLowerCase() &&
                !!e.ctrlKey === !!binding.ctrl &&
                !!e.shiftKey === !!binding.shift &&
                !!e.altKey === !!binding.alt &&
                !!e.metaKey === !!binding.meta;
            if (match) {
                e.preventDefault();
                e.stopPropagation();
                binding.handler();
                return;
            }
        }
    }

    setEnabled(enabled: boolean) { this.enabled = enabled; }
    getAll(): KeyBinding[] { return [...this.bindings]; }
}

export const keybindings = new KeybindingManager();
