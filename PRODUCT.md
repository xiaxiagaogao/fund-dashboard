# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

The primary user is the fund operator, reviewing fund value, positions, and trading performance. Members also use the dashboard to inspect their own assets and returns, including on phones. This priority was confirmed for the September 2026 redesign.

## Product Purpose

XG fund makes a shared Binance futures account transparent to its participants. Fund accounting uses NAV units: deposits issue shares and withdrawals redeem them at the applicable NAV.

## Operating Context

The existing SvelteKit SPA has a member dashboard, an administrator review workspace, account and cash-event management, and login. A Go backend supplies authenticated data and embeds the static frontend for deployment. Snapshots are recorded periodically; data freshness must be stated accurately.

## Capabilities and Constraints

- Preserve existing business capabilities and role-based access during the redesign.
- Preserve NAV accounting, gross realized PnL, commissions, net realized PnL, and unrealized PnL distinctions.
- Preserve benchmark comparisons, allocation, member comparison, trade history, drawdown, daily PnL, and operator management workflows.
- Existing API contracts and the current SvelteKit/Tailwind stack are the implementation baseline.
- The user requested replacement layouts and visuals with a more refined finish, prioritizing operator workflows while supporting members.
- This request authorizes local frontend work; deployment is a separate task.

## Evidence on Hand

Product behavior is represented by README.md, web/src/lib/api.ts, existing routes and components, and backend API handlers. Existing design references are in the local Fund dashboard frontend design directory. Any preview fixtures must be explicitly identified as synthetic data.

## Product Principles

- Make balances, exposure, performance, and freshness easy to inspect together.
- Preserve financial meaning when changing presentation.
- Keep member workflows direct and operator controls discoverable.
- Give desktop and mobile controls appropriate, accessible interactions.

## Brand Commitments

Keep the XG fund name. The user selected the graphite research workstation direction for the redesign, with operator workflows leading the experience. Built visual rules are recorded in DESIGN.md.
