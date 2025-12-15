# thunk
Thunk is an LLM-powered tool that transforms Git repository histories into human-readable narratives of software evolution. It uses retrieval-augmented generation and long-context summarization to explain not just what changed in a codebase, but why it changed, helping developers understand design decisions and architectural progress over time.

## Development Setup

### Prerequisites
- Go 1.21+
- Docker Desktop (for Milvus vector store)
- OpenAI API key

### Running Milvus Locally

Thunk uses Milvus for vector storage. To set up Milvus locally, use the provided setup script:

```bash
# Start Milvus using embedded etcd
./standalone_embed.sh start

# Verify it's running
docker ps

# Stop when done
./standalone_embed.sh stop
```

The script will start Milvus with embedded etcd and persist data in the `./volumes/milvus` directory.

### Environment Configuration

Create a `.env` file in the project root:

```bash
# OpenAI
OPENAI_API_KEY=your_api_key_here

# GitHub
GITHUB_TOKEN=your_github_token_here

# Milvus
MILVUS_ADDRESS=localhost:19530
MILVUS_COLLECTION=thunk_episodes
MILVUS_DIMENSION=3072
```

### Running Tests

```bash
# Unit tests only (skips integration tests)
go test -short ./...

# Full test suite including Milvus integration tests
./standalone_embed.sh start
go test -v ./internal/rag/store/
```
