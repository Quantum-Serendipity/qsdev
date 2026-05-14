# Claude Code Settings - feiskyer/claude-code-settings
- **Source**: https://github.com/feiskyer/claude-code-settings
- **Retrieved**: 2026-03-27

## Overview
Comprehensive collection of Claude Code settings, skills, and sub-agents for AI-assisted development workflows. Includes hooks directory but primary focus is on skills and provider configurations.

## Skills Ecosystem
- **codex-skill**: Non-interactive automation using OpenAI Codex
- **autonomous-skill**: Complex task execution via dual-agent pattern
- **nanobanana-skill**: Image generation via Google Gemini API
- **youtube-transcribe-skill**: YouTube subtitle extraction
- **deep-research**: Multi-agent orchestration for systematic research
- **kiro-skill**: Interactive feature development workflow
- **spec-kit-skill**: Constitution-based development via GitHub Spec-Kit

## Provider Configurations
Pre-configured settings for multiple providers:
- copilot-settings.json: GitHub Copilot proxy (localhost:4141)
- litellm-settings.json: LiteLLM gateway (localhost:4000)
- deepseek-settings.json: DeepSeek v3.1
- qwen-settings.json: Alibaba DashScope with Qwen3-Coder-Plus
- vertex-settings.json: Google Cloud Vertex AI
- azure-settings.json: Azure AI
- openrouter-settings.json: OpenRouter API

## Installation
Available as Claude Code Plugin or via npx skills.

## Note
Repository includes a hooks/ directory but specific hook configurations were not documented in the README. Primary value is the skills ecosystem and multi-provider support.
