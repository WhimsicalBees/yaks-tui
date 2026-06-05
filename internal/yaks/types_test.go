package yaks

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDecodeListFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/list.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var roots []Yak
	if err := json.Unmarshal(data, &roots); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(roots) != 1 {
		t.Fatalf("want 1 root, got %d", len(roots))
	}
	root := roots[0]
	if root.Name != "deploy app" {
		t.Errorf("root name = %q", root.Name)
	}
	if root.ID == "" {
		t.Error("root id is empty")
	}
	if len(root.Children) != 2 {
		t.Fatalf("want 2 children, got %d", len(root.Children))
	}

	// Find the "write tests" child regardless of order.
	var wt *Yak
	for i := range root.Children {
		if root.Children[i].Name == "write tests" {
			wt = &root.Children[i]
		}
	}
	if wt == nil {
		t.Fatal("write tests child not found")
	}
	if wt.State != "wip" {
		t.Errorf("write tests state = %q, want wip", wt.State)
	}
	if wt.Context == nil || *wt.Context == "" {
		t.Error("write tests context should be set")
	}
	if wt.Fields["plan"] == "" {
		t.Error("write tests should have a plan field")
	}
	if len(wt.Tags) == 0 {
		t.Error("write tests should have tags")
	}
}
