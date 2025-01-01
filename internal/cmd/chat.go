package cmd

import (
	"bufio"
	"context"
	"dogdev/pkg/chat"
	"dogdev/pkg/service/monitor"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
	"github.com/tmc/langchaingo/llms/openai"
)

var (
	chatCmd = &cobra.Command{
		Use:   "chat",
		Short: "Start an interactive chat session",
		Long:  `Start an interactive chat session with the LLM agent.`,
		RunE:  runChat,
	}

	// Configuration
	config *Config
)

func init() {
	rootCmd.AddCommand(chatCmd)
}

type chatSession struct {
	llm         llms.Model
	embedder    embeddings.Embedder
	agent       monitor.Agent
	history     []string
	pendingFile *os.File
}

func createLLM() (llms.Model, error) {
	switch config.LLM.Provider {
	case "ollama":
		return ollama.New(ollama.WithModel(config.LLM.Model))
	case "openai":
		opts := []openai.Option{openai.WithModel(config.LLM.Model)}
		if config.LLM.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(config.LLM.BaseURL))
		}
		if config.LLM.APIKey != "" {
			opts = append(opts, openai.WithToken(config.LLM.APIKey))
		}
		return openai.New(opts...)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", config.LLM.Provider)
	}
}

func createEmbedder() (embeddings.Embedder, error) {
	switch config.Embedding.Provider {
	case "ollama":
		llm, err := ollama.New(ollama.WithModel(config.Embedding.Model))
		if err != nil {
			return nil, fmt.Errorf("failed to initialize embedding LLM: %w", err)
		}
		return embeddings.NewEmbedder(llm)
	case "openai":
		opts := []openai.Option{openai.WithModel(config.Embedding.Model)}
		if config.LLM.BaseURL != "" {
			opts = append(opts, openai.WithBaseURL(config.LLM.BaseURL))
		}
		if config.LLM.APIKey != "" {
			opts = append(opts, openai.WithToken(config.LLM.APIKey))
		}
		llm, err := openai.New(opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize embedding LLM: %w", err)
		}
		return embeddings.NewEmbedder(llm)
	default:
		return nil, fmt.Errorf("unsupported embedding provider: %s", config.Embedding.Provider)
	}
}

func newChatSession() (*chatSession, error) {
	llm, err := createLLM()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM: %w", err)
	}

	embedder, err := createEmbedder()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedder: %w", err)
	}

	opts := monitor.Options{
		QdrantURL:      config.Qdrant.URL,
		QdrantAPIKey:   config.Qdrant.APIKey,
		CollectionName: config.Qdrant.Collection,
		TopK:           config.Qdrant.TopK,
		ScoreThreshold: config.Qdrant.ScoreThreshold,
		ChunkSize:      config.Qdrant.ChunkSize,
		ChunkOverlap:   config.Qdrant.ChunkOverlap,
	}

	monitorAgent, err := monitor.NewAgent(embedder, llm, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize agent: %w", err)
	}

	return &chatSession{
		llm:         llm,
		embedder:    embedder,
		agent:       monitorAgent,
		history:     make([]string, 0),
		pendingFile: nil,
	}, nil
}

func (s *chatSession) handleAddFile(filepath string) error {
	if filepath == "" {
		return fmt.Errorf("please provide a file path")
	}

	if s.pendingFile != nil {
		s.pendingFile.Close()
		s.pendingFile = nil
	}

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	s.pendingFile = file
	s.history = append(s.history, fmt.Sprintf("System: File %s ready for next query", filepath))
	fmt.Printf("File %s ready. Please enter your query.\n", filepath)
	return nil
}

func (s *chatSession) handleChat(ctx context.Context, input string) error {
	s.history = append(s.history, fmt.Sprintf("User: %s", input))

	chatInput := chat.Input{
		Query: input,
		File:  s.pendingFile,
	}

	response, err := s.agent.Query(ctx, chatInput)
	if err != nil {
		return fmt.Errorf("failed to process query: %w", err)
	}

	if s.pendingFile != nil {
		s.pendingFile.Close()
		s.pendingFile = nil
	}

	s.history = append(s.history, fmt.Sprintf("Assistant: %s", response))
	fmt.Println(response)
	return nil
}

func (s *chatSession) showHistory() {
	fmt.Println("\nConversation history:")
	for i, msg := range s.history {
		fmt.Printf("%d: %s\n", i+1, msg)
	}
}

func (s *chatSession) showHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  /add <filepath> - Add a file for the next query")
	fmt.Println("  /history - Show conversation history")
	fmt.Println("  /help - Show this help message")
	fmt.Println("  /exit - Exit the program")
}

func runChat(cmd *cobra.Command, args []string) error {
	var err error
	config, err = loadConfig("")
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	session, err := newChatSession()
	if err != nil {
		return fmt.Errorf("failed to initialize chat session: %w", err)
	}

	defer func() {
		if session.pendingFile != nil {
			session.pendingFile.Close()
		}
	}()

	fmt.Println("Welcome to the Interactive LLM Agent!")
	fmt.Printf("Using LLM: %s/%s\n", config.LLM.Provider, config.LLM.Model)
	fmt.Printf("Using Embeddings: %s/%s\n", config.Embedding.Provider, config.Embedding.Model)
	fmt.Println("Type /help to see available commands")

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	for {
		fmt.Print("\n> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read input: %w", err)
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		switch {
		case input == "/exit":
			fmt.Println("Goodbye!")
			return nil
		case input == "/help":
			session.showHelp()
		case input == "/history":
			session.showHistory()
		case strings.HasPrefix(input, "/add "):
			filepath := strings.TrimPrefix(input, "/add ")
			if err := session.handleAddFile(filepath); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		default:
			if err := session.handleChat(ctx, input); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
		}
	}
}
