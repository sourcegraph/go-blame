package blame

import (
	// "os"
	// "path/filepath"
	"reflect"
	"testing"
)

var testRepoDir = "goblametest"

func TestBlameFile(t *testing.T) {
	hunks, commits, err := BlameFile(testRepoDir, "goblametest.go")
	if err != nil {
		t.Fatalf("Failed to compute blame: %v", err)
	}

	expHunks := []Hunk{
		{CommitID: "26e6e00a6bfd5430a5a8840a543465dc8cac801e", LineStart: 0, LineEnd: 1, CharStart: 0, CharEnd: 20},
		{CommitID: "c497236203ba6400272034a9db7be00859c9863d", LineStart: 1, LineEnd: 2, CharStart: 20, CharEnd: 21},
		{CommitID: "7653ddfbc69a584272a18fe5e675b95025e84bb9", LineStart: 2, LineEnd: 4, CharStart: 21, CharEnd: 37},
		{CommitID: "d858245d0690b83df437ad830ab1e971d389d68d", LineStart: 4, LineEnd: 5, CharStart: 37, CharEnd: 43},
		{CommitID: "7653ddfbc69a584272a18fe5e675b95025e84bb9", LineStart: 5, LineEnd: 6, CharStart: 43, CharEnd: 45},
	}
	expCommits := map[string]Commit{
		"26e6e00a6bfd5430a5a8840a543465dc8cac801e": {
			ID:     "26e6e00a6bfd5430a5a8840a543465dc8cac801e",
			Author: Author{Name: "Beyang Liu", Email: "<beyang.liu@gmail.com>"},
		},
		"c497236203ba6400272034a9db7be00859c9863d": {
			ID:     "c497236203ba6400272034a9db7be00859c9863d",
			Author: Author{Name: "Beyang Liu", Email: "<beyang.liu@gmail.com>"},
		},
		"7653ddfbc69a584272a18fe5e675b95025e84bb9": {
			ID:     "7653ddfbc69a584272a18fe5e675b95025e84bb9",
			Author: Author{Name: "Ricky Bobby", Email: "<ricky@bobby.com>"},
		},
		"d858245d0690b83df437ad830ab1e971d389d68d": {
			ID:     "d858245d0690b83df437ad830ab1e971d389d68d",
			Author: Author{Name: "Sam Hamilton", Email: "<sam@salinas.com>"},
		},
	}

	if !reflect.DeepEqual(expHunks, hunks) {
		t.Errorf("Expected hunks: %+v, but got %+v", expHunks, hunks)
	}

	if !reflect.DeepEqual(expCommits, commits) {
		t.Errorf("Expected commits: %+v, but got %+v", expCommits, commits)
	}
}
