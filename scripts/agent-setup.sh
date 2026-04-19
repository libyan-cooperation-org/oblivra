#!/bin/bash
# OBLIVRA Agent Setup Script (Sovereign Mode)
# Grants necessary capabilities to avoid running as root while maintaining deep visibility.

# Determine the absolute path to the agent binary
AGENT_BIN="$(pwd)/bin/oblivra-agent"

if [ ! -f "$AGENT_BIN" ]; then
    echo "❌ Error: $AGENT_BIN not found. Please build the agent first."
    exit 1
fi

echo "🛡️ Applying sovereign capabilities to $AGENT_BIN..."

# CAP_DAC_READ_SEARCH: Bypass file read permission checks (for logs like /var/log/secure)
# CAP_SYS_ADMIN: Required for eBPF operations (loading maps, attaching probes)
# CAP_NET_ADMIN: For network isolation and firewall management
# CAP_SYS_PTRACE: For deep process inspection (Phase 9 forensics)

# We use +ep (Effective and Permitted)
sudo setcap 'cap_dac_read_search,cap_sys_admin,cap_net_admin,cap_sys_ptrace+ep' "$AGENT_BIN"

if [ $? -eq 0 ]; then
    echo "✅ Successfully applied capabilities."
    getcap "$AGENT_BIN"
    echo "You can now run the agent as a non-root user while maintaining full observability."
else
    echo "❌ Failed to apply capabilities. Ensure you have sudo privileges."
    exit 1
fi
