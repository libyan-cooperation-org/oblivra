import { Component, createMemo } from 'solid-js';

interface SparklineProps {
    data: number[];
    color: string;
    width?: number; // px
    height?: number; // px
    strokeWidth?: number;
}

export const Sparkline: Component<SparklineProps> = (props) => {
    // We assume a viewBox of 100x100 for easy scaling, but stretch via width/height
    const viewBoxWidth = 100;
    const viewBoxHeight = 100;

    const pathData = createMemo(() => {
        const { data } = props;
        if (!data || data.length === 0) return '';

        const max = Math.max(...data);
        const min = Math.min(...data);
        const range = max - min || 1; // Prevent div by zero if flatline

        const stepX = viewBoxWidth / (data.length - 1);

        // Build the SVG path string (M x,y L x,y ...)
        return data.reduce((path, val, idx) => {
            // Normalize Y to 0-100 (where 0 is top and 100 is bottom in SVG coords)
            // Leave a small 5% margin top/bottom so stroke isn't clipped
            const normalizedY = 95 - (((val - min) / range) * 90);
            const x = idx * stepX;

            if (idx === 0) {
                return `M ${x},${normalizedY}`;
            }

            // Simple Cubic Bezier for smooth curves (C x1 y1, x2 y2, x y)
            const prevX = (idx - 1) * stepX;
            const prevVal = data[idx - 1];
            const prevY = 95 - (((prevVal - min) / range) * 90);

            const controlPointX1 = prevX + (stepX * 0.5);
            const controlPointY1 = prevY;
            const controlPointX2 = x - (stepX * 0.5);
            const controlPointY2 = normalizedY;

            return `${path} C ${controlPointX1},${controlPointY1} ${controlPointX2},${controlPointY2} ${x},${normalizedY}`;
        }, '');
    });

    const areaPathData = createMemo(() => {
        const line = pathData();
        if (!line) return '';
        return `${line} L ${viewBoxWidth},${viewBoxHeight} L 0,${viewBoxHeight} Z`;
    });

    const rgbaFill = createMemo(() => {
        return props.color.startsWith('rgb')
            ? props.color.replace('rgb', 'rgba').replace(')', ', 0.2)')
            : `${props.color}33`; // Hex fallback + 20% opacity
    });

    return (
        <svg
            width={props.width || '100%'}
            height={props.height || 32}
            viewBox={`0 0 ${viewBoxWidth} ${viewBoxHeight}`}
            preserveAspectRatio="none"
            style="overflow: visible; display: block;"
        >
            <defs>
                <linearGradient id={`gradient-${props.color}`} x1="0" x2="0" y1="0" y2="1">
                    <stop offset="0%" stop-color={rgbaFill()} />
                    <stop offset="100%" stop-color="transparent" />
                </linearGradient>
            </defs>
            <path
                d={areaPathData()}
                fill={`url(#gradient-${props.color})`}
                stroke="none"
            />
            <path
                d={pathData()}
                fill="none"
                stroke={props.color}
                stroke-width={props.strokeWidth || 2}
                stroke-linecap="round"
                stroke-linejoin="round"
                vector-effect="non-scaling-stroke"
            />
        </svg>
    );
};
