# Design System — Colign

## Product Context
- **What this is:** Spec-driven development platform where teams write, review, and align on specs before AI implements code
- **Who it's for:** Developers and PMs working in teams
- **Space/industry:** DevTool / PM collaboration (peers: Linear, Notion, GitHub Projects, Plane)
- **Project type:** Web app / dashboard with strong document-editing focus

## Aesthetic Direction
- **Direction:** Clean Developer Tool — cool, precise, focused
- **Decoration level:** Intentional — borders over shadows, flat architectural surfaces
- **Mood:** Focused and professional. Clear hierarchy, good readability, distinct status colors. Specs are treated as documents with weight, not tickets in a list.
- **Reference sites:** Linear (restraint, blue palette), Notion (readability)

## Typography
- **All text:** Pretendard — clean, modern sans-serif with excellent Korean support and readability at all sizes. Used for everything: headlines, body, UI labels.
- **Code/Data:** Geist Mono — already in project, good tabular-nums support
- **Loading:**
  - Pretendard: `https://cdn.jsdelivr.net/gh/orioncactus/pretendard@v1.3.9/dist/web/variable/pretendardvariable-dynamic-subset.min.css`
  - Geist Mono: already bundled via `geist` package
- **Scale:**
  - Display (spec titles): 36px / line-height 1.2 / font-weight 700
  - Section heading: 22px / line-height 1.3 / font-weight 600
  - Body (spec content): 15px / line-height 1.7 / font-weight 400
  - UI labels/nav: 13px / line-height 1.4 / font-weight 500
  - Caption/timestamp: 11px / line-height 1.4 / uppercase tracking 0.06em / font-weight 600

## Color

### Approach
Restrained — cool neutral dark foundation with the existing blue primary. Semantic colors are muted versions of their full-strength counterparts. The blue primary provides focus and trust.

### Dark Mode (default)
| Token | OKLch | Hex (approx) | Usage |
|-------|-------|------|-------|
| `--background` | `oklch(0.11 0.005 270)` | `#0F1014` | App background — cool near-black with faint blue undertone |
| `--card` / `--surface` | `oklch(0.15 0.005 270)` | `#17181D` | Cards, panels, sidebars |
| `--popover` / `--elevated` | `oklch(0.18 0.005 270)` | `#1E1F25` | Modals, dropdowns, popovers |
| `--bg-hover` | `oklch(0.21 0.005 270)` | `#24252C` | Hover states |
| `--border` | `oklch(0.25 0.005 270)` | `#2D2E36` | Default borders — visible, not washed out |
| `--border-strong` | `oklch(0.33 0.005 270)` | `#3E4048` | Active/focused borders |
| `--foreground` | `oklch(0.93 0.005 270)` | `#E6E6EC` | Primary text — cool off-white |
| `--secondary-foreground` | `oklch(0.78 0.005 270)` | `#B0B0BA` | Secondary text |
| `--muted-foreground` | `oklch(0.55 0.005 270)` | `#7C7C88` | Muted text, metadata |
| `--text-faint` | `oklch(0.40 0.005 270)` | `#52525C` | Placeholders, disabled |
| `--primary` | `oklch(0.623 0.214 259.815)` | `#5B6CF5` | Brand primary — existing blue |
| `--primary-foreground` | `oklch(0.985 0 0)` | `#FBFBFB` | Text on primary |
| `--accent` | `oklch(0.21 0.005 270)` | `#24252C` | Accent surface |
| `--destructive` | `oklch(0.65 0.2 25)` | `#E0564B` | Error/destructive actions |
| `--ring` | `oklch(0.623 0.214 259.815)` | `#5B6CF5` | Focus ring — matches primary |

### Light Mode
| Token | OKLch | Usage |
|-------|-------|-------|
| `--background` | `oklch(0.98 0.002 270)` | App background |
| `--card` | `oklch(1 0 0)` | Cards, panels |
| `--border` | `oklch(0.90 0.003 270)` | Default borders |
| `--foreground` | `oklch(0.15 0.005 270)` | Primary text |
| `--muted-foreground` | `oklch(0.55 0.005 270)` | Muted text |
| `--primary` | `oklch(0.50 0.214 259.815)` | Brand primary (darker for light bg) |

### Semantic Colors
| Token | Dark | Usage |
|-------|------|-------|
| `--success` | `oklch(0.72 0.19 145)` | Approved, passing, positive — green |
| `--warning` | `oklch(0.78 0.16 75)` | Needs attention, pending — amber |
| `--error` | `oklch(0.65 0.2 25)` | Rejected, breaking, destructive — red |
| `--info` | `oklch(0.70 0.12 230)` | Informational — light blue |

### Workflow Stage Colors
| Stage | OKLch | Hex (approx) | Usage |
|-------|-------|------|-------|
| Draft | `oklch(0.64 0.02 270)` | `#8B8FA0` | Muted gray — not yet structured |
| Spec | `oklch(0.78 0.16 75)` | `#E5A83B` | Amber — needs attention, under review |
| Review | `oklch(0.70 0.12 230)` | `#5EA3D4` | Light blue — reviewing |
| Approved | `oklch(0.72 0.19 145)` | `#6DBF6D` | Green — ready to implement |

### Chart Colors
| Token | OKLch | Usage |
|-------|-------|-------|
| `--chart-1` | `oklch(0.623 0.214 259.815)` | Primary blue |
| `--chart-2` | `oklch(0.72 0.19 145)` | Green |
| `--chart-3` | `oklch(0.78 0.16 75)` | Amber |
| `--chart-4` | `oklch(0.70 0.12 230)` | Light blue |
| `--chart-5` | `oklch(0.65 0.18 310)` | Purple |

## Spacing
- **Base unit:** 4px
- **Density:** Comfortable
- **Scale:** 2xs(2px) xs(4px) sm(8px) md(16px) lg(24px) xl(32px) 2xl(48px) 3xl(64px)
- **Spec content area:** generous — line-height 1.7, max-width 680px for optimal reading

## Layout
- **Approach:** Grid-disciplined
- **Grid:** Sidebar (220px fixed) + content area
- **Max content width:** 1152px / `max-w-6xl` (dashboard, project list), 1024px / `max-w-5xl` (settings, change detail), full-width (project detail — workspace with wiki sidebar + editor), 680px (spec reading area)
- **Border radius:** Tight hierarchy — sm:4px, md:6px, lg:8px (no rounded-full on cards)
- **Surfaces:** Flat, bordered. No box-shadows on cards. Border is the primary spatial separator.

## Motion
- **Approach:** Minimal-functional — only transitions that aid comprehension
- **Easing:** enter(ease-out) exit(ease-in) move(ease-in-out)
- **Duration:** micro(50-100ms) short(150-200ms) medium(200-300ms)
- **Rules:** No bounce, no spring physics. Slide, reveal, fade only. Respect `prefers-reduced-motion`.

## Anti-patterns (do NOT use)
- Purple/violet gradients
- Glass/frosted surfaces or backdrop-blur cards
- Box shadows on cards (use borders)
- Opacity below 40% on borders or backgrounds (was a previous issue)
- Centered-everything layouts
- Decorative blobs or abstract shapes
- Full-rounded corners on content cards

## Decisions Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-04-07 | Initial design system created | Competitive research (Linear, Notion, Shortcut, Plane) + Codex & Claude design voices. |
| 2026-04-07 | Pretendard for all text | User preference: Korean language support + clean modern feel. Replaces Geist Sans. Single font for consistency. |
| 2026-04-07 | Keep existing blue primary | `oklch(0.623 0.214 259.815)` — existing brand color maintained. No arbitrary color change. |
| 2026-04-07 | Cool neutral dark palette | User feedback: warm brown undertone was too heavy. Cool grays with faint blue undertone to complement primary. |
| 2026-04-07 | No serif fonts | User preference: no Instrument Serif. Pretendard only, differentiation through weight hierarchy. |
| 2026-04-08 | Stage colors redesigned | Draft→Gray, Spec→Amber, Approved→Green. Primary blue was identical to Spec blue — separated by making stages status-based (neutral→attention→positive). |
