<!-- Source: https://shields.io/badges/endpoint-badge -->
<!-- Retrieved: 2026-05-12 -->

# Shields.io Endpoint Badge Documentation

## Overview
The endpoint badge allows you to provide badge content through a JSON endpoint, with configurable cache behavior. Shields fetches the JSON and formats it into a badge.

## Required Query Parameter
- **`url`** (string): The URL to your JSON endpoint

## JSON Schema Response

Your endpoint must return a JSON object with these properties:

| Property | Required | Default | Description |
|----------|----------|---------|-------------|
| `schemaVersion` | Yes | -- | Must always be `1` |
| `label` | Yes | -- | Left-side text; use empty string to omit |
| `message` | Yes | -- | Right-side text (cannot be empty) |
| `color` | No | lightgrey | Right-side color (named colors, hex, rgb/rgba, hsl/hsla, CSS colors) |
| `labelColor` | No | grey | Left-side color |
| `isError` | No | false | Set `true` for error badges; prevents color overrides |
| `namedLogo` | No | -- | Simple-icons slug for logo |
| `logoSvg` | No | -- | Custom SVG logo string |
| `logoColor` | No | -- | Logo color (simple-icons only) |
| `logoSize` | No | -- | Set to `auto` for adaptive resizing |
| `style` | No | flat | Template: flat, flat-square, plastic, for-the-badge, social |

## Optional Query Parameters

- **`style`**: Badge template style
- **`logo`**: Simple-icons slug
- **`logoColor`**: Logo color (simple-icons only)
- **`logoSize`**: Set to `auto` for wider logos
- **`label`**: Override left text (URL-encode spaces/special characters)
- **`labelColor`**: Left background color
- **`color`**: Right background color
- **`cacheSeconds`**: HTTP cache lifetime (defaults applied per-badge; values below minimum ignored)
- **`link`**: Specify click behavior (works with `<object>` HTML tags only)

## Example

**Endpoint returns:** `{ "schemaVersion": 1, "label": "hello", "message": "sweet world", "color": "orange" }`

**Badge URL:** `https://img.shields.io/badge/hello-sweet_world-orange`
