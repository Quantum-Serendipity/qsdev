# MCP JSON Schema: Key Type Definitions (2025-11-25)

- **Source**: https://raw.githubusercontent.com/modelcontextprotocol/specification/refs/heads/main/schema/2025-11-25/schema.json
- **Retrieved**: 2026-05-14

## CallToolResult

```json
{
  "description": "The server's response to a tool call.",
  "properties": {
    "_meta": {
      "additionalProperties": {},
      "description": "See [General fields: `_meta`]",
      "type": "object"
    },
    "content": {
      "description": "A list of content objects",
      "items": {"$ref": "#/$defs/ContentBlock"},
      "type": "array"
    },
    "isError": {
      "description": "Whether the tool call ended in an error.",
      "type": "boolean"
    },
    "structuredContent": {
      "additionalProperties": {},
      "description": "Optional JSON object",
      "type": "object"
    }
  },
  "required": ["content"],
  "type": "object"
}
```

## Annotations

```json
{
  "description": "Optional annotations for the client.",
  "properties": {
    "audience": {
      "description": "Describes the intended audience",
      "items": {"$ref": "#/$defs/Role"},
      "type": "array"
    },
    "lastModified": {
      "description": "ISO 8601 formatted timestamp",
      "type": "string"
    },
    "priority": {
      "description": "Importance for server operation (0-1)",
      "maximum": 1,
      "minimum": 0,
      "type": "number"
    }
  },
  "type": "object"
}
```

## TextContent (with _meta on individual content items)

```json
{
  "properties": {
    "_meta": {
      "additionalProperties": {},
      "type": "object"
    },
    "annotations": {
      "$ref": "#/$defs/Annotations"
    },
    "text": {
      "type": "string"
    },
    "type": {
      "const": "text",
      "type": "string"
    }
  },
  "required": ["text", "type"],
  "type": "object"
}
```

## Key Finding: _meta Extensibility

The `_meta` field appears on BOTH the CallToolResult envelope AND on individual content items (TextContent, ImageContent, AudioContent, EmbeddedResource). It is defined inline as:

```json
{
  "additionalProperties": {},
  "type": "object"
}
```

This permits any additional properties — it is a fully open extensible container. No properties are mandated. This is the mechanism through which vendor-specific metadata (like Anthropic's `anthropic/maxResultSizeChars`) is conveyed.

Custom provenance metadata could be placed here without violating the MCP specification.
