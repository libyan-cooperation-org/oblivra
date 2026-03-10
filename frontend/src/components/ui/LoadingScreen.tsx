import { Component } from 'solid-js';
import { AppLogo } from './AppLogo';

export const LoadingScreen: Component<{ message?: string }> = (props) => {
    return (
        <div class="loading-screen-premium">
            <div class="mesh-background" />
            <div class="content">
                <div class="logo-wrapper">
                    <AppLogo size={80} class="floating-logo" />
                    <div class="logo-glow" />
                </div>
                <div class="loading-info">
                    <h1 class="shimmer-text">OBLIVRASHELL</h1>
                    <div class="progress-container">
                        <div class="progress-bar-glow" />
                        <div class="progress-bar-infinite" />
                    </div>
                    <p class="status-message">{props.message || 'Initializing quantum core...'}</p>
                </div>
            </div>

            <style>{`
                .loading-screen-premium {
                    position: fixed;
                    inset: 0;
                    background: var(--bg-primary);
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    z-index: 9999;
                    overflow: hidden;
                }

                .mesh-background {
                    position: absolute;
                    inset: 0;
                    background: var(--mesh-gradient);
                    opacity: 0.4;
                    filter: blur(60px);
                    animation: pulse-accent 8s ease-in-out infinite;
                }

                .content {
                    position: relative;
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    gap: 40px;
                    animation: fade-in-up 0.8s var(--anim-easing-enter);
                }

                .logo-wrapper {
                    position: relative;
                }

                .floating-logo {
                    filter: drop-shadow(0 0 20px var(--accent-glow));
                    animation: float 4s ease-in-out infinite;
                }

                .logo-glow {
                    position: absolute;
                    inset: -20px;
                    background: var(--accent-glow);
                    filter: blur(40px);
                    opacity: 0.3;
                    border-radius: 50%;
                }

                .loading-info {
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    gap: 16px;
                }

                .shimmer-text {
                    font-size: 24px;
                    font-weight: 800;
                    letter-spacing: 4px;
                    background: linear-gradient(90deg, #fff 0%, var(--accent-primary) 50%, #fff 100%);
                    background-size: 200% auto;
                    -webkit-background-clip: text;
                    -webkit-text-fill-color: transparent;
                    animation: shimmer-text-flow 3s linear infinite;
                }

                .progress-container {
                    width: 240px;
                    height: 2px;
                    background: rgba(255, 255, 255, 0.05);
                    border-radius: 1px;
                    position: relative;
                    overflow: hidden;
                }

                .progress-bar-glow {
                    position: absolute;
                    inset: 0;
                    box-shadow: 0 0 10px var(--accent-glow);
                }

                .progress-bar-infinite {
                    position: absolute;
                    height: 100%;
                    width: 40%;
                    background: var(--accent-gradient);
                    border-radius: 1px;
                    animation: slide-infinite 1.5s var(--anim-easing-standard) infinite;
                }

                .status-message {
                    font-size: 13px;
                    color: var(--text-secondary);
                    font-family: var(--font-mono);
                    opacity: 0.8;
                }

                @keyframes float {
                    0%, 100% { transform: translateY(0); }
                    50% { transform: translateY(-15px); }
                }

                @keyframes shimmer-text-flow {
                    to { background-position: 200% center; }
                }

                @keyframes slide-infinite {
                    0% { left: -40%; }
                    100% { left: 140%; }
                }
            `}</style>
        </div>
    );
};
