<!-- Source: https://blog.jonsdocs.org.uk/2020/11/14/moving-from-nixos-to-ubuntu/ -->
<!-- Retrieved: 2026-03-20 -->

# Moving from NixOS to Ubuntu

**Author:** Jonathan
**Date:** November 14, 2020
**Duration of NixOS Use:** Several years (specific timeframe not stated)

## What He Liked About NixOS
Jonathan appreciated NixOS's declarative configuration system, noting that "the entire environment can be declared in a configuration file." Key benefits included:
- Ability to reproduce identical environments across multiple devices
- NixOS containers that eliminated the need to run development servers on his host machine
- The rollback feature: "the ability to switch back to an older system state" provided peace of mind when using the unstable channel

## Pain Points & Reasons for Switching
Jonathan encountered persistent compatibility issues:
- Brasero (optical media software) failed despite correct permissions
- TeamViewer and Zoom produced terminal errors
- Steam couldn't display his games library or access community features due to broken GLib dependencies
- The Nixpkgs repository was overwhelmed with 4,400+ issues and insufficient maintainers

He also noted NixOS's niche status made finding solutions difficult and felt he needed "his system to work out of the box."

## The Switch to Ubuntu
After a failed attempt with Manjaro (six installation failures), Jonathan chose Ubuntu. His assessment: applications "worked straight away" and he successfully upgraded from 20.04 LTS to 20.10 without issues. Future plans include learning Docker to replicate his container workflow.
