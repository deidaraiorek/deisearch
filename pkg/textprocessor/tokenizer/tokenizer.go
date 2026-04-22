package tokenizer

import (
	"regexp"
	"strings"
	"unicode"
)

type Tokenizer struct {
	StopWords map[string]bool
	minLength int
	maxLength int
}

func NewTokenizer() *Tokenizer {
	return &Tokenizer{
		StopWords: defaultStopWords(),
		minLength: 2,
		maxLength: 50,
	}
}

func (t *Tokenizer) Tokenize(text string) []string {
	normalized := t.normalize(text)
	words := t.split(normalized)

	tokens := make([]string, 0)

	for _, word := range words {
		if word == "" {
			continue
		}

		if t.StopWords[word] {
			continue
		}

		if len(word) < t.minLength || len(word) > t.maxLength {
			continue
		}

		if !t.IsValidToken(word) {
			continue
		}

		tokens = append(tokens, word)
	}
	return tokens
}

func (t *Tokenizer) TokenizeToFrequency(text string) map[string]int {
	tokens := t.Tokenize(text)
	result := make(map[string]int)

	for _, token := range tokens {
		result[token]++
	}
	return result
}

func (t *Tokenizer) normalize(text string) string {
	text = strings.ToLower(text)

	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "and")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")

	text = strings.ReplaceAll(text, "-", " ")
	text = strings.ReplaceAll(text, "_", " ")

	return text
}

func (t *Tokenizer) split(text string) []string {
	re := regexp.MustCompile(`[a-z0-9]+`)
	return re.FindAllString(text, -1)
}

func (t *Tokenizer) IsValidToken(word string) bool {
	alphaCount := 0
	digitCount := 0

	for _, r := range word {
		if unicode.IsLetter(r) {
			alphaCount++
		} else if unicode.IsDigit(r) {
			digitCount++
		}
	}
	if alphaCount == 0 {
		return false
	}
	if digitCount > alphaCount {
		return false
	}
	return true
}

func defaultStopWords() map[string]bool {
	words := []string{
		// Articles
		"a", "an", "the",

		// Pronouns
		"i", "me", "my", "myself", "we", "our", "ours", "ourselves",
		"you", "your", "yours", "yourself", "yourselves",
		"he", "him", "his", "himself", "she", "her", "hers", "herself",
		"it", "its", "itself", "they", "them", "their", "theirs", "themselves",

		// Prepositions
		"of", "at", "by", "for", "with", "about", "against", "between",
		"into", "through", "during", "before", "after", "above", "below",
		"to", "from", "up", "down", "in", "out", "on", "off", "over", "under",

		// Conjunctions
		"and", "or", "but", "if", "while", "because", "as", "until",
		"than", "so", "nor", "yet",

		// Common verbs
		"is", "am", "are", "was", "were", "be", "been", "being",
		"have", "has", "had", "having",
		"do", "does", "did", "doing",
		"will", "would", "should", "could", "can", "may", "might", "must",

		// Other common words
		"this", "that", "these", "those",
		"what", "which", "who", "whom", "whose", "when", "where", "why", "how",
		"all", "each", "every", "both", "few", "more", "most", "other", "some", "such",
		"no", "not", "only", "own", "same", "then", "there", "too", "very",
	}

	stopWords := make(map[string]bool, len(words))
	for _, word := range words {
		stopWords[word] = true
	}
	return stopWords
}
