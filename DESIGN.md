---
name: Autonomous Intelligence Architecture
colors:
  surface: '#111416'
  surface-dim: '#111416'
  surface-bright: '#37393c'
  surface-container-lowest: '#0c0f11'
  surface-container-low: '#191c1e'
  surface-container: '#1d2022'
  surface-container-high: '#272a2c'
  surface-container-highest: '#323537'
  on-surface: '#e1e2e5'
  on-surface-variant: '#bacac5'
  inverse-surface: '#e1e2e5'
  inverse-on-surface: '#2e3133'
  outline: '#859490'
  outline-variant: '#3c4a46'
  surface-tint: '#3cddc7'
  primary: '#57f1db'
  on-primary: '#003731'
  primary-container: '#2dd4bf'
  on-primary-container: '#00574d'
  inverse-primary: '#006b5f'
  secondary: '#acccd4'
  on-secondary: '#16353b'
  secondary-container: '#304d55'
  on-secondary-container: '#9fbec6'
  tertiary: '#8be7ff'
  on-tertiary: '#003640'
  tertiary-container: '#3fceed'
  on-tertiary-container: '#005564'
  error: '#ffb4ab'
  on-error: '#690005'
  error-container: '#93000a'
  on-error-container: '#ffdad6'
  primary-fixed: '#62fae3'
  primary-fixed-dim: '#3cddc7'
  on-primary-fixed: '#00201c'
  on-primary-fixed-variant: '#005047'
  secondary-fixed: '#c8e8f1'
  secondary-fixed-dim: '#acccd4'
  on-secondary-fixed: '#001f25'
  on-secondary-fixed-variant: '#2e4b52'
  tertiary-fixed: '#acedff'
  tertiary-fixed-dim: '#4cd7f6'
  on-tertiary-fixed: '#001f26'
  on-tertiary-fixed-variant: '#004e5c'
  background: '#111416'
  on-background: '#e1e2e5'
  surface-variant: '#323537'
typography:
  headline-xl:
    fontFamily: Geist
    fontSize: 48px
    fontWeight: '600'
    lineHeight: 56px
    letterSpacing: -0.03em
  headline-xl-mobile:
    fontFamily: Geist
    fontSize: 32px
    fontWeight: '600'
    lineHeight: 40px
    letterSpacing: -0.02em
  headline-lg:
    fontFamily: Geist
    fontSize: 32px
    fontWeight: '600'
    lineHeight: 40px
    letterSpacing: -0.025em
  headline-lg-mobile:
    fontFamily: Geist
    fontSize: 24px
    fontWeight: '600'
    lineHeight: 32px
    letterSpacing: -0.02em
  headline-md:
    fontFamily: Geist
    fontSize: 22px
    fontWeight: '500'
    lineHeight: 28px
    letterSpacing: -0.015em
  headline-sm:
    fontFamily: Geist
    fontSize: 18px
    fontWeight: '500'
    lineHeight: 24px
    letterSpacing: -0.01em
  body-lg:
    fontFamily: Geist
    fontSize: 16px
    fontWeight: '400'
    lineHeight: 26px
    letterSpacing: -0.01em
  body-md:
    fontFamily: Geist
    fontSize: 14px
    fontWeight: '400'
    lineHeight: 22px
  body-sm:
    fontFamily: Geist
    fontSize: 12px
    fontWeight: '400'
    lineHeight: 18px
  label-lg:
    fontFamily: JetBrains Mono
    fontSize: 13px
    fontWeight: '500'
    lineHeight: 18px
    letterSpacing: -0.01em
  label-md:
    fontFamily: JetBrains Mono
    fontSize: 11px
    fontWeight: '500'
    lineHeight: 16px
    letterSpacing: 0.02em
  label-sm:
    fontFamily: JetBrains Mono
    fontSize: 10px
    fontWeight: '500'
    lineHeight: 14px
    letterSpacing: 0.04em
rounded:
  sm: 0.125rem
  DEFAULT: 0.25rem
  md: 0.375rem
  lg: 0.5rem
  xl: 0.75rem
  full: 9999px
spacing:
  gutter: 1rem
  gutter-mobile: 0.75rem
  margin: 2rem
  margin-mobile: 1rem
  space-xs: 0.25rem
  space-sm: 0.5rem
  space-md: 0.75rem
  space-lg: 1.25rem
  space-xl: 2rem
---

## Brand & Style

This design system establishes a high-performance, developer-first environment tailored for building, orchestrating, and inspecting autonomous AI coding agents. The aesthetic bridges the disciplined minimalism of modern developer environments (e.g., Linear, Supabase, Vercel) with an alchemical, arcane technical tone inspired by the wizard hat motif—transmuted into an ultra-sharp, futuristic command center.

The style merges dark-mode-first technical minimalism with subtle glassmorphism and luminous energy conduits. Deep, petrol-tinted slates anchor the canvas, while hyper-precise borders, dense data-dense layouts, and radiant cyan-emerald accents highlight real-time agent execution, telemetry, and automated synthesis. The experience evokes absolute precision, omniscient control, and effortless cognitive flow.

## Colors

The palette is engineered around dark slate petrol foundations derived directly from the wizard insignia, contrasted with electric cyan and mint-emerald execution indicators.

- **Foundational Neutrals:** 
  - Canvas / Background: `#0B0E10` (deep cosmic void)
  - Surface Neutral: `#111416` (charcoal slate)
  - Container Base: `#182024` (elevated slate container)
  - Subdued Borders: `#222D32` (structural hairline borders)
  - Muted Text & Icons: `#5A6569` and `#7E8B91`
  - Body Text: `#E2E8F0`
  - High-Emphasis Headlines: `#F8FAFC`

- **Primary & Magical Accents:**
  - Primary Accent (`#2DD4BF`): Radiant emerald cyan for interactive focal points, active agent tasks, and terminal prompts.
  - Secondary Petrol (`#1F3D44`): The signature wizard-teal extracted directly from the icon, utilized for structured card surfaces, chip containers, and selected states.
  - Tertiary Pulse (`#06B6D4`): Hyper-vibrant cyan for real-time agent stream telemetry, AST diff highlights, and live socket connections.

- **Status & Semantics:**
  - Success/Synthesized: `#10B981`
  - In-flight/Thought Chain: `#38BDF8`
  - Warning/Human-in-the-loop required: `#F59E0B`
  - Failure/Syntax Error: `#EF4444`

## Typography

The typographic hierarchy implements **Geist** for natural, razor-sharp narrative and administrative navigation, paired symmetrically with **JetBrains Mono** for technical telemetry, keybindings, commit hashes, prompt templates, and autonomous agent logs.

- **Headlines:** Set in tight negative tracking (`-0.02em` to `-0.03em`) to impart an engineered, structural tone. 
- **Code & Diagnostics:** All code editor panels, token metrics, agent thoughts, diffs, and parameter inputs leverage `JetBrains Mono` with explicit uppercase tracking for miniature badges (`label-sm`).
- **Readability & Balance:** Body text is calibrated at 14px base (`body-md`) with comfortable leading (`22px`) to facilitate scanning of verbose LLM reasoning loops.

## Layout & Spacing

The layout is built on a 12-column adaptive fluid grid suited for high-density multi-panel IDE experiences (left sidebar directory, central code/flowcanvas, right terminal/agent orchestration rail). 

- **Desktop (1280px+):** Full 3-panel split view with docked secondary toolbars, 1rem gutters, and 2rem boundary margins. Panels support dynamic resizing down to fixed minimum bounds.
- **Tablet (768px - 1279px):** Auto-collapsing sidebars into icon-dock rails; agent console slides up as a bottom drawer.
- **Mobile (< 768px):** Single-column stacked mode with persistent bottom tab navigation for switching between Agent Chat, Code Viewer, and Terminal. Margins reduce to `1rem` (`margin-mobile`).
- **Spatial Rhythm:** Built strictly around a 4px modular unit (`0.25rem`), preferring compact density (`space-sm` for inline controls, `space-md` for row spacing) to maximize contextual workspace for code and logs.

## Elevation & Depth

Visual depth is achieved through **tonal container layering** and **low-contrast micro-borders**, rejecting heavy traditional dropshadows in favor of illuminated neon backdrops:

- **Surface Tiers:**
  - *Tier 0 (Root Canvas):* `#0B0E10`
  - *Tier 1 (Sidebars & Inactive Panels):* `#111416`
  - *Tier 2 (Cards, Editors, Workbenches):* `#151B1E`
  - *Tier 3 (Modals, Context Menus, Floating Palettes):* `#1A2328` with `backdrop-filter: blur(16px)`

- **Hairline Borders:** All structural containers feature a crisp 1px solid border at `rgba(255, 255, 255, 0.07)` or tinted petrol `#1F3D44` for active panels.
- **Agent Auroral Glows:** Floating dialogs and active agent processes project a concentrated, diffuse aura: `box-shadow: 0 0 24px -6px rgba(45, 212, 191, 0.12), 0 8px 32px -4px rgba(0, 0, 0, 0.5)`.

## Shapes

The design system adopts a **soft, precision-milled radius profile (`roundedness: 1`)**. 

- Default elements (buttons, inputs, menu rows): `0.25rem` (4px).
- Containers, code blocks, diff views, and cards: `0.5rem` (8px).
- Elevated sheets, command palettes (Cmd+K), and modals: `0.75rem` (12px).
- Circular pills (`9999px`) are strictly reserved for agent execution status indicators, token usage tags, and connection status dots.

## Components

### Buttons
- **Primary:** Background `var(--primary_color_hex)` (`#2DD4BF`), foreground text `#0B0E10`, font-weight `600`. Hover state triggers a luminescent glow (`box-shadow: 0 0 16px rgba(45, 212, 191, 0.4)`).
- **Secondary (Wizard Petrol):** Background `#1F3D44`, foreground `#F8FAFC`, border 1px solid `rgba(45, 212, 191, 0.2)`. Hover shifts background to `#264B54`.
- **Ghost/Tertiary:** Background transparent, border 1px solid transparent, foreground `#7E8B91`. Hover reveals `#182024` and white text.

### Chips & Agent Status Badges
- Compact height (22px), `JetBrains Mono` 10px uppercase with letter spacing.
- Background in translucent petrol `rgba(31, 61, 68, 0.6)` with a 1px border `rgba(45, 212, 191, 0.25)`. Includes an inline 6px pulsing dot denoting agent status: green for idle, cyan for thinking, amber for human input.

### Input Fields & Terminal Prompts
- Background `#111416`, border 1px solid `#222D32`, text `#F8FAFC`, font `Geist` for descriptions, `JetBrains Mono` for code/prompt inputs.
- Active focus state swaps border to `#2DD4BF` with a subtle inset glow: `box-shadow: 0 0 0 1px #2DD4BF`.

### Cards & Workflow Nodes
- Background `#151B1E`, border 1px solid `#1F3D44`.
- Cards housing agent run loops feature a subtle gradient header rule: `1px solid linear-gradient(90deg, #2DD4BF, transparent)`.

### Lists & Tree Views
- Dense row height (28px - 32px), zero lateral margins within tree structures, subtle hover fill (`#1A2328`), and active selection bar anchored via a 2px left border in `#2DD4BF`.

### Checkboxes & Radios
- Square 14px controls with 3px corner radii. Unchecked: border 1px solid `#5A6569`. Checked: solid `#2DD4BF` with obsidian checkmark icon.

### Specialized Agent Components
- **Thought Stream Container:** Monospaced foldable accordions styled with muted left guide rules (`#222D32`) highlighting real-time agent tool executions and function calls.
- **Diff Viewers:** Line additions styled with `#10B981` at 12% alpha fill with a green left edge; deletions in `#EF4444` at 12% alpha.