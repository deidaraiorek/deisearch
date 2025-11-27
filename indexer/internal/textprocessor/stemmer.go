package textprocessor

import (
	"github.com/kljensen/snowball"
)

type Stemmer struct{}

func NewStemmer() *Stemmer {
	return &Stemmer{}
}

func (s *Stemmer) Stem(word string) string {
	stemmed, err := snowball.Stem(word, "english", true)
	if err != nil {
		return word
	}
	return stemmed
}

func (s *Stemmer) StemBatch(words []string) []string {
	stemmed := make([]string, len(words))
	for i, word := range words {
		stemmed[i] = s.Stem(word)
	}
	return stemmed
}
