# DogDev CLI

An AI-powered assistant that enables natural language queries for Grafana monitoring, making metrics exploration and analysis more intuitive and efficient.

## Features

- Interactive chat with LLM
- File context support for enhanced responses
- Chat history tracking
- Command-based interface

## Local Development Prerequisites

- Go 1.23.3 or higher
- Ollama installed with the following models:
  - llama3.2:latest
  - nomic-embed-text:latest
- Qdrant installed and running on localhost:6333

## Run the CLI

```bash
go run cmd/main.go
```

## Add a file for context and query

```bash
> /add /path/to/file
> "your query"
```

## Show chat history

```bash
> /history
```
