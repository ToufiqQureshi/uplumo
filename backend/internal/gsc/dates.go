package gsc

// DateRange is the inclusive report interval. Runtime date validation and
// comparisons live in performance.go and are shared by HTTP and MCP reports.
type DateRange struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}
