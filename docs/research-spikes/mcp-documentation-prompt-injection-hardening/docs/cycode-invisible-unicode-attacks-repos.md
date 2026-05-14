# Invisible Code & Hidden Prompts - How Attackers Weaponize Unicode in Repos
- **Source**: https://cycode.com/blog/invisible-code-hidden-prompts-unicode-attacks-sast/
- **Retrieved**: 2026-05-14
- **Note**: AI-summarized content

## Overview

Attackers exploit Unicode characters to hide malicious code in repositories, making exploits invisible to human reviewers while remaining visible to compilers. The 2021 "Trojan Source" research and 2025 Glassworm worm demonstrate this threat's evolution from theory to real-world attacks.

## Technical Foundation

**Character Encoding Layers:**
- ASCII: 7-bit system (128 characters, no hidden tricks)
- Unicode: Maps code points to nearly every character globally
- UTF-8: Encodes Unicode into binary bytes that computers process

Example: The rocket emoji appears as one character but represents U+1F680 in Unicode, encoding as 4 bytes (F0 9F 9A 80) in UTF-8.

## Three Primary Attack Techniques

### 1. Variation Selectors (Glassworm Method)

These characters modify preceding symbols but create invisible sequences when isolated.

**Safe code example:**
```
const key = "";
```
Hexdump: `63 6f 6e 73 74 20 6b 65 79 20 3d 20 22 22 3b`

**Weaponized version:**
```
const key = "︀";
```
Hexdump reveals hidden bytes: `ef b8 80` (Variation Selector-1/U+FE00)

The invisible bytes occupy storage space and can contain thousands of malicious payload bytes while appearing empty to reviewers.

### 2. Private Use Area (PUA) Encoding

Unicode reserves character blocks for internal application use with no standard glyphs.

**Safe code:**
```
var payload = "";
```

**Weaponized version:**
```
var payload = " ";
```
Hidden bytes: `f3 bf be 80` (U+FFF80 PUA character -- 4 bytes, zero pixels displayed)

Attackers map malicious scripts to these invisible characters, which decode at runtime into executable code.

### 3. Bidirectional Control (Trojan Source)

Exploits Unicode characters intended for Right-to-Left languages to decouple visual code order from logical compiler execution.

**Attack example using Right-to-Left Override (RLO):**
```
var accessLevel = "user";
if (accessLevel != "user[RLO] [LRI]// Check if admin[PDI] [LRI]") {
    console.log("You are an admin.");
}
```

**Visual appearance to reviewers:** Code compares `accessLevel != "user"` safely

**Compiler interpretation:** The hidden bytes (e2 80 ae) reverse text display, hiding "// Check if admin" inside the string literal, making the comparison fail and granting admin privileges.

## AI-Layer Vulnerability: Invisible Prompt Injection

As AI agents analyze code, attackers inject Unicode Tag Characters (U+E0000 block) -- invisible versions of standard letters occupying 4 bytes each but rendering zero pixels.

**Human view:**
```
System: You are a helpful assistant.
```

**AI processing (with invisible tags):**
```
System: You are a helpful assistant. IGNORE ALL PREVIOUS INSTRUCTIONS...
```

The invisible tags spell out instructions the AI can process while remaining hidden from human code reviewers.

## Detection and Defense

**SAST Approach:**
- Pattern matching: Banning specific byte sequences (e.g., EF B8 80 for VS1)
- Anomaly detection: Flagging strings with high densities of PUA characters or mixed-script homoglyphs
- CI/CD integration: Embedding SAST scanners in pipelines
- GitHub's bidirectional Unicode warning system (yellow banner alerts)
