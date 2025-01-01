package monitor

type Options struct {
	QdrantURL      string
	QdrantAPIKey   string
	CollectionName string
	TopK           int
	ScoreThreshold float32
	ChunkSize      int
	ChunkOverlap   int
}

func DefaultOptions() Options {
	return Options{
		QdrantURL:      "http://localhost:6334",
		CollectionName: "monitor",
		TopK:           4,
		ScoreThreshold: 0.7,
		ChunkSize:      500,
		ChunkOverlap:   50,
	}
}
