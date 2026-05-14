<!-- Source: https://karl-voit.at/2025/08/30/end-of-my-nixos/ -->
<!-- Retrieved: 2026-03-20 -->

# Good bye NixOS, Hello Debian (Again)!

**Author:** Karl Voit (public voit)
**Date:** August 30, 2025
**Duration of NixOS Use:** Approximately 2 years (started September 2023)

## Author's Background
Nearly three decades of GNU/Linux experience. Maintains strong opinions about tool selection, particularly favoring Xfce and conservative software choices over trendy alternatives.

## Primary Pain Points with NixOS

### Steep Learning Curve
The author emphasizes that NixOS demands becoming a "Nix wizard" -- requiring months or even lifetime commitment to master. They note that "very, very basic and easy things" required consulting experts rather than solving independently.

### Python Compatibility Crisis
A major frustration involved running Python scripts. The author states that "NixOS and Python are a no go" due to complex dependency management with C libraries like NumPy. Even functional Nix shells "broke after a Python package upgrade," and simple four-line scripts became 50+ line configurations with no sustainable solution.

### Configuration Limitations
Setting Xfce preferences involved tedious workarounds using xfconf.settings with inconsistent syntax (True/False vs 0/1), and many settings produced no effect despite correct configuration.

### Documentation Issues
The author criticizes extensive outdated documentation, making it impossible to rely on online resources without deep expertise.

### System Storage
NixOS required 30GB for base installation versus 10GB for typical Debian setups.

### Final Failure
A firmware update via fwupdmgr created an unrecoverable boot loop, rendering NixOS's vaunted rollback feature useless.

## Current Setup
The author replaced NixOS with Debian 13 Trixie paired with GNOME 48 on Wayland, noting they're "productive" though adjusting to GNOME's slower performance compared to Xfce.

## Overall Assessment
The author concludes NixOS "provided me solutions to problems I never had" while introducing manageable issues into solvable ones, calling it "one of my worst IT ideas so far."
