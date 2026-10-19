package domain

// RunRequest selects a project directory and, optionally, an application entry point.
// It deliberately has no executable, shell, environment or free-form arguments.
type RunRequest struct {
	Directory string `json:"directory"`
	Entry     string `json:"entry,omitempty"`
}

// RunProposal is a launch recipe detected and validated by the backend.
type RunProposal struct {
	ID        string     `json:"id"`
	Project   string     `json:"project"`
	Directory string     `json:"directory"`
	Title     string     `json:"title"`
	Command   string     `json:"command"`
	Reason    string     `json:"reason"`
	Request   RunRequest `json:"request"`
}

// Execution is a bounded snapshot of a supervised application, never an interactive shell.
type Execution struct {
	Proposal RunProposal `json:"proposal"`
	Status   string      `json:"status"`
	Logs     string      `json:"logs"`
	URL      string      `json:"url,omitempty"`
	Error    string      `json:"error,omitempty"`
	ExitCode *int        `json:"exitCode,omitempty"`
	Revision uint64      `json:"revision"`
}

// RunPolicyError contains an actionable message safe for the chat and frontend.
type RunPolicyError struct{ Message string }

func (e *RunPolicyError) Error() string { return e.Message }
