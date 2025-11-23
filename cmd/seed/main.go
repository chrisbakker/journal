package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	meetingTitles = []string{
		"Weekly Standup", "Sprint Planning", "Retrospective",
		"1:1 with Manager", "Product Review", "Team Sync",
		"Architecture Discussion", "Client Meeting", "Design Review",
		"Bug Triage", "Quarterly Planning", "Performance Review",
	}

	noteTitles = []string{
		"Project Ideas", "Learning Notes", "Research Findings",
		"Daily Reflection", "Book Summary", "Course Notes",
		"Technical Debt Items", "Feature Brainstorm", "User Feedback",
		"Quick Thoughts", "Meeting Notes", "Goals Review",
	}

	attendeeNames = []string{
		"Alice Johnson", "Bob Smith", "Carol Martinez", "David Chen",
		"Emma Wilson", "Frank Brown", "Grace Lee", "Henry Taylor",
		"Iris Anderson", "Jack Thompson", "Karen White", "Leo Garcia",
		"Maria Rodriguez", "Nathan Clark", "Olivia Harris", "Paul Lewis",
	}

	sampleContent = []string{
		"Discussed the upcoming sprint goals and priorities.",
		"Reviewed the latest design mockups and provided feedback.",
		"Addressed several blockers preventing team progress.",
		"Brainstormed solutions for the performance issues in production.",
		"Walked through the new feature implementation approach.",
		"Analyzed user feedback from the recent release.",
		"Planned the migration strategy for the database upgrade.",
		"Identified technical debt that needs addressing.",
		"Explored new technologies that could improve our workflow.",
		"Documented the decision-making process for future reference.",
	}
)

func main() {
	dbURL := "postgresql://journal:journaldev@localhost:5432/journal?sslmode=disable"
	userID := uuid.MustParse("02a0aa58-b88a-46f1-9799-f103e04c0b72")

	totalEntries := 3000
	daysSpan := 365

	log.Printf("🌱 Starting data generation: %d entries across %d days", totalEntries, daysSpan)

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer pool.Close()

	rand.Seed(time.Now().UnixNano())
	startDate := time.Now().AddDate(0, 0, -daysSpan)

	created := 0
	for i := 0; i < totalEntries; i++ {
		dayOffset := rand.Intn(daysSpan)
		entryDate := startDate.AddDate(0, 0, dayOffset)

		typeRand := rand.Float64()
		var entryType string
		var title string
		var attendees []string

		if typeRand < 0.4 {
			entryType = "meeting"
			title = meetingTitles[rand.Intn(len(meetingTitles))]
			numAttendees := rand.Intn(5) + 1
			attendees = make([]string, numAttendees)
			used := make(map[int]bool)
			for j := 0; j < numAttendees; j++ {
				idx := rand.Intn(len(attendeeNames))
				for used[idx] {
					idx = rand.Intn(len(attendeeNames))
				}
				used[idx] = true
				attendees[j] = attendeeNames[idx]
			}
		} else if typeRand < 0.9 {
			entryType = "notes"
			title = noteTitles[rand.Intn(len(noteTitles))]
			attendees = []string{}
		} else {
			entryType = "other"
			title = fmt.Sprintf("Entry %d", i+1)
			attendees = []string{}
		}

		numParagraphs := rand.Intn(4) + 1
		bodyParts := make([]string, numParagraphs)
		for j := 0; j < numParagraphs; j++ {
			bodyParts[j] = sampleContent[rand.Intn(len(sampleContent))]
		}
		bodyText := ""
		for j, part := range bodyParts {
			if j > 0 {
				bodyText += "\n\n"
			}
			bodyText += part
		}

		deltaOps := []map[string]interface{}{
			{"insert": bodyText + "\n"},
		}
		delta := map[string]interface{}{"ops": deltaOps}
		deltaJSON, _ := json.Marshal(delta)

		bodyHTML := "<p>" + bodyText + "</p>"

		attendeesOriginal := ""
		if len(attendees) > 0 {
			attendeesOriginal = attendees[0]
			for j := 1; j < len(attendees); j++ {
				attendeesOriginal += ", " + attendees[j]
			}
		}

		query := `
			INSERT INTO entries (
				user_id, title, body_delta, body_html, body_text,
				attendees_original, attendees, type,
				day_year, day_month, day_day,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`

		_, err := pool.Exec(ctx, query,
			pgtype.UUID{Bytes: userID, Valid: true},
			title, deltaJSON, bodyHTML, bodyText,
			attendeesOriginal, attendees, entryType,
			entryDate.Year(), int(entryDate.Month()), entryDate.Day(),
			entryDate, entryDate,
		)

		if err != nil {
			log.Printf("Error creating entry %d: %v", i, err)
			continue
		}

		created++
		if (i+1)%100 == 0 {
			log.Printf("Created %d/%d entries...", i+1, totalEntries)
		}
	}

	log.Printf("✅ Successfully created %d entries!", created)

	var stats struct {
		TotalEntries int
		Meetings     int
		Notes        int
		Other        int
		OldestDate   time.Time
		NewestDate   time.Time
	}

	pool.QueryRow(ctx, `
		SELECT 
			COUNT(*),
			COUNT(*) FILTER (WHERE type = 'meeting'),
			COUNT(*) FILTER (WHERE type = 'notes'),
			COUNT(*) FILTER (WHERE type = 'other'),
			MIN(created_at),
			MAX(created_at)
		FROM entries
		WHERE user_id = $1 AND archived = false
	`, pgtype.UUID{Bytes: userID, Valid: true}).Scan(
		&stats.TotalEntries, &stats.Meetings, &stats.Notes, &stats.Other,
		&stats.OldestDate, &stats.NewestDate,
	)

	log.Printf("\n📊 Database Statistics:")
	log.Printf("  Total entries: %d", stats.TotalEntries)
	log.Printf("  Meetings: %d (%.1f%%)", stats.Meetings, float64(stats.Meetings)/float64(stats.TotalEntries)*100)
	log.Printf("  Notes: %d (%.1f%%)", stats.Notes, float64(stats.Notes)/float64(stats.TotalEntries)*100)
	log.Printf("  Other: %d (%.1f%%)", stats.Other, float64(stats.Other)/float64(stats.TotalEntries)*100)
	log.Printf("  Date range: %s to %s", stats.OldestDate.Format("2006-01-02"), stats.NewestDate.Format("2006-01-02"))
}
