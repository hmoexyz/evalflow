package main

import "testing"

func tables(t *testing.T, path string) []string {
	s, err := openStore(path)
	if err != nil {
		t.Fatalf("%s: %v", path, err)
	}
	defer s.db.Close()
	rows, err := s.db.Query(`SELECT name FROM sqlite_master WHERE type = 'table' ORDER BY name`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		out = append(out, n)
	}
	return out
}

func TestMigrationBothSchemes(t *testing.T) {
	for _, p := range []string{"/tmp/opencode/mig-legacy.db", "/tmp/opencode/mig-prev.db"} {
		got := tables(t, p)
		t.Logf("%s -> %v", p, got)
		for _, want := range []string{
			"restaurant_rating_evaluation_items",
			"restaurant_rating_workflow_forms",
			"restaurant_rating_form_items",
			"restaurant_rating_submissions",
			"restaurant_rating_submission_scores",
		} {
			found := false
			for _, g := range got {
				if g == want {
					found = true
				}
			}
			if !found {
				t.Errorf("%s missing %s", p, want)
			}
		}
	}
}
