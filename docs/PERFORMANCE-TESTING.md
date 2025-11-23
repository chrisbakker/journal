# Data Seeding Tool

This tool generates realistic test data for the Journal application to help with performance testing and development.

## Usage

```bash
# Generate 3000 entries across 365 days
make seed-data

# Or run directly
go run ./cmd/seed
```

## What it generates

- **3000 entries** spread randomly across **365 days** (1 year back from today)
- **Realistic distribution**:
  - 40% meetings (with 1-5 attendees)
  - 50% notes
  - 10% other entries
- **Varied content**: Multiple paragraphs with realistic business content
- **Attendee names**: 16 different names used randomly in meetings
- **Realistic titles**: Curated lists of meeting and note titles

## Configuration

Edit `cmd/seed/main.go` to customize:

```go
totalEntries := 3000  // Number of entries to create
daysSpan := 365       // Days to spread entries across
```

## Statistics

After running, you'll see stats like:

```
📊 Database Statistics:
  Total entries: 3000
  Meetings: 1200 (40.0%)
  Notes: 1500 (50.0%)
  Other: 300 (10.0%)
  Date range: 2024-11-23 to 2025-11-23
```

## Performance Testing

This tool is useful for:
- Testing calendar performance with many days containing entries
- Testing search and autocomplete with large datasets
- Verifying vector embedding generation at scale
- Load testing the frontend with realistic data volumes

## Cleaning Up

To remove all test data and start fresh:

```bash
make db-migrate-down
make db-migrate-up
```

Or to reset the entire database:

```bash
make clean-db
make db-start
make db-migrate-up
```
