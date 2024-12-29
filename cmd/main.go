package main

import (
	"bufio"
	"context"
	"dogdev/pkg/chat"
	"dogdev/pkg/service/monitor"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type CLI struct {
	llm         llms.Model
	embedder    embeddings.Embedder
	agent       monitor.Agent
	history     []string
	pendingFile *os.File
}

func NewCLI() (*CLI, error) {
	llm, err := ollama.New(ollama.WithModel("llama3.2:latest"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LLM: %w", err)
	}

	embedllm, err := ollama.New(ollama.WithModel("nomic-embed-text:latest"))
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embed LLM: %w", err)
	}

	embedder, err := embeddings.NewEmbedder(embedllm)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize embedder: %w", err)
	}

	monitorAgent, err := monitor.NewAgent(embedder, llm, monitor.DefaultOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize agent: %w", err)
	}

	cli := &CLI{
		llm:         llm,
		embedder:    embedder,
		agent:       monitorAgent,
		history:     make([]string, 0),
		pendingFile: nil,
	}

	return cli, nil
}

func (c *CLI) handleAddFile(filepath string) error {
	if filepath == "" {
		return fmt.Errorf("please provide a file path")
	}

	// Close previous pending file if exists
	if c.pendingFile != nil {
		c.pendingFile.Close()
		c.pendingFile = nil
	}

	file, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	c.pendingFile = file
	c.history = append(c.history, fmt.Sprintf("System: File %s ready for next query", filepath))
	fmt.Printf("File %s ready. Please enter your query.\n", filepath)
	return nil
}

func (c *CLI) handleChat(ctx context.Context, input string) error {
	c.history = append(c.history, fmt.Sprintf("User: %s", input))

	chatInput := chat.Input{
		Query: input,
		File:  c.pendingFile,
	}

	response, err := c.agent.Query(ctx, chatInput)
	if err != nil {
		return fmt.Errorf("failed to process query: %w", err)
	}

	// Clear pending file after query
	if c.pendingFile != nil {
		c.pendingFile.Close()
		c.pendingFile = nil
	}

	c.history = append(c.history, fmt.Sprintf("Assistant: %s", response))
	fmt.Println(response)
	return nil
}

func (c *CLI) showHistory() {
	fmt.Println("\nConversation history:")
	for i, msg := range c.history {
		fmt.Printf("%d: %s\n", i+1, msg)
	}
}

func (c *CLI) showHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  /add <filepath> - Add a file for the next query")
	fmt.Println("  /history - Show conversation history")
	fmt.Println("  /help - Show this help message")
	fmt.Println("  /exit - Exit the program")
}

func (c *CLI) Run() error {
	fmt.Println("Welcome to the Interactive LLM Agent!")
	fmt.Println("Type /help to see available commands")

	reader := bufio.NewReader(os.Stdin)
	ctx := context.Background()

	// Ensure pending file is closed on exit
	defer func() {
		if c.pendingFile != nil {
			c.pendingFile.Close()
		}
	}()

	addFileRegex := regexp.MustCompile(`^/add(?:\s+(.+))?$`)

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

		if input == "/exit" {
			fmt.Println("Goodbye!")
			return nil
		}

		if input == "/help" {
			c.showHelp()
			continue
		}

		if input == "/history" {
			c.showHistory()
			continue
		}

		// Replace the strings.HasPrefix block with regex matching
		if matches := addFileRegex.FindStringSubmatch(input); matches != nil {
			filepath := ""
			if len(matches) > 1 {
				filepath = strings.TrimSpace(matches[1])
			}
			if err := c.handleAddFile(filepath); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			continue
		}

		if err := c.handleChat(ctx, input); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

func main() {
	cli, err := NewCLI()
	if err != nil {
		log.Fatalf("Failed to initialize CLI: %v", err)
	}

	if err := cli.Run(); err != nil {
		log.Fatalf("Error running CLI: %v", err)
	}
}
