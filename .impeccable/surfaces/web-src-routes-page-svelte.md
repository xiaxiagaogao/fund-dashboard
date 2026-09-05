---
version: 1
slug: "web-src-routes-page-svelte"
primary_target: "web/src/routes/+page.svelte"
related_targets: ["web/src/routes/review/+page.svelte","web/src/routes/admin/+page.svelte","web/src/routes/login/+page.svelte","web/src/lib/components/OverviewMetrics.svelte","web/src/lib/components/AllocationDonut.svelte","web/src/lib/components/PositionDonuts.svelte"]
---

# Fund Workspace

- Scope: home, review, management and login routes in the existing SvelteKit application.
- Visitor mode: Operate.
- Audience: the fund operator first; members checking personal value and returns, including on phones.
- Core task: inspect current fund value and exposure, compare performance, examine positions and completed trades, then enter review or management as authorized.
- Content order: four tonal, icon-led metric cards, benchmark chart with two allocation donuts, current positions, completed trades, and member comparison. Operator review connects drawdown, daily results and symbol contributions and retains its flat metric strip.
- Chosen direction: user-selected graphite research workstation, code-led, seed 66b5369a. Flat sections and aligned numeric columns keep the fund itself prominent.
- Signature interaction: the chart's stable historical readout responds to pointer, touch and keyboard without shifting the plot. The NAV card sparkline uses the same real history and selected range. Allocation sectors and legend rows share hover, focus, and persistent selection with fixed center readings. Mobile position disclosures expose detail within the list.
- User correction: "这一部分显示太机械了，加点ui" authorized small tonal metric cards; "这一部分我还是喜欢可视化的扇形图" restored financial allocation rings. These refine the selected graphite world without changing its unframed page sections or starting a new direction choice.
- Constraints: retain financial meanings and role access; present freshness accurately; keep management tables internally scrollable; label synthetic preview data explicitly.
- Finish evidence: `.impeccable/review/refine-desktop.png`, `refine-wide.png`, `refine-mobile.png`, and `refine-mobile-allocation.png`; finish review disposition `ship`, with no material fixes. UI checks and the production build pass.
- Unresolved decisions: none for this refinement. The accepted implementation continues into the authorized production release workflow.
