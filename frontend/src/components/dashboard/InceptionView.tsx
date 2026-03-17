import { Component, createSignal } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import '../../styles/inception.css';

export const InceptionView: Component = () => {
    const navigate = useNavigate();
    const [step, setStep] = createSignal(0);

    const steps = [
        {
            title: "Sovereign Initialization Complete",
            desc: "OBLIVRA is active and the secure vault is primed. Your instance is running in a completely isolated environment.",
            action: "Next Phase",
            next: () => setStep(1)
        },
        {
            title: "Connect Your First Asset",
            desc: "The fleet is currently offline. Deploy the OBLIVRA agent or connect via SSH to begin telemetry ingestion.",
            action: "Deploy Agent",
            next: () => navigate('/agents')
        },
        {
            title: "Enable Detection Rules",
            desc: "25+ Core Sigma rules are ready to be activated. Tune your security posture in the Policy Center.",
            action: "Review Rules",
            next: () => navigate('/governance')
        }
    ];

    return (
        <div class="inception-container animate-fade-in">
            <div class="inception-hero">
                <img src="/oblivra_inception_hero.png" alt="Inception Hero" class="inception-hero-img" />
                <div class="inception-overlay" />
            </div>

            <div class="inception-content">
                <div class="inception-badge">PHASE 0: INCEPTION</div>
                <h1 class="inception-title">{steps[step()].title}</h1>
                <p class="inception-desc">{steps[step()].desc}</p>
                
                <div class="inception-actions">
                    <button class="ob-btn ob-btn-primary inception-btn" onClick={steps[step()].next}>
                        {steps[step()].action} →
                    </button>
                    <button class="ob-btn ob-btn-ghost inception-btn" onClick={() => navigate('/terminal')}>
                        Open Terminal
                    </button>
                </div>

                <div class="inception-progress">
                    <div class={`progress-dot ${step() >= 0 ? 'active' : ''}`} />
                    <div class={`progress-dot ${step() >= 1 ? 'active' : ''}`} />
                    <div class={`progress-dot ${step() >= 2 ? 'active' : ''}`} />
                </div>
            </div>
        </div>
    );
};
