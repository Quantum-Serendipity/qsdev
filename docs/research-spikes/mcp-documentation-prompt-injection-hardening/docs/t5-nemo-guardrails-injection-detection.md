# NeMo Guardrails - Injection Detection Configuration
- **Source**: https://docs.nvidia.com/nemo/microservices/latest/guardrails/tutorials/injection-detection.html
- **Retrieved**: 2026-05-14

## How It Works

NeMo Guardrails injection detection operates through two main components: YARA rules for pattern matching and action handlers for response management. The system scans model outputs against defined rule sets to identify potential exploitation attempts before responses reach users.

## YARA Rules Implementation

The system leverages YARA rules, which are "a set of strings (text or binary patterns) to match and a Boolean expression that specifies the rule logic." These rules are "familiar to many security teams and are easy to audit."

## Supported Injection Types

Four primary injection categories:
- **Code injection** - Python code using shells, networking
- **SQL injection (sqli)** - SQL-based database attacks
- **Template injection** - Jinja template exploitation
- **XSS (Cross-site scripting)** - Web-based injection attacks

## Configuration Options

The `injection_detection` configuration block supports:

| Parameter | Options |
|-----------|---------|
| `injections` | Array specifying which rule types to enable |
| `action` | `reject` (returns refusal message) or `omit` (masks offending content) |
| `yara_rules` | Dictionary enabling custom inline YARA rule definitions |

## Integration Approach

Applications integrate through the guardrail configuration by specifying the injection detection flow in output processing. Operates as "part of a defense-in-depth strategy," particularly for agentic systems.

## Key Note

NeMo Guardrails injection detection focuses on CODE injection (SQL, XSS, template injection) in LLM OUTPUTS - not on detecting prompt injection attacks in LLM INPUTS. This is a complementary but different concern from what we need for documentation content screening.
