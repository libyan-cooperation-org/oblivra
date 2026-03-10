import { Component, createSignal, JSX } from 'solid-js';
import '../../styles/splitpane.css';

interface SplitPaneProps {
    direction: 'horizontal' | 'vertical';
    initialSplit?: number; // 0.0 to 1.0
    minSize?: number; // pixels
    children: [JSX.Element, JSX.Element];
    onResize?: (ratio: number) => void;
}

export const SplitPane: Component<SplitPaneProps> = (props) => {
    const [split, setSplit] = createSignal(props.initialSplit || 0.5);
    const [dragging, setDragging] = createSignal(false);
    let containerRef: HTMLDivElement | undefined;
    let dividerRef: HTMLDivElement | undefined;
    const minSize = props.minSize || 100;

    const handlePointerDown = (e: PointerEvent) => {
        e.preventDefault();
        setDragging(true);
        if (dividerRef) {
            dividerRef.setPointerCapture(e.pointerId);
        }
    };

    const handlePointerMove = (e: PointerEvent) => {
        if (!dragging() || !containerRef) return;

        const rect = containerRef.getBoundingClientRect();
        let ratio: number;

        if (props.direction === 'horizontal') {
            const x = e.clientX - rect.left;
            ratio = x / rect.width;
        } else {
            const y = e.clientY - rect.top;
            ratio = y / rect.height;
        }

        const minRatio = minSize / (props.direction === 'horizontal' ? rect.width : rect.height);
        ratio = Math.max(minRatio, Math.min(1 - minRatio, ratio));

        setSplit(ratio);
        props.onResize?.(ratio);
    };

    const handlePointerUp = (e: PointerEvent) => {
        setDragging(false);
        if (dividerRef && dividerRef.hasPointerCapture(e.pointerId)) {
            dividerRef.releasePointerCapture(e.pointerId);
        }
    };

    const isHorizontal = () => props.direction === 'horizontal';

    return (
        <div
            ref={containerRef}
            class={`split-pane ${isHorizontal() ? 'horizontal' : 'vertical'}`}
            style={{
                display: 'flex',
                "flex-direction": isHorizontal() ? 'row' : 'column',
                width: '100%',
                height: '100%',
            }}
        >
            <div
                class="split-pane-first"
                style={{
                    [isHorizontal() ? 'width' : 'height']: `${split() * 100}%`,
                    overflow: 'hidden',
                    display: 'flex'
                }}
            >
                {props.children[0]}
            </div>

            <div
                ref={dividerRef}
                class={isHorizontal() ? 'ob-resizer-vertical' : 'ob-resizer-horizontal'}
                classList={{ active: dragging() }}
                onPointerDown={handlePointerDown}
                onPointerMove={handlePointerMove}
                onPointerUp={handlePointerUp}
                onDblClick={() => {
                    setSplit(0.5);
                    props.onResize?.(0.5);
                }}
            />

            <div
                class="split-pane-second"
                style={{
                    [isHorizontal() ? 'width' : 'height']: `${(1 - split()) * 100}%`,
                    overflow: 'hidden',
                    display: 'flex'
                }}
            >
                {props.children[1]}
            </div>
        </div>
    );
};
