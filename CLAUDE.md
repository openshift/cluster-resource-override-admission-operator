# ClusterResourceOverride Operator Claude Code Configuration

This repository includes Claude Code specific configuration for enhanced development assistance.

Please also refer to @AGENTS.md for guidance to all AI agents when working with code in this repository.

## Commands

- **Release Chores:** See [.claude/commands/release-chores.md](./.claude/commands/release-chores.md) for automating version bumps and dependency updates. OpenShift ships new minor releases every quarter (~4 months), and this command automates the bulk of the per-release chores (Go version, Kubernetes deps, image references, OLM bundle regeneration).
