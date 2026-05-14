# MCP JSON Schema: Key Type Definitions (2025-03-26)

- **Source**: https://raw.githubusercontent.com/modelcontextprotocol/specification/refs/heads/main/schema/2025-03-26/schema.json
- **Retrieved**: 2026-05-14

## CallToolResult

Purpose: Server response to tool invocation

Properties:
- `content` (required): Array of TextContent | ImageContent | AudioContent | EmbeddedResource
- `isError` (optional): boolean - "Whether the tool call ended in an error"
- `_meta` (optional): object with additionalProperties allowed

## Content Types

### TextContent
- `type` (required): const "text"
- `text` (required): string
- `annotations` (optional): Annotations object

### ImageContent
- `type` (required): const "image"
- `data` (required): string (base64-encoded)
- `mimeType` (required): string
- `annotations` (optional): Annotations object

### AudioContent
- `type` (required): const "audio"
- `data` (required): string (base64-encoded)
- `mimeType` (required): string
- `annotations` (optional): Annotations object

### EmbeddedResource
- `type` (required): const "resource"
- `resource` (required): TextResourceContents | BlobResourceContents
- `annotations` (optional): Annotations object

## Annotations Type

Purpose: "Optional annotations for the client"

Properties:
- `audience` (optional): array of Role enum values
- `priority` (optional): number (0-1 range, where "1 means most important")

## Key Extensibility Feature

All result types include `_meta` field with `additionalProperties: true`. This is explicitly designed to support "clients and servers to attach additional metadata."

This is the primary mechanism through which custom provenance metadata could be conveyed, though no standard fields are defined for this purpose.
