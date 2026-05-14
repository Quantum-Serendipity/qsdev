# libzim - Reference Implementation of ZIM Specification
> Source: https://github.com/openzim/libzim
> Retrieved: 2026-05-14

## Overview
Libzim is "the reference implementation for the ZIM file format," a software library enabling read/write operations for ZIM files across multiple systems and architectures. The project is part of the openZIM initiative focused on offline content distribution.

## Key Statistics
- **Stars:** 236
- **Forks:** 68
- **Language Composition:** C++ (95.4%), C (2.4%), Meson (1.2%), Python (1.0%)
- **Latest Release:** 9.7.0 (May 9, 2026)
- **License:** GPLv2 or later
- **Repository:** openzim/libzim on GitHub

## Core Features & Capabilities

**Primary Functions:**
- Read and write ZIM file format implementations
- Search API capabilities (requires Xapian compilation)
- Multi-platform support with cross-compilation options
- Compression support via LZMA and Zstd

**Supported Compression Methods:**
The library integrates LZMA and Zstd compression technologies for file optimization.

## Dependencies

**Required Libraries:**
- LZMA (liblzma-dev)
- ICU (libicu-dev)
- Zstd (libzstd-dev)
- Xapian (libxapian-dev, optional)

**Development/Testing:**
- Google Test framework
- ZIM Testing Suite (openzim/zim-testing-suite)

**Documentation Build:**
- Doxygen, Sphinx, sphinx_rtd_theme, Breathe, Exhale

## Build Instructions

**Build System:** Meson (v0.43+) with Ninja

**Standard Compilation:**
```
meson . build
ninja -C build
```

**Static Linking:**
Add `--default-library=static` option to Meson command.

**Build Without Xapian:**
```
meson . build -Dwith_xapian=false
```

**Documentation:**
```
meson . build -Ddoc=true
ninja -C build doc
```

## Testing Framework

The project includes unit tests requiring external ZIM test datasets. Test execution workflow:

```
meson . build
cd build
ninja
ninja download_test_data
meson test
```

**Notable Test Considerations:**
- Some tests require up to 16GB memory (skip with `SKIP_BIG_MEMORY_TEST=1`)
- Multithreaded error detection tests may need timing adjustments via `WAIT_TIME_FACTOR_TEST`

## Installation & Distribution

**Installation:**
```
ninja -C build install
```

**Uninstallation:**
```
ninja -C build uninstall
```

**Pre-compiled Binaries:** Available at download.openzim.org/release/libzim/ for multiple platforms including macOS Homebrew.

## Platform Support
- POSIX systems (primarily GNU/Linux, tested on Ubuntu and Fedora)
- Microsoft Windows (requires careful compiler configuration)
- Cross-compilation supported across architectures

## Search Capabilities
Libzim provides search API functionality when compiled with Xapian support. The `LIBZIM_WITH_XAPIAN` define indicates Xapian compilation status in installed versions.
