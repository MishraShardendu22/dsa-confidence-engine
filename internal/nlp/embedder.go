package nlp

import (
	"context"
	"hash/fnv"
	"math"
	"strings"
	"unicode"

	"github.com/MishraShardendu22/dsa-confidence-engine/internal/dsa"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
}

// LocalHashingEmbedder produces deterministic unit-normalized semantic embeddings
// using subword character n-grams and token hashing.
type LocalHashingEmbedder struct {
	dims int
}

func NewLocalHashingEmbedder(dims ...int) *LocalHashingEmbedder {
	d := 256
	if len(dims) > 0 && dims[0] > 0 {
		d = dims[0]
	}
	return &LocalHashingEmbedder{dims: d}
}

func (e *LocalHashingEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	vec := make([]float32, e.dims)
	normText := dsa.NormalizeText(text)
	words := strings.Fields(normText)
	if len(words) == 0 {
		return vec, nil
	}

	for _, w := range words {
		// Word token hash
		h := hashToken(w) % uint32(e.dims)
		vec[h] += 2.0 // higher weight for whole tokens

		// Character n-grams (tri-grams and 4-grams) for subword semantic capture
		runes := []rune(w)
		for n := 3; n <= 4; n++ {
			if len(runes) >= n {
				for i := 0; i <= len(runes)-n; i++ {
					sub := string(runes[i : i+n])
					sh := hashToken(sub) % uint32(e.dims)
					vec[sh] += 0.5
				}
			}
		}
	}

	// L2 normalization
	var sumSq float64
	for _, val := range vec {
		sumSq += float64(val * val)
	}

	if sumSq > 0 {
		norm := float32(math.Sqrt(sumSq))
		for i := range vec {
			vec[i] /= norm
		}
	}

	return vec, nil
}

func hashToken(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()
}

func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0.0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i] * b[i])
		normA += float64(a[i] * a[i])
		normB += float64(b[i] * b[i])
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	sim := dot / (math.Sqrt(normA) * math.Sqrt(normB))
	return math.Max(-1.0, math.Min(1.0, sim))
}

func Tokenize(text string) []string {
	var words []string
	var curr strings.Builder

	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			curr.WriteRune(r)
		} else if curr.Len() > 0 {
			words = append(words, curr.String())
			curr.Reset()
		}
	}
	if curr.Len() > 0 {
		words = append(words, curr.String())
	}
	return words
}
