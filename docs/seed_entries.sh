#!/bin/bash

API_BASE="http://localhost:8080/api"

# Function to create an entry
create_entry() {
    local year=$1
    local month=$2
    local day=$3
    local title=$4
    local body_html=$5
    local type=$6
    local attendees=$7

    # Format date as YYYY-MM-DD
    local date=$(printf "%04d-%02d-%02d" $year $month $day)
    
    # Create body_text (plain text version)
    local body_text="${body_html}"

    curl -X POST "${API_BASE}/entries" \
        -H "Content-Type: application/json" \
        -d "{
            \"title\": \"${title}\",
            \"body_delta\": {\"ops\":[{\"insert\":\"${body_html}\\\\n\"}]},
            \"body_html\": \"<p>${body_html}</p>\",
            \"body_text\": \"${body_text}\",
            \"type\": \"${type}\",
            \"date\": \"${date}\",
            \"attendees_original\": \"$(echo ${attendees} | jq -r 'join(\", \")' 2>/dev/null || echo '')\"
        }" 2>/dev/null | jq -r '.id // "Created"'
    
    echo "✓ Created: ${title}"
}

echo "🌱 Seeding professional journal entries..."
echo ""

# Get today's date components
TODAY_YEAR=$(date +%Y)
TODAY_MONTH=$(date +%m | sed 's/^0*//')
TODAY_DAY=$(date +%d | sed 's/^0*//')

# Calculate previous days
YESTERDAY_DAY=$((TODAY_DAY - 1))
TWO_DAYS_AGO=$((TODAY_DAY - 2))
THREE_DAYS_AGO=$((TODAY_DAY - 3))

# Today's entries
create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${TODAY_DAY} \
    "Q4 Planning Review" \
    "Reviewed Q4 objectives with the leadership team. Key focus areas: product roadmap finalization, resource allocation for upcoming features, and risk mitigation strategies. Action items: schedule follow-up with engineering leads, prepare detailed timeline for stakeholder presentation." \
    "meeting" \
    '["Sarah Chen", "Michael Rodriguez", "Lisa Park"]'

create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${TODAY_DAY} \
    "Technical Architecture Discussion" \
    "Deep dive into microservices migration strategy. Discussed trade-offs between incremental migration vs. big bang approach. Team consensus on phased approach starting with authentication service. Need to document decision rationale and create RFC for review." \
    "meeting" \
    '["Alex Kumar", "Jordan Smith"]'

create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${TODAY_DAY} \
    "User Research Findings" \
    "Analyzed feedback from 50+ customer interviews. Key insights: users struggle with onboarding flow, feature discoverability is low, but core functionality meets needs. Priority: redesign first-time user experience. Scheduled design sprint for next week." \
    "notes" \
    '[]'

# Yesterday's entries
create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${YESTERDAY_DAY} \
    "Sprint Planning - Sprint 24" \
    "Planned upcoming two-week sprint. Committed to 8 stories totaling 34 points. Focus areas: performance optimization, bug fixes from production monitoring, and initial implementation of new analytics dashboard. Team velocity stable at ~35 points per sprint." \
    "meeting" \
    '["Team"]'

create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${YESTERDAY_DAY} \
    "Production Incident Post-Mortem" \
    "Investigated API latency spike from 2AM-4AM. Root cause: database query timeout due to missing index on user_events table. Immediate fix: added composite index. Long-term: implement query performance monitoring and alerting. No customer impact due to automatic failover." \
    "notes" \
    '[]'

create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${YESTERDAY_DAY} \
    "1:1 with Sarah - Career Development" \
    "Discussed Sarah's interest in moving toward tech lead role. Strengths: strong technical skills, good mentorship abilities. Development areas: improve communication in cross-functional meetings, gain experience with system design. Plan: pair with senior architect on next major feature, present at engineering all-hands." \
    "meeting" \
    '["Sarah Chen"]'

# Two days ago
create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${TWO_DAYS_AGO} \
    "Product Roadmap Alignment" \
    "Synced with product team on Q1 priorities. Engineering capacity: 3 teams, ~15 engineers. Top priorities: 1) Mobile app performance improvements, 2) API v3 launch, 3) Admin dashboard rebuild. Flagged concerns about timeline for dashboard - may need to defer or add resources." \
    "meeting" \
    '["Emma Wilson", "David Lee", "Rachel Martinez"]'

create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${TWO_DAYS_AGO} \
    "Code Review: Payment Service Refactor" \
    "Reviewed PR #847 - significant refactor of payment processing logic. Code quality is excellent, good test coverage at 94%. Suggested improvements: extract validation logic into separate module, add more detailed logging for audit trail, consider circuit breaker pattern for external API calls." \
    "notes" \
    '[]'

create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${TWO_DAYS_AGO} \
    "Security Audit Findings Review" \
    "Met with security team to review Q3 audit results. 3 high-priority findings: outdated dependencies in legacy services, insufficient rate limiting on public APIs, missing encryption for certain sensitive data fields. Created JIRA tickets for each, assigned to team leads. Target resolution: 2 weeks for critical items." \
    "meeting" \
    '["Security Team", "James Wilson"]'

# Three days ago
create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${THREE_DAYS_AGO} \
    "Engineering All-Hands Presentation" \
    "Presented team's progress on infrastructure modernization initiative. Highlights: 60% of services now containerized, deployment time reduced from 45min to 8min, zero-downtime deployments achieved. Next phase: implement comprehensive observability stack with Grafana and Prometheus." \
    "notes" \
    '[]'

create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${THREE_DAYS_AGO} \
    "Vendor Evaluation: Monitoring Tools" \
    "Compared Datadog, New Relic, and Grafana Cloud for our monitoring needs. Datadog: best UX, highest cost. New Relic: good APM features, medium cost. Grafana Cloud: most flexible, lowest cost but requires more setup. Recommendation: Grafana Cloud given our team's expertise and budget constraints." \
    "meeting" \
    '["Alex Kumar", "Infrastructure Team"]'

create_entry ${TODAY_YEAR} ${TODAY_MONTH} ${THREE_DAYS_AGO} \
    "Customer Escalation Resolution" \
    "Worked with support team on enterprise customer issue - data export feature timing out for large datasets. Implemented pagination and background job processing. Solution deployed to production, customer validated fix. Added monitoring to prevent similar issues. Excellent collaboration across teams." \
    "notes" \
    '[]'

echo ""
echo "✅ Seeding complete! Created entries for the last 4 days."
echo "📸 Your journal is now ready for professional screenshots!"
