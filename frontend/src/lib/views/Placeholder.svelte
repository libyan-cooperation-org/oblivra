<script lang="ts">
  let { name }: { name: string } = $props();
</script>

<div style="display:flex; flex-direction:column; align-items:center; justify-content:center; height:100%; padding:64px 32px; text-align:center;">

  <!-- Animated bracket frame -->
  <div style="position:relative; width:72px; height:72px; margin-bottom:24px;">
    <!-- Corner brackets -->
    <span style="position:absolute; top:0;    left:0;  width:16px; height:16px; border-top:2px solid var(--color-cyan-500); border-left:2px solid var(--color-cyan-500); opacity:0.8;"></span>
    <span style="position:absolute; top:0;    right:0; width:16px; height:16px; border-top:2px solid var(--color-cyan-500); border-right:2px solid var(--color-cyan-500); opacity:0.8;"></span>
    <span style="position:absolute; bottom:0; left:0;  width:16px; height:16px; border-bottom:2px solid var(--color-cyan-500); border-left:2px solid var(--color-cyan-500); opacity:0.8;"></span>
    <span style="position:absolute; bottom:0; right:0; width:16px; height:16px; border-bottom:2px solid var(--color-cyan-500); border-right:2px solid var(--color-cyan-500); opacity:0.8;"></span>

    <!-- Center icon -->
    <div style="position:absolute; inset:12px; display:grid; place-items:center; font-size:28px; color:var(--color-base-300);">◌</div>
  </div>

  <h2 style="font-family:'Rajdhani',sans-serif; font-weight:700; font-size:20px; letter-spacing:3px; text-transform:uppercase; color:#e8f4f8; margin-bottom:8px;">{name}</h2>

  <div style="width:48px; height:1px; background:linear-gradient(90deg,transparent,var(--color-cyan-500),transparent); margin:0 auto 16px;"></div>

  <p style="font-family:'Share Tech Mono',monospace; font-size:11px; letter-spacing:0.5px; line-height:1.8; color:var(--color-base-200); max-width:420px;">
    This surface is part of the OBLIVRA roadmap but isn't wired up yet.
    The skeleton is live — real services will plug in during the next phases.
  </p>

  <div style="margin-top:24px; padding:6px 16px; border:1px solid var(--color-base-600); background:var(--color-base-800); display:inline-block;">
    <span style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:3px; color:var(--color-sig-warn); text-transform:uppercase;">● PENDING IMPLEMENTATION</span>
  </div>
</div>
