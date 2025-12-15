package rag

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

func init() {
	// Load .env for Milvus configuration
	_ = godotenv.Load("../../../.env")
}

// Common errors for Milvus operations
var (
	ErrInvalidDimension = errors.New("invalid vector dimension")
	ErrEmptyRecords     = errors.New("no records provided for insertion")
	ErrConnectionFailed = errors.New("failed to connect to Milvus")
	ErrInsertFailed     = errors.New("failed to insert records")
	ErrSearchFailed     = errors.New("failed to search vectors")
	ErrMissingMetadata  = errors.New("required metadata fields missing")
)

// MilvusConfig holds configuration for Milvus connection and collection
type MilvusConfig struct {
	Address        string // Milvus server address (e.g., "localhost:19530")
	CollectionName string // Name of the collection
	Dimension      int    // Vector dimension (e.g., 3072 for text-embedding-3-large)
	IndexType      string // Index type (default: "HNSW")
	MetricType     string // Similarity metric (default: "COSINE")

	// HNSW index parameters
	M              int // HNSW M parameter (default: 16)
	EfConstruction int // HNSW efConstruction (default: 256)
}

// DefaultMilvusConfig returns default configuration from environment variables
func DefaultMilvusConfig() MilvusConfig {
	address := os.Getenv("MILVUS_ADDRESS")
	if address == "" {
		address = "localhost:19530"
	}

	collection := os.Getenv("MILVUS_COLLECTION")
	if collection == "" {
		collection = "thunk_episodes"
	}

	dimension := 3072 // Default for text-embedding-3-large
	// Could add MILVUS_DIMENSION env var parsing here

	return MilvusConfig{
		Address:        address,
		CollectionName: collection,
		Dimension:      dimension,
		IndexType:      "HNSW",
		MetricType:     "COSINE",
		M:              16,
		EfConstruction: 256,
	}
}

// MilvusStore implements VectorStore interface using Milvus
type MilvusStore struct {
	client client.Client
	config MilvusConfig
}

// NewMilvusStore creates a new Milvus vector store instance
// Connects to Milvus and ensures the collection exists with proper schema
func NewMilvusStore(ctx context.Context, config MilvusConfig) (*MilvusStore, error) {
	// Validate configuration
	if config.Dimension <= 0 {
		return nil, ErrInvalidDimension
	}

	// Connect to Milvus
	c, err := client.NewGrpcClient(ctx, config.Address)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConnectionFailed, err)
	}

	store := &MilvusStore{
		client: c,
		config: config,
	}

	// Create collection if it doesn't exist
	if err := store.ensureCollection(ctx); err != nil {
		c.Close()
		return nil, err
	}

	return store, nil
}

// ensureCollection creates the collection with schema if it doesn't exist
func (m *MilvusStore) ensureCollection(ctx context.Context) error {
	has, err := m.client.HasCollection(ctx, m.config.CollectionName)
	if err != nil {
		return fmt.Errorf("failed to check collection existence: %w", err)
	}

	if has {
		return nil // Collection already exists
	}

	// Define schema for episode embeddings
	schema := &entity.Schema{
		CollectionName: m.config.CollectionName,
		AutoID:         true,
		Fields: []*entity.Field{
			{
				Name:       "id",
				DataType:   entity.FieldTypeInt64,
				PrimaryKey: true,
				AutoID:     true,
			},
			{
				Name:     "episode_id",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "64",
				},
			},
			{
				Name:     "text",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "65535",
				},
			},
			{
				Name:     "embedding",
				DataType: entity.FieldTypeFloatVector,
				TypeParams: map[string]string{
					"dim": fmt.Sprintf("%d", m.config.Dimension),
				},
			},
			{
				Name:     "start_date",
				DataType: entity.FieldTypeInt64, // Unix timestamp
			},
			{
				Name:     "end_date",
				DataType: entity.FieldTypeInt64, // Unix timestamp
			},
			{
				Name:     "authors",
				DataType: entity.FieldTypeVarChar,
				TypeParams: map[string]string{
					"max_length": "1024", // Comma-separated author names
				},
			},
			{
				Name:     "commit_count",
				DataType: entity.FieldTypeInt64,
			},
			{
				Name:     "file_count",
				DataType: entity.FieldTypeInt64,
			},
		},
	}

	// Create collection
	if err := m.client.CreateCollection(ctx, schema, entity.DefaultShardNumber); err != nil {
		return fmt.Errorf("failed to create collection: %w", err)
	}

	// Create HNSW index on embedding field
	idx, err := entity.NewIndexHNSW(entity.COSINE, m.config.M, m.config.EfConstruction)
	if err != nil {
		return fmt.Errorf("failed to create index config: %w", err)
	}

	if err := m.client.CreateIndex(ctx, m.config.CollectionName, "embedding", idx, false); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// Load collection into memory
	if err := m.client.LoadCollection(ctx, m.config.CollectionName, false); err != nil {
		return fmt.Errorf("failed to load collection: %w", err)
	}

	return nil
}

// Insert adds embedding records to Milvus with metadata
func (m *MilvusStore) Insert(ctx context.Context, records []EmbeddingRecord, metadata map[string]interface{}) error {
	if len(records) == 0 {
		return ErrEmptyRecords
	}

	// Validate required metadata
	episodeID, ok := metadata["episode_id"].(string)
	if !ok || episodeID == "" {
		return fmt.Errorf("%w: episode_id", ErrMissingMetadata)
	}

	startDate, _ := metadata["start_date"].(time.Time)
	endDate, _ := metadata["end_date"].(time.Time)
	authors, _ := metadata["authors"].([]string)
	commitCount, _ := metadata["commit_count"].(int)
	fileCount, _ := metadata["file_count"].(int)

	// Prepare column data
	episodeIDs := make([]string, len(records))
	texts := make([]string, len(records))
	embeddings := make([][]float32, len(records))
	startDates := make([]int64, len(records))
	endDates := make([]int64, len(records))
	authorsStr := make([]string, len(records))
	commitCounts := make([]int64, len(records))
	fileCounts := make([]int64, len(records))

	authorsJoined := ""
	if len(authors) > 0 {
		authorsJoined = authors[0]
		for i := 1; i < len(authors); i++ {
			authorsJoined += "," + authors[i]
		}
	}

	for i, record := range records {
		episodeIDs[i] = episodeID
		texts[i] = record.Text
		embeddings[i] = record.Embedding
		startDates[i] = startDate.Unix()
		endDates[i] = endDate.Unix()
		authorsStr[i] = authorsJoined
		commitCounts[i] = int64(commitCount)
		fileCounts[i] = int64(fileCount)
	}

	// Insert data
	columns := []entity.Column{
		entity.NewColumnVarChar("episode_id", episodeIDs),
		entity.NewColumnVarChar("text", texts),
		entity.NewColumnFloatVector("embedding", m.config.Dimension, embeddings),
		entity.NewColumnInt64("start_date", startDates),
		entity.NewColumnInt64("end_date", endDates),
		entity.NewColumnVarChar("authors", authorsStr),
		entity.NewColumnInt64("commit_count", commitCounts),
		entity.NewColumnInt64("file_count", fileCounts),
	}

	if _, err := m.client.Insert(ctx, m.config.CollectionName, "", columns...); err != nil {
		return fmt.Errorf("%w: %v", ErrInsertFailed, err)
	}

	// Flush to ensure data is persisted
	if err := m.client.Flush(ctx, m.config.CollectionName, false); err != nil {
		return fmt.Errorf("failed to flush data: %w", err)
	}

	return nil
}

// Search performs top-K similarity search with optional filtering
func (m *MilvusStore) Search(ctx context.Context, queryVector []float32, topK int, opts *SearchOptions) ([]ContextChunk, error) {
	if len(queryVector) != m.config.Dimension {
		return nil, fmt.Errorf("%w: expected %d, got %d", ErrInvalidDimension, m.config.Dimension, len(queryVector))
	}

	// Build filter expression
	expr := ""
	if opts != nil {
		if len(opts.EpisodeIDs) > 0 {
			expr = fmt.Sprintf(`episode_id in ["%s"]`, opts.EpisodeIDs[0])
			for i := 1; i < len(opts.EpisodeIDs); i++ {
				expr = fmt.Sprintf(`%s or episode_id == "%s"`, expr, opts.EpisodeIDs[i])
			}
		}
	}

	// Configure search parameters
	sp, err := entity.NewIndexHNSWSearchParam(64) // ef parameter for search
	if err != nil {
		return nil, fmt.Errorf("failed to create search params: %w", err)
	}

	// Perform vector search
	vectors := []entity.Vector{entity.FloatVector(queryVector)}
	outputFields := []string{"episode_id", "text", "start_date", "end_date", "authors", "commit_count", "file_count"}

	results, err := m.client.Search(
		ctx,
		m.config.CollectionName,
		nil, // partition names
		expr,
		outputFields,
		vectors,
		"embedding",
		entity.COSINE,
		topK,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSearchFailed, err)
	}

	if len(results) == 0 {
		return []ContextChunk{}, nil
	}

	// Parse results into ContextChunks
	chunks := make([]ContextChunk, 0, results[0].ResultCount)

	for i := 0; i < results[0].ResultCount; i++ {
		chunk := ContextChunk{
			Score:    results[0].Scores[i],
			Metadata: make(map[string]interface{}),
		}

		// Extract fields
		for _, field := range results[0].Fields {
			switch field.Name() {
			case "episode_id":
				chunk.EpisodeID = field.(*entity.ColumnVarChar).Data()[i]
			case "text":
				chunk.Text = field.(*entity.ColumnVarChar).Data()[i]
			case "start_date":
				chunk.StartDate = time.Unix(field.(*entity.ColumnInt64).Data()[i], 0)
			case "end_date":
				chunk.EndDate = time.Unix(field.(*entity.ColumnInt64).Data()[i], 0)
			case "authors":
				authorsStr := field.(*entity.ColumnVarChar).Data()[i]
				// Could parse comma-separated authors here if needed
				chunk.Authors = []string{authorsStr}
			case "commit_count":
				chunk.CommitCount = int(field.(*entity.ColumnInt64).Data()[i])
			case "file_count":
				chunk.FileCount = int(field.(*entity.ColumnInt64).Data()[i])
			}
		}

		chunks = append(chunks, chunk)
	}

	return chunks, nil
}

// Query checks which episode IDs exist in the store
func (m *MilvusStore) Query(ctx context.Context, episodeIDs []string) (map[string]bool, error) {
	if len(episodeIDs) == 0 {
		return map[string]bool{}, nil
	}

	// Build filter expression for the given episode IDs
	expr := fmt.Sprintf(`episode_id == "%s"`, episodeIDs[0])
	for i := 1; i < len(episodeIDs); i++ {
		expr = fmt.Sprintf(`%s or episode_id == "%s"`, expr, episodeIDs[i])
	}

	// Query the collection to get matching episode IDs
	// We use a simple query to get just the episode_id field
	results, err := m.client.Query(
		ctx,
		m.config.CollectionName,
		nil, // partition names
		expr,
		[]string{"episode_id"},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query episodes: %w", err)
	}

	// Build existence map
	existenceMap := make(map[string]bool, len(episodeIDs))
	// Initialize all as non-existent
	for _, id := range episodeIDs {
		existenceMap[id] = false
	}

	// Mark found episodes as existing
	for _, column := range results {
		if column.Name() == "episode_id" {
			if varcharCol, ok := column.(*entity.ColumnVarChar); ok {
				for _, id := range varcharCol.Data() {
					existenceMap[id] = true
				}
			}
		}
	}

	return existenceMap, nil
}

// Delete removes records by episode IDs
func (m *MilvusStore) Delete(ctx context.Context, episodeIDs []string) error {
	if len(episodeIDs) == 0 {
		return nil
	}

	expr := fmt.Sprintf(`episode_id == "%s"`, episodeIDs[0])
	for i := 1; i < len(episodeIDs); i++ {
		expr = fmt.Sprintf(`%s or episode_id == "%s"`, expr, episodeIDs[i])
	}

	if err := m.client.Delete(ctx, m.config.CollectionName, "", expr); err != nil {
		return fmt.Errorf("failed to delete records: %w", err)
	}

	return nil
}

// GetStats returns collection statistics
func (m *MilvusStore) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats, err := m.client.GetCollectionStatistics(ctx, m.config.CollectionName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return map[string]interface{}{
		"row_count": stats["row_count"],
	}, nil
}

// Close releases resources and closes the Milvus connection
func (m *MilvusStore) Close() error {
	if m.client != nil {
		return m.client.Close()
	}
	return nil
}
