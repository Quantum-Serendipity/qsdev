<!-- Source: https://determinate.systems/blog/we-want-to-make-nix-better/ -->
<!-- Retrieved: 2026-03-20 -->

# We Want to Make Nix Better (Determinate Systems)

**Author:** Graham Christensen, founder of Determinate Systems

## Acknowledged Problems with Nix

The article identifies several barriers to Nix adoption:

- **Steep learning curve:** The expression language feels unfamiliar to most developers, and core concepts lack digestibility
- **Paradigm shift:** The package manager operates differently from familiar tools like apt, Homebrew, or deb
- **High investment cost:** Learning and using Nix demands substantial time and energy, making organizational adoption difficult

## User Community Pain Points

Christensen recognizes that even experienced developers question whether Nix's benefits justify the effort. Organizations face a rational calculus: existing tools are "good enough," so committing resources to Nix adoption feels risky. As the article notes, people want Nix's features -- "hermetic development environments, fully reproducible package builds, declaratively configured operating systems" -- without the friction.

## Specific Improvements Being Pursued

The article highlights external dependencies as a concrete problem area. The authors cite real examples: Protobuf compiler dependencies in Rust projects, Nokogiri in Ruby, and OpenSSL across languages. While Nix already solves this through development environments, the solution requires writing Nix code and consulting documentation -- barriers for non-Nix users.

## Current State of Adoption Barriers

Determinate Systems frames Nix as powerful but positioned as a developer tool requiring specialized knowledge. Their mission involves making Nix's benefits accessible without requiring users to directly engage with its complexity, allowing "Nix quietly do its vital work in the background."
