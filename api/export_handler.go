package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ExportEntries exports all entries as a zip file
func (h *Handler) ExportEntries(c *gin.Context) {
	// Fetch all entries
	entries, err := h.queries.ListAllEntries(context.Background())
	if err != nil {
		log.Printf("Error fetching entries for export: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch entries"})
		return
	}

	// Create a buffer to write the zip file
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Add each entry as a separate file
	for _, entry := range entries {
		// Create filename based on date and title
		date := fmt.Sprintf("%04d-%02d-%02d", entry.DayYear, entry.DayMonth, entry.DayDay)
		safeTitle := sanitizeFilename(entry.Title)
		filename := fmt.Sprintf("%s_%s_%s.json", date, entry.ID.String()[:8], safeTitle)

		// Create the file in the zip
		writer, err := zipWriter.Create(filename)
		if err != nil {
			log.Printf("Error creating zip entry: %v", err)
			continue
		}

		// Prepare entry data for export
		exportData := map[string]interface{}{
			"id":         entry.ID.String(),
			"title":      entry.Title,
			"body_html":  entry.BodyHtml,
			"body_delta": entry.BodyDelta,
			"type":       entry.Type,
			"date": map[string]int32{
				"year":  entry.DayYear,
				"month": entry.DayMonth,
				"day":   entry.DayDay,
			},
			"attendees":  entry.Attendees,
			"created_at": entry.CreatedAt.Time.Format(time.RFC3339),
			"updated_at": entry.UpdatedAt.Time.Format(time.RFC3339),
		}

		// Marshal to JSON with pretty print
		jsonData, err := json.MarshalIndent(exportData, "", "  ")
		if err != nil {
			log.Printf("Error marshaling entry: %v", err)
			continue
		}

		// Write JSON to zip file
		_, err = writer.Write(jsonData)
		if err != nil {
			log.Printf("Error writing to zip: %v", err)
			continue
		}
	}

	// Create a metadata file
	metadata := map[string]interface{}{
		"export_date":   time.Now().Format(time.RFC3339),
		"entry_count":   len(entries),
		"export_format": "json",
		"version":       "1.0",
	}

	metaWriter, err := zipWriter.Create("metadata.json")
	if err == nil {
		metaData, _ := json.MarshalIndent(metadata, "", "  ")
		metaWriter.Write(metaData)
	}

	// Close the zip writer
	err = zipWriter.Close()
	if err != nil {
		log.Printf("Error closing zip writer: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create zip file"})
		return
	}

	// Set headers for file download
	filename := fmt.Sprintf("journal_export_%s.zip", time.Now().Format("2006-01-02"))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Header("Content-Length", fmt.Sprintf("%d", buf.Len()))

	// Send the zip file
	c.Data(http.StatusOK, "application/zip", buf.Bytes())
}

// sanitizeFilename removes or replaces characters that are problematic in filenames
func sanitizeFilename(s string) string {
	// Replace problematic characters with underscores
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	safe := replacer.Replace(s)

	// Limit length
	if len(safe) > 50 {
		safe = safe[:50]
	}

	return safe
}
