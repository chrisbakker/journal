package vectorservice

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	db "github.com/chrisbakker/journal/generated"
	"github.com/chrisbakker/journal/ollama"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pgvector/pgvector-go"
)

type VectorService struct {
	queries        *db.Queries
	ollamaClient   *ollama.Client
	updateInterval time.Duration
	batchSize      int32
	mu             sync.Mutex
	running        bool
	stopCh         chan struct{}
}

func New(queries *db.Queries, ollamaClient *ollama.Client, updateInterval time.Duration, batchSize int32) *VectorService {
	return &VectorService{
		queries:        queries,
		ollamaClient:   ollamaClient,
		updateInterval: updateInterval,
		batchSize:      batchSize,
		stopCh:         make(chan struct{}),
	}
}

func (s *VectorService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	log.Println("Vector service started")

	// Initial update
	go s.updateVectors(ctx)

	// Periodic updates
	ticker := time.NewTicker(s.updateInterval)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.updateVectors(ctx)
			case <-s.stopCh:
				ticker.Stop()
				return
			}
		}
	}()
}

func (s *VectorService) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopCh)
	s.running = false
	log.Println("Vector service stopped")
}

func (s *VectorService) updateVectors(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Get all users - for now just use the default test user
	testUserID := uuid.MustParse("02a0aa58-b88a-46f1-9799-f103e04c0b72")

	// Convert uuid.UUID to pgtype.UUID
	pgUUID := pgtype.UUID{
		Bytes: testUserID,
		Valid: true,
	}

	entries, err := s.queries.GetEntriesNeedingVectors(ctx, db.GetEntriesNeedingVectorsParams{
		UserID: pgUUID,
		Limit:  s.batchSize,
	})
	if err != nil {
		log.Printf("Error fetching entries needing vectors: %v", err)
		return
	}

	if len(entries) == 0 {
		return
	}

	log.Printf("Updating vectors for %d entries", len(entries))

	for _, entry := range entries {
		// Combine title and body for embedding - use plain text from Quill
		text := s.prepareTextForEmbedding(entry.Title, entry.BodyText)

		// Generate embedding
		embedding, err := s.ollamaClient.GenerateEmbedding(ctx, text)
		if err != nil {
			log.Printf("Error generating embedding for entry %s: %v", entry.ID, err)
			continue
		}

		// Convert []float32 to pgvector.Vector pointer
		vec := pgvector.NewVector(embedding)

		// Update entry with vector
		err = s.queries.UpdateEntryVector(ctx, db.UpdateEntryVectorParams{
			ID:              entry.ID,
			EmbeddingVector: &vec,
		})
		if err != nil {
			log.Printf("Error updating vector for entry %s: %v", entry.ID, err)
			continue
		}
	}

	log.Printf("Successfully updated %d vectors", len(entries))
}

func (s *VectorService) prepareTextForEmbedding(title, bodyText string) string {
	// Combine title and body (plain text from Quill, no HTML stripping needed)
	if title != "" {
		return title + "\n\n" + bodyText
	}
	return bodyText
}

func stripHTML(html string) string {
	// Simple HTML tag stripping - removes everything between < and >
	var result strings.Builder
	inTag := false

	for _, char := range html {
		if char == '<' {
			inTag = true
		} else if char == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(char)
		}
	}

	return strings.TrimSpace(result.String())
}

func (s *VectorService) SearchSimilarEntries(ctx context.Context, userID uuid.UUID, query string, limit int32) ([]db.SearchSimilarEntriesRow, error) {
	// Generate embedding for the query
	embedding, err := s.ollamaClient.GenerateEmbedding(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Convert to pgvector.Vector (not pointer for query parameter)
	vec := pgvector.NewVector(embedding)

	// Convert uuid.UUID to pgtype.UUID
	pgUUID := pgtype.UUID{
		Bytes: userID,
		Valid: true,
	}

	// Search for similar entries
	results, err := s.queries.SearchSimilarEntries(ctx, db.SearchSimilarEntriesParams{
		UserID:  pgUUID,
		Column2: vec,
		Limit:   limit,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search similar entries: %w", err)
	}

	return results, nil
}
