# UI Framework Reference

`tool/utilivos/` ships a layered framework intended to cover most plugin UI work with little or no custom CSS.

## File layers

- `base.css` - design tokens, semantic aliases, focus/motion defaults, global reset
- `layout.css` - page and layout primitives (`stack`, `inline`, `grid`, `split`, containers)
- `utilities.css` - spacing, display, alignment, sizing, overflow, truncation, visibility, interaction helpers
- `patterns.css` - semantic screen patterns (settings, forms, results, dashboard stats, searchable list, detail sections, action bars)
- `components.css` - reusable controls and surfaces; imports `utilities.css` and `patterns.css`
- `tool.js` - `window.UtilivosUI` helper factories and localization/theme helpers

Backward compatibility: existing imports remain valid:

```html
<link rel="stylesheet" href="./utilivos/base.css" />
<link rel="stylesheet" href="./utilivos/layout.css" />
<link rel="stylesheet" href="./utilivos/components.css" />
<script src="./utilivos/tool.js"></script>
```

## Naming conventions

- Prefix: `uv-`
- Pattern: block + modifiers (`.uv-button`, `.uv-button--secondary`)
- State helpers: `.uv-is-selected`, `.uv-is-disabled`, `.uv-state-error`, etc.

## Core primitives

- Layout: `.uv-tool-page`, `.uv-tool-page__content`, `.uv-stack`, `.uv-inline`, `.uv-grid`, `.uv-split`, `.uv-container`
- Inputs: `.uv-input`, `.uv-select`, `.uv-textarea`, `.uv-check`, `.uv-radio`, `.uv-switch`
- Actions: `.uv-button`, `.uv-icon-button`
- Surfaces: `.uv-card`, `.uv-panel`, `.uv-alert`, `.uv-empty-state`, `.uv-result`
- Data display: `.uv-list`, `.uv-list__row`, `.uv-badge`, `.uv-tabs`, `.uv-tab`
- Overlays/loading: `.uv-modal`, `.uv-dialog`, `.uv-dropdown`, `.uv-popover`, `.uv-spinner`, `.uv-skeleton`

## Theming

- Theme tokens come from host-provided `--uv-*` variables.
- Plugins must not redefine `--uv-*` tokens.
- Light/dark support is automatic through host token injection and `data-uv-theme` handling.

## Accessibility defaults

- Visible focus style via `:focus-visible`
- Reduced-motion handling via `prefers-reduced-motion`
- Contrast-safe defaults based on semantic token aliases

## Custom CSS guidance

- Prefer framework classes first.
- Keep plugin-specific CSS to narrow, app-specific behavior.
- Target roughly <=10% custom styling for standard tools.
