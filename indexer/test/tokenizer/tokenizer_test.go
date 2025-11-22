package tokenizer_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/yourusername/deisearch/indexer/internal/tokenizer"
)

func TestTokenize(t *testing.T) {
	tok := tokenizer.NewTokenizer()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "basic text",
			input:    "The quick brown fox jumps over the lazy dog",
			expected: []string{"quick", "brown", "fox", "jumps", "lazy", "dog"},
		},
		{
			name:     "with punctuation",
			input:    "Hello, world! How are you?",
			expected: []string{"hello", "world"},
		},
		{
			name:     "with numbers",
			input:    "Python 3.11 is great for AI/ML tasks",
			expected: []string{"python", "great", "ai", "ml", "tasks"},
		},
		{
			name:     "hyphenated words",
			input:    "machine-learning and deep-learning are cool",
			expected: []string{"machine", "learning", "deep", "learning", "cool"},
		},
		{
			name:     "mixed alphanumeric",
			input:    "COVID-19 pandemic in 2020 was tough",
			expected: []string{"covid", "pandemic", "tough"},
		},
		{
			name:     "HTML entities",
			input:    "This&nbsp;is&amp;test&lt;html&gt;",
			expected: []string{"isandtest", "html"},
		},
		{
			name:     "single character removal",
			input:    "I have a big dog",
			expected: []string{"big", "dog"},
		},
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "only stop words",
			input:    "the and or but",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tok.Tokenize(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Tokenize(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestTokenizeToFrequency(t *testing.T) {
	tok := tokenizer.NewTokenizer()

	input := "the quick brown fox jumps over the quick fox"
	result := tok.TokenizeToFrequency(input)

	expected := map[string]int{
		"quick": 2,
		"brown": 1,
		"fox":   2,
		"jumps": 1,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("TokenizeToFrequency() = %v, want %v", result, expected)
	}
}

func TestTokenizeFrequencyCount(t *testing.T) {
	tok := tokenizer.NewTokenizer()

	input := "machine learning machine learning deep learning"
	result := tok.TokenizeToFrequency(input)

	expected := map[string]int{
		"machine":  2,
		"learning": 3,
		"deep":     1,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("TokenizeToFrequency() = %v, want %v", result, expected)
	}
}

func TestIsValidToken(t *testing.T) {
	tok := tokenizer.NewTokenizer()

	tests := []struct {
		token    string
		expected bool
	}{
		{"hello", true},
		{"world", true},
		{"covid19", true},
		{"123", false},
		{"999", false},
		{"abc123def", true},
		{"123abc", true},
		{"a1b2c3d4", true},
	}

	for _, tt := range tests {
		t.Run(tt.token, func(t *testing.T) {
			result := tok.IsValidToken(tt.token)
			if result != tt.expected {
				t.Errorf("isValidToken(%q) = %v, want %v", tt.token, result, tt.expected)
			}
		})
	}
}

func TestStopWords(t *testing.T) {
	tok := tokenizer.NewTokenizer()

	commonStopWords := []string{"the", "a", "an", "and", "or", "but", "is", "was", "are"}

	for _, word := range commonStopWords {
		if !tok.StopWords[word] {
			t.Errorf("Expected %q to be a stop word", word)
		}
	}

	notStopWords := []string{"machine", "learning", "search", "engine"}
	for _, word := range notStopWords {
		if tok.StopWords[word] {
			t.Errorf("Expected %q to NOT be a stop word", word)
		}
	}
}

func TestLengthFiltering(t *testing.T) {
	tok := tokenizer.NewTokenizer()

	result := tok.Tokenize("a b c hello world")
	expected := []string{"hello", "world"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Tokenize() = %v, want %v", result, expected)
	}

	longToken := strings.Repeat("a", 60)
	result = tok.Tokenize(longToken + " hello")
	expected = []string{"hello"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Long token filtering failed: got %v, want %v", result, expected)
	}
}

func BenchmarkTokenize(b *testing.B) {
	tok := tokenizer.NewTokenizer()
	text := `Machine learning is a subset of artificial intelligence that focuses on
	building systems that learn from data. Deep learning, a subset of machine learning,
	uses neural networks with multiple layers to analyze various factors of data.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.Tokenize(text)
	}
}

func BenchmarkTokenizeToFrequency(b *testing.B) {
	tok := tokenizer.NewTokenizer()
	text := `Machine learning is a subset of artificial intelligence that focuses on
	building systems that learn from data. Deep learning, a subset of machine learning,
	uses neural networks with multiple layers to analyze various factors of data.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tok.TokenizeToFrequency(text)
	}
}
