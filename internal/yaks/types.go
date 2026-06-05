package yaks

// Yak mirrors one node of `yx list --format json`. Fields not needed by the
// TUI (depth/connector/prefix render hints) are intentionally omitted; encoding/json
// ignores unknown JSON keys by default.
type Yak struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	FullPath string            `json:"full_path"`
	State    string            `json:"state"`
	Context  *string           `json:"context"`
	ParentID *string           `json:"parent_id"`
	Fields   map[string]string `json:"fields"`
	Tags     []string          `json:"tags"`
	Children []Yak             `json:"children"`
}

// Valid states, in triage order.
const (
	StateTodo    = "todo"
	StateWip     = "wip"
	StateBlocked = "blocked"
	StateDone    = "done"
)
