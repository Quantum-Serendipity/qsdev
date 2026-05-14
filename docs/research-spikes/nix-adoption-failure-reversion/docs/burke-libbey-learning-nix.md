<!-- Source: https://notes.burke.libbey.me/learning-nix/ -->
<!-- Retrieved: 2026-03-20 -->

# Learning Nix - Burke Libbey

## Overview
The page documents a work-in-progress learning strategy for Nix, dated December 17, 2018. It outlines Burke Libbey's sequential approach to mastering the Nix package manager and NixOS.

## Learning Strategy (Seven Steps)

1. **Install Nix on macOS** - Starting point for hands-on experience
2. **Manual exploration** - Read through `man nix-env` and play around with most of the options. Skim the other `man nix-*`
3. **Mental model building** - Explore configuration directories (~/.nix-profile, ~/.nix-channels, ~/.nix-defexpr, /nix/var/nix/profiles)
4. **Interactive learning** - Complete the nixcloud.io tour ("the last handful of exercises might break your brain" if unfamiliar with functional programming)
5. **Practical NixOS deployment** - Set up a DigitalOcean instance, recommending 4GB RAM over 1GB
6. **Structured learning** - Work through Nix Pills documentation (later sections become "kind of... arcane")
7. **Source code exploration** - Clone nixpkgs repository and navigate the codebase starting with default.nix

## Note
This page contains no information about Shopify experience or adoption challenges — only technical learning guidance. Published December 2018, which places it before the formal 2019 Nix adoption attempt at Shopify.
