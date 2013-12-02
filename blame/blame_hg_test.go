package blame

import (
	"github.com/kr/pretty"
	"reflect"
	"strings"
	"testing"
)

var testRepoDirHg = "../go-vcs-hgtest"

var expHunksHg = map[string][]Hunk{
	"foo": []Hunk{
		{CommitID: "d047adf8d7ff", LineStart: 0, LineEnd: 1, CharStart: 0, CharEnd: 11},
		{CommitID: "52f96eab35cf", LineStart: 2, LineEnd: 3, CharStart: 12, CharEnd: 27},
		{CommitID: "d14ec9caa006", LineStart: 4, LineEnd: 4, CharStart: 28, CharEnd: 39},
		{CommitID: "52f96eab35cf", LineStart: 5, LineEnd: 6, CharStart: 40, CharEnd: 49},
	},
	"empty-file.txt": nil,
}

var expCommitsHg = map[string]Commit{
	// "c84bb8d093f2": {
	// 	ID:         "c84bb8d093f2",
	// 	Message:    "add empty file",
	// 	Author:     Author{Name: "Quinn Slack", Email: "qslack@qslack.com"},
	// 	AuthorDate: mustParseTime("Mon Dec 02 03:31:13 2013 -0800"),
	// },
	// "bcc18e469216": {
	// 	ID:         "bcc18e469216",
	// 	Message:    "bar",
	// 	Author:     Author{Name: "Quinn Slack", Email: "qslack@qslack.com"},
	// 	AuthorDate: mustParseTime("Sat Jun 01 19:57:17 2013 -0700"),
	// },
	// "0c28a98a22ee": {
	// 	ID:         "0c28a98a22ee",
	// 	Message:    "bar",
	// 	Author:     Author{Name: "Quinn Slack", Email: "qslack@qslack.com"},
	// 	AuthorDate: mustParseTime("Sat Jun 01 19:40:15 2013 -0700"),
	// },
	"d047adf8d7ff": {
		ID:         "d047adf8d7ff",
		Message:    "foo",
		Author:     Author{Name: "Quinn Slack", Email: "qslack@qslack.com"},
		AuthorDate: mustParseTime("Sat Jun 01 19:39:51 2013 -0700"),
	},
	"52f96eab35cf": {
		ID:         "52f96eab35cf",
		Message:    "append",
		Author:     Author{Name: "Quinn Slack", Email: "qslack@qslack.com"},
		AuthorDate: mustParseTime("Mon Dec 02 05:14:51 2013 -0800"),
	},
	"d14ec9caa006": {
		ID:         "d14ec9caa006",
		Message:    "interleave",
		Author:     Author{Name: "Quinn Slack", Email: "qslack@qslack.com"},
		AuthorDate: mustParseTime("Mon Dec 02 05:16:51 2013 -0800"),
	},
}

func TestBlameRepository_Hg(t *testing.T) {
	hunks, commits, err := BlameRepository(testRepoDirHg, "tip", nil)
	if err != nil {
		t.Fatalf("Failed to compute blame: %v", err)
	}

	if !reflect.DeepEqual(expHunksHg, hunks) {
		t.Errorf("Hunks don't match: %+v != %+v\n%v", expHunksHg, hunks, strings.Join(pretty.Diff(expHunksHg, hunks), "\n"))
	}

	if !reflect.DeepEqual(expCommitsHg, commits) {
		t.Errorf("Commits don't match: %+v != %+v", expCommitsHg, commits)
	}
}

func TestBlameFile_Hg(t *testing.T) {
	hunks, commits, err := BlameFile(testRepoDirHg, "foo", "tip")
	if err != nil {
		t.Fatalf("Failed to compute blame: %v", err)
	}

	if !reflect.DeepEqual(expHunksHg["foo"], hunks) {
		t.Errorf("Hunks don't match: %+v != %+v", expHunksHg["foo"], hunks)
	}

	if !reflect.DeepEqual(expCommitsHg, commits) {
		t.Errorf("Commits don't match: %+v != %+v", expCommitsHg, commits)
	}
}

func TestParseHgAnnotateLine(t *testing.T) {
	tests := []struct {
		line   string
		parsed *hgAnnotateLine
	}{}
	for _, test := range tests {
		parsed, err := parseHgAnnotateLine(test.line)
		if err != nil {
			t.Errorf("%q: parseHgAnnotateLine failed: %s", test.line, err)
			continue
		}
		if !reflect.DeepEqual(test.parsed, parsed) {
			t.Errorf("%q: want %+v, got %+v", test.line, test.parsed, parsed)
			continue
		}
	}
}
