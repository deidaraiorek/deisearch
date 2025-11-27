package textprocessor_test

import (
	"reflect"
	"testing"

	"github.com/yourusername/deisearch/indexer/internal/textprocessor"
)

func TestProcess(t *testing.T) {
	processor := textprocessor.NewTextProcessor()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "basic text with stemming",
			input:    "The dogs are running in the park",
			expected: []string{"dog", "run", "park"},
		},
		{
			name:     "tech content",
			input:    "Building scalable systems requires databases and algorithms",
			expected: []string{"build", "scalabl", "system", "requir", "databas", "algorithm"},
		},
		{
			name:     "past tense",
			input:    "He walked quickly and talked loudly",
			expected: []string{"walk", "quick", "talk", "loud"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.Process(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Process(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestProcessToFrequency(t *testing.T) {
	processor := textprocessor.NewTextProcessor()

	input := "running dogs and running cats are running fast"
	result := processor.ProcessToFrequency(input)

	expected := map[string]int{
		"run":  3,
		"dog":  1,
		"cat":  1,
		"fast": 1,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("ProcessToFrequency() = %v, want %v", result, expected)
	}
}

func TestProcessDocument(t *testing.T) {
	processor := textprocessor.NewTextProcessor()

	doc := textprocessor.DocumentFields{
		Title:       "Machine Learning Tutorial",
		Description: "Learn machine learning basics",
		Content:     "Machine learning is a powerful approach to building intelligent systems",
	}

	result := processor.ProcessDocument(doc)

	if result.TermFrequencies["machin"] == 0 {
		t.Error("Expected 'machin' to be in term frequencies")
	}
	if result.TermFrequencies["learn"] == 0 {
		t.Error("Expected 'learn' to be in term frequencies")
	}

	if result.TotalTerms == 0 {
		t.Error("Expected total terms > 0")
	}

	if result.UniqueTerms == 0 {
		t.Error("Expected unique terms > 0")
	}
}

func TestProcessDocumentWithWeights(t *testing.T) {
	processor := textprocessor.NewTextProcessor()

	doc := textprocessor.DocumentFields{
		Title:       "machine learning",
		Description: "machine learning",
		Content:     "machine learning",
	}

	result := processor.ProcessDocumentWithWeights(doc, 3, 2, 1)

	expectedMachineFreq := 6

	if result.TermFrequencies["machin"] != expectedMachineFreq {
		t.Errorf("Expected 'machin' frequency = %d, got %d",
			expectedMachineFreq, result.TermFrequencies["machin"])
	}

	expectedLearningFreq := 6
	if result.TermFrequencies["learn"] != expectedLearningFreq {
		t.Errorf("Expected 'learn' frequency = %d, got %d",
			expectedLearningFreq, result.TermFrequencies["learn"])
	}
}

func TestProcessDocumentEmptyFields(t *testing.T) {
	processor := textprocessor.NewTextProcessor()

	doc := textprocessor.DocumentFields{
		Title:       "Test",
		Description: "",
		Content:     "",
	}

	result := processor.ProcessDocument(doc)

	if result.UniqueTerms != 1 {
		t.Errorf("Expected 1 unique term, got %d", result.UniqueTerms)
	}

	if result.TermFrequencies["test"] != 1 {
		t.Error("Expected 'test' to appear once")
	}
}

func BenchmarkProcess(b *testing.B) {
	processor := textprocessor.NewTextProcessor()
	text := `Machine learning is a subset of artificial intelligence that focuses on
	building systems that learn from data. Deep learning uses neural networks with
	multiple layers to analyze various factors of data.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.Process(text)
	}
}

func BenchmarkProcessToFrequency(b *testing.B) {
	processor := textprocessor.NewTextProcessor()
	text := `Machine learning is a subset of artificial intelligence that focuses on
	building systems that learn from data. Deep learning uses neural networks with
	multiple layers to analyze various factors of data.`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.ProcessToFrequency(text)
	}
}

func BenchmarkProcessDocument(b *testing.B) {
	processor := textprocessor.NewTextProcessor()
	doc := textprocessor.DocumentFields{
		Title:       "Introduction to Machine Learning and AI",
		Description: "Learn the fundamentals of machine learning, deep learning, and artificial intelligence",
		Content: `Machine learning is a subset of artificial intelligence that focuses on
		building systems that learn from data. Deep learning uses neural networks with
		multiple layers to analyze various factors of data. This comprehensive guide
		covers supervised learning, unsupervised learning, reinforcement learning,
		and the latest advances in neural network architectures.`,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor.ProcessDocument(doc)
	}
}
