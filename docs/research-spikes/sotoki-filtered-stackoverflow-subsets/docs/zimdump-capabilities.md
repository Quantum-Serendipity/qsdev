<!-- Source: https://raw.githubusercontent.com/openzim/zim-tools/main/src/zimdump.cpp -->
<!-- Retrieved: 2026-05-14 -->

# zimdump Operations and Features

## Supported Operations

Based on the usage documentation, zimdump supports four main commands:

1. **list** - Enumerate entries with optional details
2. **dump** - Extract contents to filesystem
3. **show** - Display a single entry's content
4. **info** - Print archive metadata

## Dump Capabilities

Yes, zimdump can dump all entries to a directory. The `dump` command accepts `--dir=DIR` to specify the output location and will process the entire archive by default.

## Filtering Options

The tool supports filtering by:

- **Namespace** (`--ns N`) - Restricts operations to a specific namespace
- **URL/Path** (`--url URL`) - Targets a specific article
- **Index** (`--idx INDEX`) - Selects entry by numeric position

The namespace filter defaults to "A" when used with `--url`, or applies no filter during dump operations if omitted.

## Dump Modes

Two extraction modes exist:

1. **Symlink mode** (`--redirect`) - Creates symbolic links for redirect entries (Unix/Linux only)
2. **HTML redirect mode** (default) - Generates HTML redirect files instead of symlinks; always used on Windows

The tool also supports detailed listing output via the `--details` flag and returns specific exit codes (0 for success, 1 for entry mismatches, 2 for errors during dump operations).

## Key Limitation for Filtering Use Case

zimdump can only filter by namespace, specific URL, or index position. It has NO content-based filtering (e.g., by tags embedded in HTML). To filter a SO ZIM by tags, you would need to:
1. Dump ALL entries to disk
2. Parse each HTML file to determine its tags
3. Delete unwanted entries
4. Repack with zimwriterfs

This is extremely inefficient for a 75 GB ZIM file.
