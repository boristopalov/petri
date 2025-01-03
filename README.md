# Petri

An orchestrator framework for multi-agent LLM apps, intended to be used for observing AI-to-AI interactions.

## Overview

Petri allows you to:

- Create experiments with multiple AI agents
- Define custom interaction environments
- Control interactions by "stepping through" experiments
- Collect metrics and analyze results
- Study emergent behaviors and cultural evolution

## Project Structure

```
petri/
├── cmd/              # Command-line applications
├── internal/         # Private implementation code
├── pkg/             # Public libraries
├── configs/         # Configuration files
└── docs/            # Documentation
```

## Getting Started

1. Install dependencies:

```bash
go mod download
```

2. Run an example experiment:

```bash
go run cmd/petri/main.go run configs/experiments/chat_room.yaml
```

## Configuration

Experiments are configured using YAML files. See `configs/experiments/` for examples.

## License

MIT License
