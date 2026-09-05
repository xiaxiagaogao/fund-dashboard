---
name: XG fund
description: A graphite research workstation for fund value, exposure, and performance.
colors:
  graphite-bg: "#111314"
  graphite-panel: "#181b1d"
  graphite-raised: "#222629"
  graphite-line: "#303537"
  graphite-control: "#353b3e"
  text-primary: "#f5f7f7"
  text-secondary: "#c9d0d2"
  text-mid: "#b0b9bc"
  text-muted: "#9aa5aa"
  text-quiet: "#828d92"
  mineral-green: "#afd4bf"
  mineral-green-bright: "#c6e1d1"
  mineral-green-tint: "#afd4bf14"
  soft-red: "#e58d89"
  soft-red-bright: "#efaaa5"
  soft-red-tint: "#e58d8914"
  benchmark-blue: "#8cbbea"
  benchmark-ochre: "#d3bc8d"
typography:
  headline:
    fontFamily: "Manrope, PingFang SC, Microsoft YaHei, ui-sans-serif, system-ui, sans-serif"
    fontSize: "25px"
    fontWeight: 650
    lineHeight: 1.4
    letterSpacing: "0"
  title:
    fontFamily: "Manrope, PingFang SC, Microsoft YaHei, ui-sans-serif, system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 650
    lineHeight: 1.5
    letterSpacing: "0"
  body:
    fontFamily: "Manrope, PingFang SC, Microsoft YaHei, ui-sans-serif, system-ui, sans-serif"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "0"
  label:
    fontFamily: "Manrope, PingFang SC, Microsoft YaHei, ui-sans-serif, system-ui, sans-serif"
    fontSize: "12px"
    fontWeight: 500
    lineHeight: "16px"
    letterSpacing: "0"
  metric:
    fontFamily: "IBM Plex Mono, ui-monospace, SFMono-Regular, monospace"
    fontSize: "26px"
    fontWeight: 500
    lineHeight: 1.25
    letterSpacing: "0"
rounded:
  sm: "4px"
  nav: "5px"
  md: "6px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "20px"
  section: "24px"
  wide: "32px"
components:
  button-primary:
    backgroundColor: "{colors.mineral-green}"
    textColor: "{colors.graphite-bg}"
    typography: "{typography.label}"
    rounded: "{rounded.md}"
    padding: "8px 14px"
  button-primary-hover:
    backgroundColor: "{colors.mineral-green-bright}"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.text-secondary}"
    typography: "{typography.label}"
    rounded: "{rounded.md}"
    padding: "8px 14px"
  button-ghost-hover:
    backgroundColor: "{colors.graphite-raised}"
  button-icon:
    backgroundColor: "transparent"
    textColor: "{colors.text-muted}"
    rounded: "{rounded.md}"
    height: "40px"
    width: "40px"
  input:
    backgroundColor: "{colors.graphite-bg}"
    textColor: "{colors.text-primary}"
    typography: "{typography.body}"
    rounded: "{rounded.md}"
    padding: "10px 12px"
    width: "100%"
  nav-active:
    backgroundColor: "{colors.graphite-raised}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.nav}"
    padding: "12px"
  chip-positive:
    backgroundColor: "rgb(175 212 191 / 0.08)"
    textColor: "{colors.mineral-green}"
    rounded: "{rounded.sm}"
    padding: "2px 6px"
  chip-negative:
    backgroundColor: "rgb(229 141 137 / 0.08)"
    textColor: "{colors.soft-red}"
    rounded: "{rounded.sm}"
    padding: "2px 6px"
  segment-active:
    backgroundColor: "{colors.graphite-control}"
    textColor: "{colors.text-primary}"
    typography: "{typography.label}"
    rounded: "{rounded.sm}"
    padding: "4px 12px"
  metric-band:
    backgroundColor: "transparent"
    textColor: "{colors.text-primary}"
    typography: "{typography.metric}"
  card:
    backgroundColor: "{colors.graphite-panel}"
    textColor: "{colors.text-primary}"
    rounded: "{rounded.md}"
    padding: "20px"
---

# Design System: XG fund

## Overview

**Creative North Star: "石墨投研台 / Graphite Research Workstation"**

The graphite research workstation (石墨投研台) is XG fund's restrained, continuous workspace for inspecting balances, exposure, and performance. Neutral graphite surfaces, pale mineral green, aligned numerals, and compact divisions make repeated comparison the dominant visual activity.

The interface stays flat and information-led. Manrope carries navigation and labels; IBM Plex Mono separates financial readings from prose. Color identifies fund performance, benchmarks, losses, and control state. Lucide icons and data-derived charts supply the visual assets; there is no shipping raster imagery.

**Key Characteristics:**

- Flat graphite surfaces with fine dividers.
- Compact headings and aligned financial numerals.
- Mineral green actions, distinct benchmark colors, and soft red losses.
- Desktop comparison layouts that collapse into direct mobile reading order.

## Colors

The palette is neutral graphite with pale data colors; hue carries meaning while most interface structure remains gray. Frontmatter values are normative and follow the global CSS properties and Tailwind theme.

### Primary

- **Mineral Green:** fund series, positive results, primary actions, active navigation icons, and focused field borders. Its brighter variant marks primary-button hover; its tint supports positive feedback.

### Secondary

- **Benchmark Blue:** Nasdaq 100 comparison and general keyboard-focus outlines.
- **Benchmark Ochre:** S&P 500 comparison and cautionary metadata.

### Tertiary

- **Soft Red:** negative financial results and errors; the brighter variant serves alert text and the tint supports failure feedback.

### Neutral

- **Graphite Background, Panel, and Raised:** continuous page canvas, restrained containers, and selected or hovered controls.
- **Graphite Line and Control:** structural dividers, control borders, and selected segments.
- **Primary, Secondary, Mid, Muted, and Quiet Text:** descending emphasis for values, controls, labels, timestamps, and chart axes. Keep essential values in the stronger text roles.

**The Data Color Rule.** Use the established fund, benchmark, and gain/loss colors consistently; pair color with labels, signs, or direction text.

## Typography

**Interface Font:** Manrope, with PingFang SC and Microsoft YaHei for Chinese text and the configured sans-serif fallbacks.
**Number Font:** IBM Plex Mono, with the configured monospace fallbacks.

The hierarchy is deliberately compact. Page titles use the headline role, section headings use title, ordinary interface text uses body, and field or metric labels use label. Metric values use the separate mono role. There is no display or hero type role.

At the compact breakpoint (600px), page headings and metric values become (22px). Chart legends step from (19px) to (17px). Supporting table and chart annotations use smaller local sizes; those details are not a general body-text scale. All lettering uses zero tracking.

**The Aligned Number Rule.** Use IBM Plex Mono and tabular numerals for financial values; preserve precision, units, and financial meaning.

## Layout

Use continuous sections with a shared left edge, flexible headers, and measured vertical gaps. Section padding is (24px) vertically; section-heading spacing is (20px). Metric bands have four columns from (1024px) upward and two columns below, with internal separators and stable number blocks.

The desktop shell uses a fixed sidebar (200px), a top bar (65px), and centered main content capped at (1500px) including its padding. Main padding is (32px 36px 8px). At (1100px) and below, the sidebar narrows to (176px) and main horizontal padding becomes (24px). At (767px) and below, the sidebar is replaced by a compact top bar (58px) and, when multiple routes are available, fixed bottom navigation with safe-area padding. Main horizontal padding becomes (18px), and bottom space protects content from navigation.

Comparison layouts use a flexible main track plus a secondary track (300px, reducing to 260px); they stack at (900px). Review charts stack at (1150px). Management places its form beside the ledger on a (285px) track, then stacks it at (1100px). Wide tables scroll inside their region; mobile positions become expandable rows.

## Elevation & Depth

The routed interface uses no box shadows. Tonal surfaces, fine borders, and spacing establish depth. Hover changes background or text color; focus uses an outline or field-border change. The metric band and major content sections remain unframed.

**The Flat Workspace Rule.** Separate page sections with spacing and fine rules. Reserve a bordered container for an individual item or a genuinely framed control.

## Shapes

Controls and the existing individual-card primitive use restrained corners (6px); chips and segments use (4px), navigation rows use (5px). Structural rules are thin (1px). Circular geometry is limited to identity avatars, chart markers, and native binary controls. Do not promote the retained legacy card wrappers into the page-composition model.

## Components

### Buttons

Compact, deliberate controls. Primary buttons use mineral green with graphite text; ghost buttons use a transparent surface, secondary text, and a graphite border. Both have a minimum height (40px), color transitions (150ms), and half opacity when disabled. Icon buttons are normally fixed (40px square), use Lucide glyphs, and expose their accessible name as a small tooltip on hover or keyboard focus. Compact account controls have narrower local dimensions.

General keyboard focus is a benchmark-blue outline (2px, offset 4px). Busy refresh icons rotate linearly (1s); disabled states preserve control geometry.

### Inputs / Fields

Dark, bordered fields use the shared input primitive with a minimum height (44px). Focus changes the border to mineral green; placeholder text uses the quiet role. Selects retain their native interaction and add trailing space. Errors appear in labelled alert regions; success and failure feedback use the corresponding semantic tint. Checkboxes retain native semantics and mineral-green accent color.

### Chips and Segments

Positive and negative chips use lightly tinted backgrounds and explicit text, with compact type (11px). Neutral chips use a raised graphite surface. Range controls use selected graphite segments, visible pressed state, and wrapping; the chart-specific variant reduces horizontal padding to (9px). Management tabs use a mineral-green underline instead of a filled segment.

### Cards / Containers

The existing individual-card primitive is a graphite panel with a subtle border and no shadow. The routed dashboard and review screens express their major groups as metric bands and unframed sections. Keep this distinction when reusing the retained primitive.

### Navigation

Desktop navigation combines Lucide icons with labels, a raised active row, and a mineral-green active icon. Hover brightens the row. Mobile navigation uses evenly divided icon-and-label destinations, a minimum link height (46px), and a mineral-green active state. Route availability follows the signed-in role; the current route remains explicit.

### Tables and Metric Bands

Align numbers to the right in tables and use tabular mono values. Headers are quieter than data, rows use fine separators, and hover adds a subtle graphite wash. Metric labels, primary values, and supporting notes occupy a consistent vertical sequence. Wrap long financial readings without forcing page overflow.

### Charts and Data States

Use the established series colors, restrained dashed grid lines, small markers, and low-opacity area fills. The time-series inspector keeps its legend values and timestamp above the plot; pointer, touch, and keyboard interaction update those fixed locations. Missing data has a stable empty region; a refresh dims retained chart data. Skeletons pulse (1.7s). Reduced-motion preferences disable smooth scrolling and reduce animation and transition duration to (0.01ms).

## Do's and Don'ts

- Do preserve the XG fund identity and graphite, mineral-green visual world.
- Do give amounts, percentages, units, timestamps, and data freshness stable, legible positions.
- Do use the same semantic colors and aligned number treatment across charts, tables, and summaries.
- Do preserve visible focus, labelled icon controls, loading states, and touch-friendly interactions.
- Do let sections reflow and let wide tables scroll within their own region.

- Don't turn routed page sections into stacks of floating cards.
- Don't add decorative gradients, illustrations, or motion to the financial workspace.
- Don't encode a financial distinction through color alone.
- Don't enlarge compact-panel headings into hero typography or scale type continuously with viewport width.
- Don't hide chart readings beneath pointer-following overlays or let changing values shift adjacent controls.
