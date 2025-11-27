package textprocessor_test

import (
	"testing"

	"github.com/yourusername/deisearch/indexer/internal/textprocessor"
)

func TestStem(t *testing.T) {
	stemmer := textprocessor.NewStemmer()

	tests := []struct {
		input    string
		expected string
	}{
		// Verbs
		{"running", "run"},
		{"runs", "run"},
		{"ran", "ran"}, // irregular past tense - not stemmed by Porter

		{"walking", "walk"},
		{"walked", "walk"},
		{"walks", "walk"},

		{"thinking", "think"},
		{"thinks", "think"},

		{"playing", "play"},
		{"played", "play"},
		{"plays", "play"},

		// Plural nouns
		{"cars", "car"},
		{"boxes", "box"},
		{"searches", "search"},
		{"companies", "compani"},
		{"stories", "stori"},

		// Tech terms
		{"databases", "databas"},
		{"algorithms", "algorithm"},
		{"functions", "function"},

		// Adjectives
		{"bigger", "bigger"},
		{"fastest", "fastest"},

		// Should not change
		{"machine", "machin"},
		{"learning", "learn"},
		{"data", "data"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := stemmer.Stem(tt.input)
			if result != tt.expected {
				t.Errorf("Stem(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStemBatch(t *testing.T) {
	stemmer := textprocessor.NewStemmer()

	input := []string{"running", "walked", "dogs", "quickly"}
	expected := []string{"run", "walk", "dog", "quick"}

	result := stemmer.StemBatch(input)

	if len(result) != len(expected) {
		t.Fatalf("Expected %d stems, got %d", len(expected), len(result))
	}

	for i := range result {
		if result[i] != expected[i] {
			t.Errorf("Position %d: Stem(%q) = %q, want %q",
				i, input[i], result[i], expected[i])
		}
	}
}

func BenchmarkStem(b *testing.B) {
	stemmer := textprocessor.NewStemmer()
	word := "running"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stemmer.Stem(word)
	}
}

func BenchmarkStemBatch(b *testing.B) {
	stemmer := textprocessor.NewStemmer()
	words := []string{
		"running", "walked", "quickly", "databases",
		"algorithms", "functions", "searching", "indexing",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		stemmer.StemBatch(words)
	}
}
