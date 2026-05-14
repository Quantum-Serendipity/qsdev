# MCP Specification: Tools (2025-11-25)

- **Source**: https://modelcontextprotocol.io/specification/2025-11-25/server/tools
- **Retrieved**: 2026-05-14

## CallToolResult Structure

Tool results may contain **structured** or **unstructured** content.

**Unstructured** content is returned in the `content` field of a result, and can contain multiple content items of different types.

All content types (text, image, audio, resource links, and embedded resources) support optional annotations that provide metadata about audience, priority, and modification times. This is the same annotation format used by resources and prompts.

### Text Content

```json
{
  "type": "text",
  "text": "Tool result text"
}
```

### Image Content

```json
{
  "type": "image",
  "data": "base64-encoded-data",
  "mimeType": "image/png",
  "annotations": {
    "audience": ["user"],
    "priority": 0.9
  }
}
```

### Audio Content

```json
{
  "type": "audio",
  "data": "base64-encoded-audio-data",
  "mimeType": "audio/wav"
}
```

### Resource Links

A tool MAY return links to Resources, to provide additional context or data:

```json
{
  "type": "resource_link",
  "uri": "file:///project/src/main.rs",
  "name": "main.rs",
  "description": "Primary application entry point",
  "mimeType": "text/x-rust"
}
```

### Embedded Resources

Resources MAY be embedded to provide additional context or data:

```json
{
  "type": "resource",
  "resource": {
    "uri": "file:///project/src/main.rs",
    "mimeType": "text/x-rust",
    "text": "fn main() {\n    println!(\"Hello world!\");\n}",
    "annotations": {
      "audience": ["user", "assistant"],
      "priority": 0.7,
      "lastModified": "2025-05-03T14:30:00Z"
    }
  }
}
```

### Structured Content

**Structured** content is returned as a JSON object in the `structuredContent` field of a result. For backwards compatibility, a tool that returns structured content SHOULD also return the serialized JSON in a TextContent block.

### Output Schema

Tools may provide an output schema for validation of structured results. If an output schema is provided:
- Servers MUST provide structured results that conform to this schema
- Clients SHOULD validate structured results against this schema

## Tool Definition Fields

- `name`: Unique identifier for the tool
- `title`: Optional human-readable name
- `description`: Human-readable description of functionality
- `inputSchema`: JSON Schema defining expected parameters
- `outputSchema`: Optional JSON Schema defining expected output structure
- `annotations`: Optional properties describing tool behavior
- `execution`: Optional object describing execution-related properties

## Key Trust/Security Notes

> For trust & safety and security, clients MUST consider tool annotations to be untrusted unless they come from trusted servers.

Security Considerations:
1. Servers MUST validate all tool inputs, implement proper access controls, rate limit tool invocations, sanitize tool outputs
2. Clients SHOULD prompt for user confirmation on sensitive operations, show tool inputs to the user before calling the server, validate tool results before passing to LLM, implement timeouts, log tool usage

## _meta Field (from JSON Schema)

CallToolResult includes an optional `_meta` field with `additionalProperties: true`, which supports clients and servers attaching additional metadata.

## Annotations on Content

Standard annotations available on content items:
- `audience`: array of Role enum values (e.g., ["user"], ["assistant"], ["user", "assistant"])
- `priority`: number (0-1 range, where 1 means most important)
- `lastModified`: timestamp (on embedded resources)

No custom/arbitrary annotation fields are specified in the standard — only audience, priority, and lastModified.
