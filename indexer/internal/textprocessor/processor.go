package textprocessor

import (
	"github.com/yourusername/deisearch/indexer/internal/tokenizer"
)

type TextProcessor struct {
	tokenizer *tokenizer.Tokenizer
	stemmer   *Stemmer
}

func NewTextProcessor() *TextProcessor {
	return &TextProcessor{
		tokenizer: tokenizer.NewTokenizer(),
		stemmer:   NewStemmer(),
	}
}

func (tp *TextProcessor) Process(text string) []string {
	tokens := tp.tokenizer.Tokenize(text)

	stemmed := make([]string, len(tokens))
	for i, token := range tokens {
		stemmed[i] = tp.stemmer.Stem(token)
	}
	return stemmed
}

func (tp *TextProcessor) ProcessToFrequency(text string) map[string]int {
	tokens := tp.Process(text)

	freq := make(map[string]int)
	for _, token := range tokens {
		freq[token]++
	}

	return freq
}

type DocumentFields struct {
	Title       string
	Description string
	Content     string
}

type ProcessedDocument struct {
	TermFrequencies map[string]int
	TotalTerms      int
	UniqueTerms     int
}

func (tp *TextProcessor) ProcessDocument(doc DocumentFields) ProcessedDocument {
	allText := doc.Title + " " + doc.Description + " " + doc.Content

	termFreq := tp.ProcessToFrequency(allText)
	totalTerms := 0
	for _, freq := range termFreq {
		totalTerms += freq
	}

	return ProcessedDocument{
		TermFrequencies: termFreq,
		TotalTerms:      totalTerms,
		UniqueTerms:     len(termFreq),
	}
}

func (tp *TextProcessor) ProcessDocumentWithWeights(doc DocumentFields, titleWeight, descWeight, contentWeight int) ProcessedDocument {
	termFreq := make(map[string]int)

	if doc.Title != "" && titleWeight > 0 {
		titleTerms := tp.ProcessToFrequency(doc.Title)
		for term, freq := range titleTerms {
			termFreq[term] += freq * titleWeight
		}
	}

	if doc.Description != "" && descWeight > 0 {
		descTerms := tp.ProcessToFrequency(doc.Description)
		for term, freq := range descTerms {
			termFreq[term] += freq * descWeight
		}
	}

	if doc.Content != "" && contentWeight > 0 {
		contentTerms := tp.ProcessToFrequency(doc.Content)
		for term, freq := range contentTerms {
			termFreq[term] += freq * contentWeight
		}
	}

	totalTerms := 0
	for _, freq := range termFreq {
		totalTerms += freq
	}

	return ProcessedDocument{
		TermFrequencies: termFreq,
		TotalTerms:      totalTerms,
		UniqueTerms:     len(termFreq),
	}
}
