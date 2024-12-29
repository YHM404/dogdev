package monitor

type Options struct {
	QdrantURL      string
	CollectionName string
	ChunkSize      int
	ChunkOverlap   int
	ScoreThreshold float32
	TopK           int
}

func DefaultOptions() Options {
	return Options{
		QdrantURL:      "localhost:6333",
		CollectionName: "monitor",
		ChunkSize:      300,
		ChunkOverlap:   30,
		ScoreThreshold: 0.50,
		TopK:           1,
	}
}
