<!-- Source: https://github.com/t3-oss/create-t3-app and https://create.t3.gg/en/faq -->
<!-- Retrieved: 2026-05-12 -->

# create-t3-app: Overview and FAQ

## Purpose
CLI tool to quickly scaffold a full-stack, typesafe Next.js application. Users run `npm create t3-app@latest` and answer interactive prompts to customize their setup.

## Core Technologies
The T3 Stack includes Next.js, tRPC, Tailwind CSS, TypeScript, Prisma, Drizzle, and NextAuth.js. These form the foundation, though each component is optional based on user needs.

## Design Philosophy: Three Axioms

**1. Solve Specific Problems**
The project deliberately avoids bloat. It excludes general-purpose state libraries but includes solutions like NextAuth.js that address concrete challenges within the core stack.

**2. Responsible Innovation**
Embrace newer technologies where appropriate. "We **wouldn't** bet on risky new database tech (SQL is great!). But we **happily** bet on tRPC since it's just functions that are trivial to move off."

**3. Typesafety as Non-Negotiable**
Full-stack type safety across the entire application is central, not optional.

## Key Design Decisions

- **Scaffolding tool, not a framework** — Once initialized, the codebase belongs to the user, with no post-install CLI to maintain updates automatically
- **Modular selection** — Developers select which pieces they need; the CLI constructs their setup accordingly
- **No prescribed solutions** — Rejects prescribing solutions for state management or deployment strategies
- **Start building, learn along the way** — Rather than extensive prerequisite learning, assumes users have familiarity with some stack components

## What This Means for gdev

The T3 pattern is: strongly opinionated about what technologies to include, but flexible about which subset the user actually wants. Each optional component is well-integrated when selected, but cleanly absent when not. This is a "curated menu" rather than "blank canvas" approach.
