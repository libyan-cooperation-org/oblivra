#!/bin/bash
set -euo pipefail

VERSION="${1:-0.1.0}"
APP_NAME="sovereign-terminal"
PKG_DIR="build/packages"

echo "📦 Packaging Sovereign Terminal v${VERSION}"
mkdir -p "${PKG_DIR}"

# Debian package
if command -v dpkg-deb &> /dev/null; then
    echo "🐧 Creating .deb package..."
    DEB_DIR="${PKG_DIR}/deb"
    mkdir -p "${DEB_DIR}/DEBIAN"
    mkdir -p "${DEB_DIR}/usr/bin"
    mkdir -p "${DEB_DIR}/usr/share/applications"
    mkdir -p "${DEB_DIR}/usr/share/icons/hicolor/256x256/apps"

    cp "build/bin/${APP_NAME}" "${DEB_DIR}/usr/bin/"

    cat > "${DEB_DIR}/DEBIAN/control" << EOF
Package: sovereign-terminal
Version: ${VERSION}
Section: net
Priority: optional
Architecture: amd64
Depends: libgtk-3-0, libwebkit2gtk-4.0-37
Maintainer: Sovereign Security <support@sovereign.dev>
Description: Secure Terminal Management Platform
 A sovereign security terminal platform for SSH management,
 credential vaulting, and infrastructure operations.
EOF

    cat > "${DEB_DIR}/usr/share/applications/${APP_NAME}.desktop" << EOF
[Desktop Entry]
Name=Sovereign Terminal
Comment=Secure Terminal Management Platform
Exec=/usr/bin/${APP_NAME}
Icon=sovereign-terminal
Terminal=false
Type=Application
Categories=Network;System;Security;
EOF

    cp build/appicon.png "${DEB_DIR}/usr/share/icons/hicolor/256x256/apps/${APP_NAME}.png" 2>/dev/null || true

    dpkg-deb --build "${DEB_DIR}" "${PKG_DIR}/${APP_NAME}_${VERSION}_amd64.deb"
    echo "  ✅ ${APP_NAME}_${VERSION}_amd64.deb"
fi

# AppImage
if command -v appimagetool &> /dev/null; then
    echo "📦 Creating AppImage..."
    APPDIR="${PKG_DIR}/AppDir"
    mkdir -p "${APPDIR}/usr/bin"
    mkdir -p "${APPDIR}/usr/share/icons/hicolor/256x256/apps"

    cp "build/bin/${APP_NAME}" "${APPDIR}/usr/bin/"

    cat > "${APPDIR}/AppRun" << 'EOF'
#!/bin/bash
SELF=$(readlink -f "$0")
HERE=${SELF%/*}
exec "${HERE}/usr/bin/sovereign-terminal" "$@"
EOF
    chmod +x "${APPDIR}/AppRun"

    cat > "${APPDIR}/${APP_NAME}.desktop" << EOF
[Desktop Entry]
Name=Sovereign Terminal
Exec=sovereign-terminal
Icon=sovereign-terminal
Type=Application
Categories=Network;System;
EOF

    cp build/appicon.png "${APPDIR}/${APP_NAME}.png" 2>/dev/null || true
    cp build/appicon.png "${APPDIR}/usr/share/icons/hicolor/256x256/apps/${APP_NAME}.png" 2>/dev/null || true

    ARCH=x86_64 appimagetool "${APPDIR}" "${PKG_DIR}/${APP_NAME}-${VERSION}-x86_64.AppImage"
    echo "  ✅ ${APP_NAME}-${VERSION}-x86_64.AppImage"
fi

echo ""
echo "📁 Packages:"
ls -lh "${PKG_DIR}/"*.{deb,AppImage,rpm} 2>/dev/null || echo "  (no packages built)"

PHASE 4 COMPLETION CHECKLIST

text

✅ AI-Powered Command Suggestions
   ✅ Command history indexing and frequency tracking
   ✅ Context-aware completions (Docker, K8s, Git, system)
   ✅ Natural language → command translation (50+ patterns)
   ✅ Error explanation (15+ error patterns)
   ✅ Fix suggestions for failed commands
   ✅ Dangerous command detection (25+ patterns)
   ✅ Ollama/local LLM integration
   ✅ Session context detection (OS, tools, directory)

✅ Cloud Provider Integrations
   ✅ AWS EC2 instance discovery via CLI
   ✅ GCP Compute instance discovery via gcloud
   ✅ Cloud manager for multi-provider orchestration
   ✅ Start/stop instances
   ✅ SSH user auto-detection by platform
   ✅ Region listing and filtering

✅ Kubernetes Native Support
   ✅ Context management (list, switch)
   ✅ Namespace listing
   ✅ Pod listing with status, containers, restarts
   ✅ Service listing with ports
   ✅ kubectl exec into pods
   ✅ Pod log retrieval
   ✅ Port-forward management
   ✅ Age calculation and resource parsing

✅ Connection Health Dashboard
   ✅ Periodic TCP health checks
   ✅ Latency tracking with history
   ✅ Health status (healthy/degraded/unreachable)
   ✅ Success rate calculation
   ✅ SSH banner detection
   ✅ Dashboard aggregate statistics
   ✅ Configurable check intervals
   ✅ Callback notifications

✅ Smart Host Discovery
   ✅ SSH config parser (~/.ssh/config)
   ✅ Terraform state file parser (AWS + GCP)
   ✅ Ansible inventory parser (INI format)
   ✅ SSH known_hosts parser
   ✅ Deduplication
   ✅ Source availability detection
   ✅ Metadata extraction

✅ Monitoring & Observability
   ✅ Counter, Gauge, Histogram metric types
   ✅ 20+ default metrics
   ✅ Prometheus exposition format
   ✅ HTTP metrics endpoint
   ✅ SSH, session, tunnel, vault, SFTP, security metrics
   ✅ Connection latency histograms

✅ Cross-Platform Packaging & Auto-Update
   ✅ GitHub releases update checker
   ✅ Semver comparison
   ✅ Platform-specific asset detection
   ✅ Download with SHA256 verification
   ✅ Binary replacement with backup
   ✅ Auto-check with configurable interval
   ✅ .deb package script
   ✅ AppImage package script
   ✅ Cross-compilation build script