// Package sitereport builds a client-ready site report from the data a
// project already has: Search Console, Google Analytics, tracked rankings, the
// latest site audit and the stored backlink snapshot. It buys nothing: Google
// reads are free and every other section reads stored rows, so one click is
// 0 paid calls and 0 credits.
package sitereport

import (
	"errors"
	"time"
)

// Section names a part of the report the user can include.
type Section string

// The sections a report can carry, in the order they render.
const (
	SectionSearch    Section = "search"
	SectionTraffic   Section = "traffic"
	SectionRankings  Section = "rankings"
	SectionAudit     Section = "audit"
	SectionBacklinks Section = "backlinks"
)

// AllSections is every section in render order.
var AllSections = []Section{SectionSearch, SectionTraffic, SectionRankings, SectionAudit, SectionBacklinks}

// PeriodDays are the reporting windows the builder offers.
var PeriodDays = []int{7, 28, 90}

const (
	maxTitleChars = 120
	maxNoteChars  = 4_000
	maxTableRows  = 25
	// gscDelayDays is how far behind today Search Console's final data is.
	gscDelayDays = 3
)

// ErrProjectNotFound is returned for a project outside the caller's organization.
var ErrProjectNotFound = errors.New("site report project not found")

// ErrNotConnected marks a data source the project has not connected.
var ErrNotConnected = errors.New("data source not connected")

// Error is a validation error safe to show to the user.
type Error struct{ Code, Message string }

func (e *Error) Error() string { return e.Message }

func invalid(message string) error { return &Error{Code: "validation_error", Message: message} }

// Input is one "Create report" click.
type Input struct {
	Title      string    `json:"title"`
	PeriodDays int       `json:"periodDays"`
	Sections   []Section `json:"sections"`
	Note       string    `json:"note"`
	NextSteps  string    `json:"nextSteps"`
	Share      bool      `json:"share"`
}

// Result says what was saved and which sections had no data to show.
type Result struct {
	ReportID   string        `json:"reportId"`
	Title      string        `json:"title"`
	ShareToken *string       `json:"shareToken"`
	Included   []Section     `json:"included"`
	Skipped    []SkippedPart `json:"skipped"`
}

// SkippedPart explains why a requested section is not in the report.
type SkippedPart struct {
	Section Section `json:"section"`
	Reason  string  `json:"reason"`
}

// Project is the site the report is about.
type Project struct {
	Name   string
	Domain string
}

// Period is the reporting window and the one before it, both inclusive.
type Period struct {
	Days                           int
	Start, End, PrevStart, PrevEnd time.Time
}

// Snapshot is everything one report renders. A nil section was skipped.
type Snapshot struct {
	Project     Project
	Period      Period
	GeneratedAt time.Time
	Accent      string
	Note        string
	NextSteps   string
	Search      *Search
	Traffic     *Traffic
	Rankings    *Rankings
	Audit       *Audit
	Backlinks   *Backlinks
}

// Search is Search Console performance for the period.
type Search struct {
	Current, Previous Totals
	HasPrevious       bool
	Daily             []SearchDay
	TopQueries        []Row
	TopPages          []Row
	Gains, Losses     []Change
	Opportunities     []Opportunity
	Countries         []Row
}

// Totals are summed clicks and impressions with weighted CTR and position.
type Totals struct {
	Clicks      float64 `json:"clicks"`
	Impressions float64 `json:"impressions"`
	CTR         float64 `json:"ctr"`
	Position    float64 `json:"position"`
}

// SearchDay is one day of clicks and impressions.
type SearchDay struct {
	Date        string  `json:"date"`
	Clicks      float64 `json:"clicks"`
	Impressions float64 `json:"impressions"`
}

// Row is one query, page or country.
type Row struct {
	Key         string  `json:"key"`
	Clicks      float64 `json:"clicks"`
	Impressions float64 `json:"impressions"`
	CTR         float64 `json:"ctr"`
	Position    float64 `json:"position"`
}

// Change is a query or page whose clicks moved against the previous period.
type Change struct {
	Key          string
	Clicks       float64
	PrevClicks   float64
	Position     float64
	PrevPosition float64
}

// Opportunity is a query ranking 5 to 20 (Search Console striking distance), where a small
// move up brings the most new clicks.
type Opportunity struct {
	Query       string  `json:"query"`
	Page        string  `json:"page"`
	Clicks      float64 `json:"clicks"`
	Impressions float64 `json:"impressions"`
	Position    float64 `json:"position"`
}

// Traffic is Google Analytics organic-search traffic for the period.
type Traffic struct {
	Currency        string
	Sessions        Metric
	Users           Metric
	EngagedSessions Metric
	EngagementRate  Metric
	KeyEvents       Metric
	Revenue         Metric
	Daily           []TrafficDay
}

// Metric is a value and its previous-period value, either possibly unknown.
type Metric struct {
	Value, Previous *float64
}

// TrafficDay is one day of organic sessions.
type TrafficDay struct {
	Date     string
	Sessions float64
}

// Rankings are the tracked keywords' latest positions.
type Rankings struct {
	Domain     string
	Location   string
	Device     string
	CheckedAt  *time.Time
	Total      int
	Top3       int
	Top10      int
	Top20      int
	NotRanking int
	Improved   int
	Declined   int
	Rows       []RankRow
}

// RankRow is one tracked keyword.
type RankRow struct {
	Keyword  string
	Position *int
	Previous *int
	URL      string
	Volume   *int
}

// Change is positive when the keyword moved up.
func (r RankRow) Change() int {
	if r.Position != nil && r.Previous != nil {
		return *r.Previous - *r.Position
	}
	return 0
}

// Audit is the latest completed site audit.
type Audit struct {
	CompletedAt  time.Time
	PagesCrawled int
	Critical     int
	Warning      int
	Info         int
	Issues       []Issue
	Lighthouse   *Lighthouse
}

// Issue is one audit issue type with the pages it was found on.
type Issue struct {
	Type     string
	Severity string
	Pages    int
	Examples []string
}

// Lighthouse averages the audit's mobile Lighthouse runs.
type Lighthouse struct {
	Pages         int
	Performance   *int
	Accessibility *int
	BestPractices *int
	SEO           *int
	LCPMillis     *float64
	CLS           *float64
	INPMillis     *float64
}

// Backlinks is the latest stored backlink snapshot for the project domain.
type Backlinks struct {
	CapturedAt           time.Time
	Rank                 *int64
	Backlinks            *int64
	ReferringDomains     *int64
	NewBacklinks         *int64
	LostBacklinks        *int64
	NewReferringDomains  *int64
	LostReferringDomains *int64
}
