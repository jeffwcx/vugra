# CSS Profile v1

Vugra templates use HTML syntax, but Vugra does not implement browser DOM or
full browser CSS semantics. CSS Profile v1 is the first renderer-neutral style
contract shared by the parser, layout model, render commands, native Vello path,
and wasm path.

## Selectors

Supported now:

- Class selectors such as `.counter`.

Planned in this profile, but not yet implemented by the Go style resolver:

- Element selectors.
- ID selectors.
- Simple descendant selectors.

Unsupported selectors are diagnostics. Complex combinators, pseudo-classes,
pseudo-elements, media queries, custom properties, and cascade features outside
this document are not part of v1.

## Properties

The parser accepts and preserves declarations for these v1 properties:

- Box model: `display`, `box-sizing`, `width`, `height`, `min-width`,
  `min-height`, `max-width`, `max-height`, `margin`, `padding`.
- Flex: `flex-direction`, `flex-wrap`, `flex`, `flex-grow`, `flex-shrink`,
  `flex-basis`, `align-items`, `align-self`, `justify-content`, `gap`,
  `row-gap`, `column-gap`.
- Visuals: `background`, `background-color`, `color`, `opacity`.
- Border: `border`, `border-width`, `border-color`, `border-style`,
  `border-radius`.
- Text: `font-family`, `font-size`, `font-weight`, `line-height`,
  `text-align`, `white-space`.
- Overflow: `overflow`, `overflow-x`, `overflow-y`.

The current layout MVP also keeps deterministic grid support for existing
examples and tests: `grid-template-columns`, `grid-template-rows`,
`grid-auto-rows`, `grid-column`, and `grid-row`.

Lengths currently support `px` for fixed sizes and `%` for `width`/`height`
against the parent layout size.

Vugra CSS Profile v1 defaults to UI-framework sizing: explicit `width` and
`height` describe the element's outer layout box. `box-sizing: border-box`
therefore preserves that default behavior explicitly. `box-sizing: content-box`
is also supported; in that mode explicit `width` and `height` describe the
content box and padding expands the outer layout box.

Unsupported properties produce `style.unsupported_property` warnings and are
explicit no-ops.

## Geometry

Layout boxes, render commands, Vello ops, clip rectangles, scroll offsets, and
hit-test rectangles use `float32` coordinates. Integer rounding belongs only at
software bitmap edges or host APIs that require device pixels.

## Text

Text commands carry line boxes and glyph-run placement data. Renderers may still
fall back to the plain text string while a backend-specific shaper is maturing,
but the Vugra contract is line/glyph data, not a measurement-only width and
height.
