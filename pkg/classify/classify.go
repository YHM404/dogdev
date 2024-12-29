package classify

import (
	"context"
	"fmt"
	"strings"

	"github.com/tmc/langchaingo/llms"
)

type Category struct {
	Name        string
	Description string
}

type Classifier interface {
	Classify(ctx context.Context, text string) (string, error)
}

type classifier struct {
	llm        llms.Model
	options    []llms.CallOption
	categories []Category
}

type CategoryProvider interface {
	GetCategories() []Category
}

type CategoriesMap map[string]string

func (c CategoriesMap) GetCategories() []Category {
	var categories []Category
	for name, desc := range c {
		categories = append(categories, Category{Name: name, Description: desc})
	}
	return categories
}

func NewClassifierWithCategories(llm llms.Model, categories CategoryProvider, options ...llms.CallOption) Classifier {
	return &classifier{
		llm:        llm,
		options:    options,
		categories: categories.GetCategories(),
	}
}

func (c *classifier) Classify(ctx context.Context, text string) (string, error) {
	var categoryDescriptions []string
	for _, category := range c.categories {
		categoryDescriptions = append(categoryDescriptions, fmt.Sprintf("%s: %s", category.Name, category.Description))
	}

	classificationPrompt := fmt.Sprintf(`Please classify the following text into one of these categories. Only return the category name.

Categories:
%s

Text: %s

Category:`, strings.Join(categoryDescriptions, "\n"), text)

	completion, err := llms.GenerateFromSinglePrompt(ctx, c.llm, classificationPrompt, c.options...)
	if err != nil {
		return "", fmt.Errorf("classification failed: %w", err)
	}

	return completion, nil
}
