# I Changed My Mind - NixOS is NOT the Best Linux
- **Source URL**: https://dev.to/jasper-clarke/i-changed-my-mind-nixos-is-not-the-best-linux-1cpj
- **Retrieved**: 2026-03-20
- **Type**: Blog post (DEV Community)

## Author
Jasper (handle: @jasper-clarke)

## Duration with NixOS
One year

## Primary Use Case
General system administration and development

## Reasons for Switching

### Core Development Problem
Jasper struggled significantly with development workflows requiring external libraries. He specifically cited difficulties with C, Java, and Rust development. The fundamental issue: "NixOS makes it very difficult to simply, install a library and have it accessible to the rest of your system."

### Specific Technical Complaint
Setting up an OpenGL environment demonstrated the friction. On Arch, a single command (`pacman -S gcc cmake make glfw glew libglv`) sufficed. On NixOS, he needed to manually configure `LD_LIBRARY_PATH` variables and still couldn't achieve functionality. This transformed what should be a quick learning project into a frustrating setup nightmare, causing him to abandon attempts entirely.

### Burnout Factor
The setup overhead created psychological barriers: "It turns a simple idea...to, 'I literally don't understand how to even set this up.'"

## The Switch
Jasper adopted a **hybrid approach** rather than complete abandonment. He maintains NixOS for system administration but uses an Arch Linux virtual machine specifically for graphics and library-dependent development work. Arch's straightforward package management enabled him to establish development environments in approximately 30 seconds.

## Nuanced Take
His conclusion acknowledges trade-offs: "Arch makes the operating system tedious, but the development environment easy. NixOS makes the operating system easy, but the development environment torture."

He doesn't claim NixOS is fundamentally flawed — only that it wasn't optimal for his particular development workflow.
