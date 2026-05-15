<!-- Source: https://www.microsoft.com/en-us/security/blog/2026/05/07/prompts-become-shells-rce-vulnerabilities-ai-agent-frameworks/ -->
<!-- Retrieved: 2026-05-15 -->
<!-- Note: Content returned via WebFetch AI summary -->

# When Prompts Become Shells: RCE Vulnerabilities in AI Agent Frameworks

## Overview

Microsoft's security research team identified critical vulnerabilities in Semantic Kernel, an open-source AI agent framework with over 27,000 GitHub stars. These flaws demonstrate how prompt injection can escalate into host-level remote code execution when AI models are connected to system tools.

## Core Vulnerability Concept

The fundamental issue stems from a shifted threat model: "By equipping these models with plugins (also called tools), your agents no longer just generate text; they now read files, search connected databases, run scripts, and perform other tasks to actively operate on your network."

The researchers emphasize that "The AI model itself isn't the issue as it's behaving exactly as designed by parsing language into tool schemas. The vulnerability lies in how the framework and tools trust the parsed data."

## CVE-2026-26030: In-Memory Vector Store Exploitation

### Attack Mechanism

This vulnerability exploited an unsafe string interpolation pattern in filter functions. The framework used `eval()` to execute Python lambda expressions without proper sanitization, creating a classic injection sink.

When a user queries—for example, "Find hotels in Paris"—the system generates: `lambda x: x.city == 'Paris'`. An attacker could inject malicious Python code by providing input like `' or MALICIOUS_CODE or '`, transforming the filter into executable payload.

### Bypass Technique

The developers had implemented a blocklist validator checking for dangerous identifiers like `eval`, `exec`, and `open`. However, the researchers discovered multiple bypass methods:

- **Attribute traversal**: Using `__name__`, `load_module`, and `system` attributes absent from the blocklist
- **Class hierarchy crawling**: Accessing `BuiltinImporter` to dynamically load the `os` module
- **AST validation gaps**: The structural check only validated lambda expressions but didn't check `ast.Subscript` nodes, allowing bracket notation access to blocked names

The resulting payload could launch arbitrary commands (like `calc.exe`) without triggering the security filters.

### Mitigation

Microsoft implemented four-layer protection:
- AST node-type allowlists permitting only safe constructs
- Function call validation ensuring only safe functions execute
- Dangerous attributes blocklist preventing class hierarchy traversal
- Name node restrictions limiting identifiers to lambda parameters

## CVE-2026-25592: Sandbox Escape via SessionsPythonPlugin

### The Container Boundary Problem

Semantic Kernel's `SessionsPythonPlugin` allows agents to execute Python in isolated Azure Container Apps sandboxes. The security model depends entirely on this boundary—code runs isolated with separate filesystems.

### Vulnerability Details

The `DownloadFileAsync` function was accidentally marked with a `[KernelFunction]` attribute, exposing it to the AI model as a callable tool. This parameter—`localFilePath`—dictated where files were written on the host device with "no path validation, directory restriction, or sanitization in place."

### Attack Chain

**Step 1**: An injected prompt instructs the agent's `ExecuteCode` tool to generate malicious scripts within the sandbox container.

**Step 2**: A second prompt tells the model to invoke `DownloadFileAsync`, writing the payload to the host's `Windows\Start Menu\Programs\Startup` folder.

**Step 3**: Upon next user login, the startup script executes, compromising the host completely.

### Defense Approach

The fix involved removing the `[KernelFunction]` attribute entirely, making the function invisible to the AI model. Additional safeguards included path canonicalization using `Path.GetFullPath()` and allowlist matching for developers calling the function directly.

## Broader Security Implications

The research reveals that these aren't isolated bugs but architectural patterns across agent frameworks. Key insights include:

- "Your LLM is not a security boundary. The tools you expose define your attacker's affected scope. Any tool parameter the model can influence must be treated as attacker-controlled input."

- Security requires dual-layer monitoring: AI-level intent detection plus host-level execution detection through endpoint telemetry

- When models connect to system tools, "prompt injection risks may extend beyond typical chatbot misuse and require additional safeguards"

## Practical Recommendations

Organizations should:
1. Upgrade Semantic Kernel to version 1.39.4+ (Python) or 1.71.0+ (.NET)
2. Hunt for post-exploitation signals during the vulnerable window using advanced queries
3. Monitor for suspicious child processes spawned by agent hosts
4. Implement path validation on file operations
5. Restrict which functions are exposed as `[KernelFunction]` attributes

The research includes an interactive CTF challenge allowing security practitioners to exploit CVE-2026-26030 in controlled environments.
