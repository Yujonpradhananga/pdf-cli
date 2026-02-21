# Dual Page View Design

## Summary

Add a 2-page view mode that displays two consecutive pages simultaneously, with two layout options (vertical stacking and horizontal side-by-side). Navigation flips by one page by default, with Shift+Arrow for 2-page jumps.

## State

- New field `dualPageMode string` on `DocumentViewer`: `""` (off) → `"vertical"` → `"horizontal"` → back to `""`
- Key `2` cycles through the three states
- Second page displayed is always `currentPage + 1`

## Navigation

- Arrows / j / k / Space: flip by 1 page (2-3 → 3-4 → 4-5)
- Shift+Left / Shift+Right: flip by 2 pages (2-3 → 4-5 → 6-7), only in 2-page mode
- At the last page, show single page if no next page exists

## Rendering

- Both pages always rendered as images (no text/mixed mode in dual view)
- Two independent image renders using existing `savePageAsImage` + `renderWithTermImg` pipeline
- Vertical: each page gets ~half terminal height, full width
- Horizontal: each page gets half terminal width, full height
- Dark mode, zoom, and fit mode apply to both pages identically
- Kitty synchronized update wraps both renders

## Status Bar

- Shows `Page X-(X+1)/Z` when in dual mode
- Includes `[2pg-v]` or `[2pg-h]` indicator

## Approach

Two independent image renders (Option A). Each page rendered as a separate terminal image at half-dimensions. Reuses existing rendering pipeline with minimal changes.
