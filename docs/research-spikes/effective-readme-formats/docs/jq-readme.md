<!-- Source: https://raw.githubusercontent.com/jqlang/jq/master/README.md -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Full content - jq has a short README. -->

# jq

`jq` is a lightweight and flexible command-line JSON processor akin to `sed`,`awk`,`grep`, and friends for JSON data. It's written in portable C and has zero runtime dependencies, allowing you to easily slice, filter, map, and transform structured data.

## Documentation

- **Official Documentation**: [jqlang.org](https://jqlang.org)
- **Try jq Online**: [play.jqlang.org](https://play.jqlang.org)

## Installation

### Prebuilt Binaries

Download the latest releases from the [GitHub release page](https://github.com/jqlang/jq/releases).

### Docker Image

Pull the [jq image](https://github.com/jqlang/jq/pkgs/container/jq) to start quickly with Docker.

#### Run with Docker

##### Example: Extracting the version from a `package.json` file

```bash
docker run --rm -i ghcr.io/jqlang/jq:latest < package.json '.version'
```

##### Example: Extracting the version from a `package.json` file with a mounted volume

```bash
docker run --rm -i -v "$PWD:$PWD" -w "$PWD" ghcr.io/jqlang/jq:latest '.version' package.json
```

### Building from source

#### Dependencies

- libtool
- make
- automake
- autoconf

#### Instructions

```console
git submodule update --init    # if building from git to get oniguruma
autoreconf -i                  # if building from git
./configure --with-oniguruma=builtin
make clean                     # if upgrading from a version previously built from source
make -j8
make check
sudo make install
```

Build a statically linked version:

```console
make LDFLAGS=-all-static
```

##### Cross-Compilation

For details on cross-compilation, check out the GitHub Actions file and the cross-compilation wiki page.

## Community & Support

- Questions & Help: Stack Overflow (jq tag)
- Chat & Community: Discord
- Wiki & Advanced Topics: GitHub Wiki

## License

`jq` is released under the MIT License. Documentation under Creative Commons CC BY 3.0. Uses parts of "decNumber" C library under ICU License.
