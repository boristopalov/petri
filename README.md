# Petri

A framework for running experiments with interacting AI agents. Named after "petri dish", as it provides an environment for observing AI agent interactions and emergent behaviors.

## Overview

Petri allows researchers to:

- Create experiments with multiple AI agents
- Define custom interaction environments
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
