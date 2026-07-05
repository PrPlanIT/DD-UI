package database

import (
	"strings"
	"testing"
	"time"

	"dd-ui/common"
)

func TestBuildLogWhere(t *testing.T) {
	now := time.Unix(1_000_000, 0).UTC()

	// Empty filter → only the default 1-hour time window.
	conds, args := buildLogWhere(common.LogFilter{}, now)
	if len(conds) != 1 || len(args) != 1 {
		t.Fatalf("empty filter: %d conds / %d args, want 1/1 (%v)", len(conds), len(args), conds)
	}
	if !strings.Contains(conds[0], "timestamp >= $1") {
		t.Errorf("default window cond = %q, want timestamp >= $1", conds[0])
	}
	if got := args[0].(time.Time); !got.Equal(now.Add(-time.Hour)) {
		t.Errorf("default window arg = %v, want %v", got, now.Add(-time.Hour))
	}

	// Since set → used instead of the default window.
	since := now.Add(-6 * time.Hour)
	_, args = buildLogWhere(common.LogFilter{Since: since}, now)
	if got := args[0].(time.Time); !got.Equal(since) {
		t.Errorf("Since arg = %v, want %v", got, since)
	}

	// Multi-dimension filter → one cond each, sequential placeholders, ILIKE-wrapped search.
	conds, args = buildLogWhere(common.LogFilter{
		HostNames: []string{"anchorage"},
		Levels:    []string{"ERROR", "WARN"},
		Search:    "boom",
	}, now)
	if len(conds) != 4 || len(args) != 4 { // window + host + level + search
		t.Fatalf("multi filter: %d conds / %d args, want 4/4 (%v)", len(conds), len(args), conds)
	}
	joined := strings.Join(conds, " AND ")
	for _, want := range []string{"hostname = ANY($2)", "level = ANY($3)", "message ILIKE $4"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing cond %q in %q", want, joined)
		}
	}
	if args[3] != "%boom%" {
		t.Errorf("search arg = %q, want %%boom%%", args[3])
	}
}
