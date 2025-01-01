package monitor

import (
	"context"
	"dogdev/pkg/chat"
	"dogdev/pkg/classify"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/schema"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

const (
	QueryTypeQueryMonitor string = "query_monitor"
	QueryTypeUpdateDocs   string = "update_docs"
	QueryTypeOther        string = "other"
)

const (
	collectionName = "monitor"
)

type Agent interface {
	Query(ctx context.Context, input chat.Input) (string, error)
	Close() error
}

type agent struct {
	retriever  vectorstores.Retriever
	store      *qdrant.Store
	llm        llms.Model
	classifier classify.Classifier
	splitter   textsplitter.TextSplitter
	opts       Options
}

func NewAgent(embedder embeddings.Embedder, llm llms.Model, opts Options) (Agent, error) {
	qdrantURL, err := url.Parse(opts.QdrantURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Qdrant URL: %w", err)
	}

	qdrantOpts := []qdrant.Option{
		qdrant.WithURL(*qdrantURL),
		qdrant.WithEmbedder(embedder),
		qdrant.WithCollectionName(opts.CollectionName),
	}

	if opts.QdrantAPIKey != "" {
		qdrantOpts = append(qdrantOpts, qdrant.WithAPIKey(opts.QdrantAPIKey))
	}

	store, err := qdrant.New(qdrantOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create qdrant store: %w", err)
	}

	retriever := vectorstores.ToRetriever(store, opts.TopK,
		vectorstores.WithScoreThreshold(opts.ScoreThreshold),
	)

	splitter := textsplitter.NewRecursiveCharacter(
		textsplitter.WithChunkSize(opts.ChunkSize),
		textsplitter.WithChunkOverlap(opts.ChunkOverlap),
	)

	return &agent{
		retriever: retriever,
		store:     &store,
		llm:       llm,
		classifier: classify.NewClassifierWithCategories(llm, classify.CategoriesMap{
			QueryTypeQueryMonitor: "query_monitor: Queries about the monitoring data or metrics",
			QueryTypeUpdateDocs:   "update_docs: Requests to add, update, or manage monitoring documents",
			QueryTypeOther:        "other: Other questions or requests that don't fit into the above categories",
		}),
		splitter: splitter,
		opts:     opts,
	}, nil
}

func (a *agent) Close() error {
	// TODO: Add cleanup logic if needed
	return nil
}

func (a *agent) Query(ctx context.Context, input chat.Input) (string, error) {
	queryType, err := a.classifier.Classify(ctx, input.Query)
	if err != nil {
		return "", err
	}

	switch queryType {
	case QueryTypeQueryMonitor:
		return a.QA(ctx, input)
	case QueryTypeUpdateDocs:
		err = a.Upsert(ctx, input.File)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Docs updated successfully: %s", input.File.Name()), nil
	default:
		return fmt.Sprintf("Unknown Query Type: %s", queryType), nil
	}
}

func (a *agent) QA(ctx context.Context, input chat.Input) (string, error) {
	docs, err := a.retriever.GetRelevantDocuments(ctx, input.Query)
	if err != nil {
		return "", err
	}

	if input.File != nil {
		appendDocs, err := createDocs(ctx, input.File, a.splitter)
		if err != nil {
			return "", err
		}
		docs = append(docs, appendDocs...)
	}

	sysPrompt := fmt.Sprintf("You are a monitor agent, you are given the following documents: %v, and the user query is: %s, you need to answer the user query based on the documents.", docs, input.Query)

	return llms.GenerateFromSinglePrompt(ctx, a.llm, sysPrompt)
}

func (a *agent) Upsert(ctx context.Context, file *os.File) error {
	if file == nil {
		return errors.New("file not found")
	}

	docs, err := createDocs(ctx, file, a.splitter)
	if err != nil {
		return err
	}

	_, err = a.store.AddDocuments(ctx, docs)
	return err
}

func createDocs(ctx context.Context, file *os.File, splitter textsplitter.TextSplitter) ([]schema.Document, error) {
	text := createLoader(file)
	return text.LoadAndSplit(ctx, splitter)
}

func createLoader(file *os.File) documentloaders.Loader {
	switch filepath.Ext(file.Name()) {
	case ".csv":
		return documentloaders.NewCSV(file)
	case ".txt":
		return documentloaders.NewText(file)
	default:
		return documentloaders.NewText(file)
	}
}
