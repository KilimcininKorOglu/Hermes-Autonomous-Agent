package analyzer

// AnalysisResult contains the result of analyzing an AI response
type AnalysisResult struct {
	HasProgress       bool    `json:"hasProgress"`
	IsComplete        bool    `json:"isComplete"`
	IsTestOnly        bool    `json:"isTestOnly"`
	IsStuck           bool    `json:"isStuck"`
	ExitSignal        bool    `json:"exitSignal"`
	Status            string  `json:"status"`
	WorkType          string  `json:"workType"`
	Recommendation    string  `json:"recommendation"`
	Confidence        float64 `json:"confidence"`
	OutputLength      int     `json:"outputLength"`
	ErrorCount        int     `json:"errorCount"`
	CompletionKeyword string  `json:"completionKeyword"`
}

// ExitSignals tracks exit signals across loops
type ExitSignals struct {
	Signals      []SignalEntry `json:"signals"`
	TestOnlyRuns int           `json:"testOnlyRuns"`
	DoneSignals  int           `json:"doneSignals"`
}

// SignalEntry represents a single exit signal
type SignalEntry struct {
	LoopNumber int    `json:"loopNumber"`
	Signal     string `json:"signal"`
	Timestamp  string `json:"timestamp"`
}
