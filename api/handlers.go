package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	db "github.com/chrisbakker/journal/generated"
	"github.com/chrisbakker/journal/ollama"
	"github.com/chrisbakker/journal/vectorservice"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/microcosm-cc/bluemonday"
)

type Handler struct {
	queries         *db.Queries
	defaultTimezone string
	sanitizer       *bluemonday.Policy
	vectorService   *vectorservice.VectorService
	ollamaClient    *ollama.Client
}

func NewHandler(queries *db.Queries, defaultTimezone string, vectorService *vectorservice.VectorService, ollamaClient *ollama.Client) *Handler {
	// Create a custom sanitizer policy that allows formatting tags
	sanitizer := bluemonday.UGCPolicy()
	sanitizer.AllowElements("br", "strong", "em", "u", "ul", "ol", "li", "p", "table", "thead", "tbody", "tr", "td", "th", "h1", "h2", "h3")
	sanitizer.AllowAttrs("class").OnElements("ul", "ol", "table")
	sanitizer.AllowAttrs("data-list").OnElements("li")
	sanitizer.AllowAttrs("data-row").OnElements("tr")
	sanitizer.AllowAttrs("colspan", "rowspan").OnElements("td", "th")

	return &Handler{
		queries:         queries,
		defaultTimezone: defaultTimezone,
		sanitizer:       sanitizer,
		vectorService:   vectorService,
		ollamaClient:    ollamaClient,
	}
}

// Request/Response types
type CreateEntryRequest struct {
	Title             string          `json:"title"`
	BodyDelta         json.RawMessage `json:"body_delta"`
	BodyHTML          string          `json:"body_html"`
	BodyText          string          `json:"body_text"`
	AttendeesOriginal string          `json:"attendees_original"`
	Type              string          `json:"type"`
	Date              string          `json:"date"` // YYYY-MM-DD
}

type UpdateEntryRequest struct {
	Title             *string          `json:"title,omitempty"`
	BodyDelta         *json.RawMessage `json:"body_delta,omitempty"`
	BodyHTML          *string          `json:"body_html,omitempty"`
	BodyText          *string          `json:"body_text,omitempty"`
	AttendeesOriginal *string          `json:"attendees_original,omitempty"`
	Type              *string          `json:"type,omitempty"`
}

type EntryResponse struct {
	ID                string          `json:"id"`
	UserID            string          `json:"user_id"`
	Title             string          `json:"title"`
	BodyDelta         json.RawMessage `json:"body_delta"`
	BodyHTML          string          `json:"body_html"`
	BodyText          string          `json:"body_text"`
	AttendeesOriginal string          `json:"attendees_original"`
	Attendees         []string        `json:"attendees"`
	Type              string          `json:"type"`
	DayYear           int32           `json:"day_year"`
	DayMonth          int32           `json:"day_month"`
	DayDay            int32           `json:"day_day"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

func (h *Handler) ListEntriesForDay(c *gin.Context) {
	dateStr := c.Param("date") // Format: YYYY-MM-DD
	dateParts := strings.Split(dateStr, "-")
	if len(dateParts) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
		return
	}
	year, _ := strconv.Atoi(dateParts[0])
	month, _ := strconv.Atoi(dateParts[1])
	day, _ := strconv.Atoi(dateParts[2])

	userID := h.getDefaultUserID(c)

	entries, err := h.queries.ListEntriesForDay(c.Request.Context(), db.ListEntriesForDayParams{
		UserID:   userID,
		DayYear:  int32(year),
		DayMonth: int32(month),
		DayDay:   int32(day),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := make([]EntryResponse, len(entries))
	for i, entry := range entries {
		response[i] = entryToResponse(entry)
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) CreateEntry(c *gin.Context) {
	var req CreateEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	dateParts := strings.Split(req.Date, "-")
	if len(dateParts) != 3 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format, expected YYYY-MM-DD"})
		return
	}
	year, _ := strconv.Atoi(dateParts[0])
	month, _ := strconv.Atoi(dateParts[1])
	day, _ := strconv.Atoi(dateParts[2])

	if req.Type != "meeting" && req.Type != "notes" && req.Type != "other" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be meeting, notes, or other"})
		return
	}

	// Use HTML from Quill directly, sanitize it
	bodyHTML := h.sanitizer.Sanitize(req.BodyHTML)
	attendees := normalizeAttendees(req.AttendeesOriginal)
	userID := h.getDefaultUserID(c)

	entry, err := h.queries.CreateEntry(c.Request.Context(), db.CreateEntryParams{
		UserID:            userID,
		Title:             req.Title,
		BodyDelta:         req.BodyDelta,
		BodyHtml:          bodyHTML,
		BodyText:          req.BodyText,
		AttendeesOriginal: req.AttendeesOriginal,
		Attendees:         attendees,
		Type:              req.Type,
		DayYear:           int32(year),
		DayMonth:          int32(month),
		DayDay:            int32(day),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, entryToResponse(entry))
}

func (h *Handler) UpdateEntry(c *gin.Context) {
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry ID"})
		return
	}

	var req UpdateEntryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existing, err := h.queries.GetEntry(c.Request.Context(), pgtype.UUID{Bytes: entryID, Valid: true})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "entry not found"})
		return
	}

	title := existing.Title
	bodyDelta := existing.BodyDelta
	bodyHTML := existing.BodyHtml
	bodyText := existing.BodyText
	attendeesOriginal := existing.AttendeesOriginal
	attendees := existing.Attendees
	entryType := existing.Type

	if req.Title != nil {
		title = *req.Title
	}
	if req.BodyDelta != nil {
		bodyDelta = *req.BodyDelta
	}
	if req.BodyHTML != nil {
		// Use HTML from Quill directly, sanitize it
		bodyHTML = h.sanitizer.Sanitize(*req.BodyHTML)
	}
	if req.BodyText != nil {
		bodyText = *req.BodyText
	}
	if req.AttendeesOriginal != nil {
		attendeesOriginal = *req.AttendeesOriginal
		attendees = normalizeAttendees(*req.AttendeesOriginal)
	}
	if req.Type != nil {
		if *req.Type != "meeting" && *req.Type != "notes" && *req.Type != "other" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type must be meeting, notes, or other"})
			return
		}
		entryType = *req.Type
	}

	entry, err := h.queries.UpdateEntry(c.Request.Context(), db.UpdateEntryParams{
		ID:                pgtype.UUID{Bytes: entryID, Valid: true},
		Title:             title,
		BodyDelta:         bodyDelta,
		BodyHtml:          bodyHTML,
		BodyText:          bodyText,
		AttendeesOriginal: attendeesOriginal,
		Attendees:         attendees,
		Type:              entryType,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, entryToResponse(entry))
}

func (h *Handler) DeleteEntry(c *gin.Context) {
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry ID"})
		return
	}

	err = h.queries.SoftDeleteEntry(c.Request.Context(), pgtype.UUID{Bytes: entryID, Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func (h *Handler) GetDaysWithEntries(c *gin.Context) {
	yearMonthStr := c.Param("yearmonth") // Format: YYYY-MM
	parts := strings.Split(yearMonthStr, "-")
	if len(parts) != 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid year-month format"})
		return
	}
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	userID := h.getDefaultUserID(c)

	days, err := h.queries.GetDaysWithEntries(c.Request.Context(), db.GetDaysWithEntriesParams{
		UserID:   userID,
		DayYear:  int32(year),
		DayMonth: int32(month),
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	dayNumbers := make([]int32, len(days))
	for i, day := range days {
		dayNumbers[i] = day.DayDay
	}

	c.JSON(http.StatusOK, gin.H{"daysWithEntries": dayNumbers})
}

func (h *Handler) UploadAttachment(c *gin.Context) {
	entryID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid entry ID"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no file provided"})
		return
	}
	defer file.Close()

	fileData := make([]byte, header.Size)
	if _, err := file.Read(fileData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read file"})
		return
	}

	userID := h.getDefaultUserID(c)

	attachment, err := h.queries.CreateAttachment(c.Request.Context(), db.CreateAttachmentParams{
		UserID:    userID,
		EntryID:   pgtype.UUID{Bytes: entryID, Valid: true},
		Filename:  header.Filename,
		MimeType:  header.Header.Get("Content-Type"),
		SizeBytes: header.Size,
		Data:      fileData,
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"id":         attachment.ID,
		"filename":   attachment.Filename,
		"mime_type":  attachment.MimeType,
		"size_bytes": attachment.SizeBytes,
		"created_at": attachment.CreatedAt,
	})
}

func (h *Handler) GetAttachment(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attachment ID"})
		return
	}

	attachment, err := h.queries.GetAttachment(c.Request.Context(), pgtype.UUID{Bytes: attachmentID, Valid: true})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "attachment not found"})
		return
	}

	c.Header("Content-Type", attachment.MimeType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.Filename))
	c.Data(http.StatusOK, attachment.MimeType, attachment.Data)
}

func (h *Handler) DeleteAttachment(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid attachment ID"})
		return
	}

	err = h.queries.DeleteAttachment(c.Request.Context(), pgtype.UUID{Bytes: attachmentID, Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// Helper functions

func (h *Handler) getDefaultUserID(c *gin.Context) pgtype.UUID {
	userIDStr := "02a0aa58-b88a-46f1-9799-f103e04c0b72"
	userID, _ := uuid.Parse(userIDStr)
	return pgtype.UUID{Bytes: userID, Valid: true}
}

func (h *Handler) deltaToHTML(delta json.RawMessage) string {
	var deltaOps struct {
		Ops []struct {
			Insert     string                 `json:"insert"`
			Attributes map[string]interface{} `json:"attributes,omitempty"`
		} `json:"ops"`
	}

	if err := json.Unmarshal(delta, &deltaOps); err != nil {
		return ""
	}

	var html strings.Builder
	var inList bool
	var currentListType string

	for _, op := range deltaOps.Ops {
		text := op.Insert
		isListItem := false
		listType := ""

		// Apply formatting based on attributes BEFORE sanitizing
		if op.Attributes != nil {
			if bold, ok := op.Attributes["bold"].(bool); ok && bold {
				text = "<strong>" + text + "</strong>"
			}
			if italic, ok := op.Attributes["italic"].(bool); ok && italic {
				text = "<em>" + text + "</em>"
			}
			if underline, ok := op.Attributes["underline"].(bool); ok && underline {
				text = "<u>" + text + "</u>"
			}
			if list, ok := op.Attributes["list"].(string); ok {
				isListItem = true
				listType = list
				text = "<li>" + strings.TrimSuffix(text, "\n") + "</li>"
			}
		}

		// Handle list opening/closing
		if isListItem {
			if !inList {
				// Start new list
				if listType == "ordered" {
					html.WriteString("<ol>")
				} else {
					html.WriteString("<ul>")
				}
				inList = true
				currentListType = listType
			} else if currentListType != listType {
				// Close previous list and start new one
				if currentListType == "ordered" {
					html.WriteString("</ol>")
				} else {
					html.WriteString("</ul>")
				}
				if listType == "ordered" {
					html.WriteString("<ol>")
				} else {
					html.WriteString("<ul>")
				}
				currentListType = listType
			}
		} else {
			// Close list if we were in one
			if inList {
				if currentListType == "ordered" {
					html.WriteString("</ol>")
				} else {
					html.WriteString("</ul>")
				}
				inList = false
				currentListType = ""
			}
		}

		// Handle line breaks (except for list items which already handle them)
		if !isListItem {
			text = strings.ReplaceAll(text, "\n", "<br>")
		}

		html.WriteString(text)
	}

	// Close any open list at the end
	if inList {
		if currentListType == "ordered" {
			html.WriteString("</ol>")
		} else {
			html.WriteString("</ul>")
		}
	}

	result := html.String()

	// Sanitize the final HTML
	return h.sanitizer.Sanitize(result)
}

func normalizeAttendees(original string) []string {
	if original == "" {
		return []string{}
	}

	parts := strings.Split(original, ",")
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	return normalized
}

func entryToResponse(entry db.Entry) EntryResponse {
	return EntryResponse{
		ID:                entry.ID.String(),
		UserID:            fmt.Sprintf("%x", entry.UserID.Bytes),
		Title:             entry.Title,
		BodyDelta:         entry.BodyDelta,
		BodyHTML:          entry.BodyHtml,
		BodyText:          entry.BodyText,
		AttendeesOriginal: entry.AttendeesOriginal,
		Attendees:         entry.Attendees,
		Type:              entry.Type,
		DayYear:           entry.DayYear,
		DayMonth:          entry.DayMonth,
		DayDay:            entry.DayDay,
		CreatedAt:         entry.CreatedAt.Time,
		UpdatedAt:         entry.UpdatedAt.Time,
	}
}

// SearchEntries searches for entries by title, body, or attendees
func (h *Handler) SearchEntries(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusOK, []EntryResponse{})
		return
	}

	// For now, use hardcoded test user
	userID, _ := uuid.Parse("02a0aa58-b88a-46f1-9799-f103e04c0b72")

	entries, err := h.queries.SearchEntries(c.Request.Context(), db.SearchEntriesParams{
		UserID:  pgtype.UUID{Bytes: userID, Valid: true},
		Column2: pgtype.Text{String: query, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to search entries"})
		return
	}

	response := make([]EntryResponse, len(entries))
	for i, entry := range entries {
		response[i] = entryToResponse(entry)
	}

	c.JSON(http.StatusOK, response)
}

// ChatRequest represents a chat message from the user
type ChatRequest struct {
	Message string `json:"message"`
}

// ChatResponse represents the AI assistant's response
type ChatResponse struct {
	Response      string          `json:"response"`
	SourceEntries []EntryResponse `json:"source_entries"`
	MessageID     string          `json:"message_id"` // Unique ID for this response
}

// Chat handles AI chat interactions (Phase 3 - RAG)
func (h *Handler) Chat(c *gin.Context) {
	var req ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if req.Message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Message is required"})
		return
	}

	// Use test user for now
	testUserID := uuid.MustParse("02a0aa58-b88a-46f1-9799-f103e04c0b72")

	// Search for similar journal entries using RAG
	similarEntries, err := h.vectorService.SearchSimilarEntries(c.Request.Context(), testUserID, req.Message, 5)
	if err != nil {
		log.Printf("Error searching similar entries: %v", err)
		// Continue without context if search fails
		similarEntries = nil
	}

	log.Printf("Chat search found %d similar entries for query: %s", len(similarEntries), req.Message)

	// Build context from similar entries
	var contextBuilder strings.Builder
	if len(similarEntries) > 0 {
		contextBuilder.WriteString("Here are some relevant journal entries:\n\n")
		for i, entry := range similarEntries {
			// Use plain text from Quill (no HTML stripping needed)
			contextBuilder.WriteString(fmt.Sprintf("%d. %s (Date: %d-%02d-%02d)\n%s\n\n",
				i+1, entry.Title, entry.DayYear, entry.DayMonth, entry.DayDay, entry.BodyText))
		}
	}

	// Build prompt for LLM
	var promptBuilder strings.Builder
	promptBuilder.WriteString("You are a helpful AI assistant with access to the user's journal entries. ")
	promptBuilder.WriteString("Use the provided context to answer questions about past events, meetings, and notes.\n\n")

	if contextBuilder.Len() > 0 {
		promptBuilder.WriteString(contextBuilder.String())
	}

	promptBuilder.WriteString("User Question: ")
	promptBuilder.WriteString(req.Message)
	promptBuilder.WriteString("\n\nIMPORTANT: After your response, on a new line, add 'CITATIONS: ' followed by ONLY the numbers of the journal entries you actually used (e.g., 'CITATIONS: 1, 3' or 'CITATIONS: none' if you didn't use any). Provide a helpful response based on the journal entries above.")

	// Get response from Ollama
	llmResponse, err := h.ollamaClient.Chat(c.Request.Context(), promptBuilder.String())
	if err != nil {
		log.Printf("Error getting LLM response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate response"})
		return
	}

	// Parse citations from LLM response
	var actualResponse string
	var citedIndices []int

	// Split response to extract citations
	parts := strings.Split(llmResponse, "CITATIONS:")
	if len(parts) == 2 {
		actualResponse = strings.TrimSpace(parts[0])
		citationsStr := strings.TrimSpace(parts[1])

		// Parse citation numbers
		if citationsStr != "none" && citationsStr != "" {
			citationParts := strings.Split(citationsStr, ",")
			for _, citStr := range citationParts {
				citStr = strings.TrimSpace(citStr)
				if num, err := strconv.Atoi(citStr); err == nil && num > 0 && num <= len(similarEntries) {
					citedIndices = append(citedIndices, num-1) // Convert to 0-based index
				}
			}
		}
	} else {
		actualResponse = llmResponse
	}

	// Only include entries that were actually cited
	var sourceEntries []EntryResponse
	for _, idx := range citedIndices {
		entry := similarEntries[idx]
		// Need to fetch full entry details
		fullEntry, err := h.queries.GetEntry(c.Request.Context(), entry.ID)
		if err != nil {
			log.Printf("Error fetching entry %s: %v", entry.ID, err)
			continue
		}
		sourceEntries = append(sourceEntries, entryToResponse(fullEntry))
	}

	log.Printf("LLM cited %d out of %d entries", len(citedIndices), len(similarEntries))

	// Generate unique message ID
	messageID := uuid.New().String()

	response := ChatResponse{
		Response:      actualResponse,
		SourceEntries: sourceEntries,
		MessageID:     messageID,
	}

	c.JSON(http.StatusOK, response)
}

func stripHTMLTags(html string) string {
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
