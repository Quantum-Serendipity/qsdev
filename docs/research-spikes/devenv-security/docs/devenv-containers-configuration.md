# devenv.sh Containers Configuration
- **Source**: https://devenv.sh/containers/
- **Retrieved**: 2026-05-12

## Container Runtime Support

The documentation does not explicitly list supported runtimes. However, it mentions using Docker for local testing and references OCI (Open Container Initiative) standards: "Generate an OCI container from your development environment."

The tool uses skopeo for copying containers to registries, which is container-agnostic.

## Isolation Capabilities

The page does not provide detailed information about isolation mechanisms, security policies, or resource constraints for containers.

## Available Configuration Options

The documentation references "the list of all container options" but doesn't detail them in this page. Key configurable aspects mentioned include:

- **Container naming and identification** (`shell`, `processes`, custom names)
- **Startup commands**: "You can specify the command to run when the container starts"
- **Content inclusion**: `copyToRoot` parameter controls what files are included in the image
- **Registry settings**: `registry` and `defaultCopyArgs` for pushing to registries
- **Conditional builds**: Use `config.container.isBuilding` to customize environments based on build context

## Image Building Process

Build mechanism: "Use `devenv container build <name>` to generate an OCI container from your development environment."

The process leverages Nix and requires a Linux builder on macOS systems. Images are generated from `devenv.nix` configurations containing processes, packages, and other environment definitions.

Note: Security-specific options (networking, volumes, credentials) aren't detailed in this excerpt.
