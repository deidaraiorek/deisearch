package embedder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type Model struct {
	mu         sync.Mutex
	serviceURL string
	client     *http.Client
}

func NewModel() (*Model, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	serviceURL := "http://127.0.0.1:5000"

	healthResp, err := client.Get(serviceURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("embedding service not available at %s: %w (start with: python embedding-service/server.py)", serviceURL, err)
	}
	defer healthResp.Body.Close()

	if healthResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding service unhealthy: status %d", healthResp.StatusCode)
	}

	return &Model{
		serviceURL: serviceURL,
		client:     client,
	}, nil
}

func (m *Model) Embed(text string) ([]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	requestBody, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := m.client.Post(m.serviceURL+"/embed", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Embedding []float32 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Embedding, nil
}

func (m *Model) EmbedBatch(texts []string) ([][]float32, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	requestBody, err := json.Marshal(map[string][]string{"texts": texts})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := m.client.Post(m.serviceURL+"/embed", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to call embedding service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding service returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Embeddings [][]float32 `json:"embeddings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Embeddings, nil
}

func (m *Model) Close() error {
	return nil
}
