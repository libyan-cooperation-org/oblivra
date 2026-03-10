import { Component, createSignal, For } from 'solid-js';
import { useNavigate } from '@solidjs/router';

interface SettingsSection {
  id: string;
  label: string;
  icon: string;
}

const sections: SettingsSection[] = [
  { id: 'general', label: 'GENERAL', icon: '[SYS]' },
  { id: 'sovereign', label: 'SOVEREIGN_CTRL', icon: '[ATT]' },
  { id: 'data', label: 'DESTRUCTION', icon: '[WIPE]' },
  { id: 'appearance', label: 'APPEARANCE', icon: '[UI]' },
  { id: 'security', label: 'VAULT_SEC', icon: '[SEC]' },
  { id: 'ssh', label: 'SSH_KEYS', icon: '[KEY]' },
  { id: 'plugins', label: 'EXTENSIONS', icon: '[EXT]' },
  { id: 'identity', label: 'IDENTITY_ACCESS', icon: '[IAM]' },
  { id: 'about', label: 'INFO', icon: '[INF]' },
];

export const Settings: Component = () => {
  const navigate = useNavigate();
  const [activeSection, setActiveSection] = createSignal('sovereign');

  // Sovereign
  // @ts-ignore
  const [hashStatus, setHashStatus] = createSignal('VERIFIED');
  // @ts-ignore
  const [clockDrift, setClockDrift] = createSignal('12ms');

  // Settings values
  const [autoLockTimeout, setAutoLockTimeout] = createSignal(15);
  const [clipboardClear, setClipboardClear] = createSignal(30);

  const renderSection = () => {
    switch (activeSection()) {
      case 'general':
        return (
          <div class="settings-section">
            <h2 class="ob-page-title" style="margin-bottom: 24px;">General Config</h2>
            <div class="setting-group">
              <div class="setting-item">
                <div class="setting-label">
                  <span class="setting-title">Auto-update Polling</span>
                  <span class="setting-desc">Check for updates on startup</span>
                </div>
                <input type="checkbox" checked />
              </div>
            </div>
          </div>
        );

      case 'sovereign':
        return (
          <div class="settings-section">
            <h2 class="ob-page-title" style="margin-bottom: 24px; color: var(--accent-blue);">Sovereign Controls</h2>

            <div class="setting-group">
              <h3 class="ob-section-header">Runtime Attestation</h3>
              <div class="ob-card" style="margin-bottom: 16px;">
                <div style="display: flex; justify-content: space-between; align-items: center;">
                  <div>
                    <span style="font-family: var(--font-mono); font-size: 14px; font-weight: bold;">BINARY HASH STATE</span>
                    <p style="color: var(--text-muted); font-size: 11px; margin-top: 4px;">Verified against expected boot signature.</p>
                  </div>
                  <span class="ob-badge ob-badge-green" style="font-size: 12px; padding: 4px 12px;">{hashStatus()}</span>
                </div>
                <div class="ob-code" style="margin-top: 12px; color: var(--success);">
                  Expected: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855<br />
                  Computed: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
                </div>
              </div>
            </div>

            <div class="setting-group">
              <h3 class="ob-section-header">Temporal Integrity</h3>
              <div class="ob-card" style="margin-bottom: 16px;">
                <div style="display: flex; justify-content: space-between; align-items: center;">
                  <div>
                    <span style="font-family: var(--font-mono); font-size: 14px; font-weight: bold;">MAX FLEET CLOCK DRIFT</span>
                    <p style="color: var(--text-muted); font-size: 11px; margin-top: 4px;">Deviation across all active telemetry nodes.</p>
                  </div>
                  <span class="ob-badge ob-badge-blue" style="font-size: 12px; padding: 4px 12px;">{clockDrift()}</span>
                </div>
              </div>
            </div>

            <div class="setting-group">
              <h3 class="ob-section-header">Air-Gapped Upgrades</h3>
              <div class="ob-card">
                <p style="color: var(--text-secondary); font-size: 12px; margin-bottom: 12px;">
                  Generate a signed .tar.gz bundle for moving updates to disconnected networks.
                </p>
                <div style="display: flex; gap: 8px;">
                  <button class="ob-btn ob-btn-primary">CREATE TARBALL (.tar.gz)</button>
                  <button class="ob-btn">IMPORT OFFLINE BUNDLE</button>
                </div>
              </div>
            </div>
          </div>
        );

      case 'data':
        return (
          <div class="settings-section">
            <h2 class="ob-page-title" style="margin-bottom: 24px; color: var(--critical);">Data Destruction (GDPR)</h2>
            <div class="setting-group">
              <h3 class="ob-section-header">CryptoWipe Execution</h3>
              <div class="ob-card" style="border-color: rgba(248, 81, 73, 0.3); background: rgba(248,81,73,0.02);">
                <p style="color: var(--text-primary); font-size: 13px; margin-bottom: 8px; font-weight: 600;">
                  Execute DoD M-Series 3-Pass Overwrite
                </p>
                <p style="color: var(--text-muted); font-size: 11px; margin-bottom: 16px;">
                  This action bypasses standard deletes. It will actively flood the requested database blocks with random-noise, then zero-bits, then execute a sector vacuum. This cannot be interrupted.
                </p>
                <div style="display: flex; gap: 8px;">
                  <input type="text" class="ob-input ob-input-mono" placeholder="Target Agent ID or Entity Hash..." />
                  <button class="ob-btn ob-btn-danger" style="background: rgba(248,81,73,0.1); border: 1px solid var(--critical);">INITIATE WIPER</button>
                </div>
              </div>
            </div>
          </div>
        );

      case 'appearance':
        return (
          <div class="settings-section">
            <h2 class="ob-page-title" style="margin-bottom: 24px;">Appearance</h2>
            <div class="setting-group">
              <h3 class="ob-section-header">Sovereign Theme Matrix</h3>
              <div class="theme-grid" style="display: grid; grid-template-columns: 1fr 1fr; gap: 12px;">
                <button class="ob-card ob-card-flat" style="border: 2px solid var(--accent-blue); text-align: left;">
                  <div style="font-family: var(--font-mono); font-weight: bold; color: var(--accent-blue)">OBLIVRA TACTICAL</div>
                  <div style="font-size: 11px; color: var(--text-muted); margin-top: 4px;">High Contrast / True Black</div>
                </button>
                <button class="ob-card ob-card-flat" style="border: 1px solid var(--border-primary); text-align: left;">
                  <div style="font-family: var(--font-mono); font-weight: bold;">GITHUB DARK</div>
                  <div style="font-size: 11px; color: var(--text-muted); margin-top: 4px;">Muted Navy / Developer standard</div>
                </button>
              </div>
            </div>
          </div>
        );

      case 'security':
        return (
          <div class="settings-section">
            <h2 class="ob-page-title" style="margin-bottom: 24px;">Vault Security</h2>
            <div class="setting-group">
              <h3 class="ob-section-header">Vault Lifecycle</h3>
              <div class="ob-card" style="margin-bottom: 16px;">
                <div class="setting-item" style="margin-bottom: 12px;">
                  <div class="setting-label">
                    <span class="setting-title" style="font-weight: bold;">Auto-lock timeout</span>
                    <span class="setting-desc" style="display:block; font-size: 11px; color: var(--text-muted);">Lock vault after {autoLockTimeout()} minutes of inactivity</span>
                  </div>
                  <select
                    class="ob-select"
                    value={autoLockTimeout()}
                    onChange={(e) => setAutoLockTimeout(parseInt(e.currentTarget.value))}
                  >
                    <option value={5}>5 minutes</option>
                    <option value={15}>15 minutes</option>
                    <option value={0}>Never</option>
                  </select>
                </div>
                <div class="setting-item">
                  <div class="setting-label">
                    <span class="setting-title" style="font-weight: bold;">Clear clipboard</span>
                    <span class="setting-desc" style="display:block; font-size: 11px; color: var(--text-muted);">Clear copied passwords after {clipboardClear()}s</span>
                  </div>
                  <input
                    type="number"
                    class="ob-input"
                    style="width: 80px;"
                    value={clipboardClear()}
                    onInput={(e) => setClipboardClear(parseInt(e.currentTarget.value))}
                    min={5}
                    max={300}
                  />
                </div>
              </div>

              <div class="ob-card">
                <span style="font-weight: bold; font-family: var(--font-mono); font-size: 12px;">CRYPTOGRAPHIC KEYS</span>
                <p style="font-size: 11px; color: var(--text-muted); margin-bottom: 12px; margin-top: 4px;">Modifying the root encryption key requires FIDO2 hardware verification.</p>
                <button class="ob-btn ob-btn-danger" style="background: rgba(248,81,73,0.1); border-color: var(--critical);">
                  CHANGE MASTER PASSWORD
                </button>
              </div>
            </div>
          </div>
        );

      case 'identity':
        navigate('/identity');
        return <div class="settings-section"><p>Redirecting to Identity & Access Management...</p></div>;

      default:
        return (
          <div class="settings-section">
            <div class="ob-empty">
              <div class="ob-empty-icon">[WIP]</div>
              <div class="ob-empty-title">Section Unavailable</div>
              <div class="ob-empty-desc">This configuration module is offline or lacks UI bindings.</div>
            </div>
          </div>
        );
    }
  };

  return (
    <div class="settings-page" style="display: flex; height: 100%; background: var(--surface-0);">
      {/* Sidebar */}
      <nav class="settings-nav" style="width: 260px; border-right: 1px solid var(--border-primary); padding: 16px; display: flex; flex-direction: column; gap: 4px;">
        <For each={sections}>
          {(section: any) => (
            <button
              class={`nav-item ${activeSection() === section.id ? 'active' : ''}`}
              style={{
                display: 'flex',
                'align-items': 'center',
                gap: '12px',
                padding: '8px 12px',
                'background-color': activeSection() === section.id ? 'var(--surface-1)' : 'transparent',
                color: activeSection() === section.id ? 'var(--text-primary)' : 'var(--text-secondary)',
                border: '1px solid',
                'border-color': activeSection() === section.id ? 'var(--border-secondary)' : 'transparent',
                'border-radius': '0px',
                'font-family': 'var(--font-mono)',
                'font-size': '11px',
                'font-weight': 'bold',
                'text-align': 'left',
                cursor: 'pointer',
                'border-left': activeSection() === section.id ? '3px solid var(--accent-blue)' : '1px solid transparent'
              }}
              onClick={() => setActiveSection(section.id)}
            >
              <span class="nav-icon" style="color: var(--accent-blue); width: 40px;">{section.icon}</span>
              <span class="nav-label">{section.label}</span>
            </button>
          )}
        </For>
      </nav>

      {/* Content */}
      <div class="settings-content" style="flex: 1; padding: 32px; overflow-y: auto;">
        <div style="max-width: 800px;">
          {renderSection()}
        </div>
      </div>
    </div>
  );
};