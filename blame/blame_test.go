package blame

import (
	"github.com/kr/pretty"
	"reflect"
	"strings"
	"testing"
)

var testRepoDir = "../goblametest"

func TestBlameFile(t *testing.T) {
	hunks, commits, err := BlameFile(testRepoDir, "goblametest.txt")
	if err != nil {
		t.Fatalf("Failed to compute blame: %v", err)
	}

	expHunks := []Hunk{
		{CommitID: "26e6e00a6bfd5430a5a8840a543465dc8cac801e", LineStart: 0, LineEnd: 1, CharStart: 0, CharEnd: 20},
		{CommitID: "c497236203ba6400272034a9db7be00859c9863d", LineStart: 1, LineEnd: 2, CharStart: 20, CharEnd: 21},
		{CommitID: "7653ddfbc69a584272a18fe5e675b95025e84bb9", LineStart: 2, LineEnd: 4, CharStart: 21, CharEnd: 37},
		{CommitID: "d858245d0690b83df437ad830ab1e971d389d68d", LineStart: 4, LineEnd: 5, CharStart: 37, CharEnd: 43},
		{CommitID: "7653ddfbc69a584272a18fe5e675b95025e84bb9", LineStart: 5, LineEnd: 6, CharStart: 43, CharEnd: 45},
		{CommitID: "496529633d7c1e8359db63aa3d297359479479ff", LineStart: 6, LineEnd: 7, CharStart: 45, CharEnd: 47},
	}
	expCommits := map[string]Commit{
		"26e6e00a6bfd5430a5a8840a543465dc8cac801e": {
			ID:     "26e6e00a6bfd5430a5a8840a543465dc8cac801e",
			Author: Author{Name: "Beyang Liu", Email: "beyang.liu@gmail.com"},
		},
		"c497236203ba6400272034a9db7be00859c9863d": {
			ID:     "c497236203ba6400272034a9db7be00859c9863d",
			Author: Author{Name: "Beyang Liu", Email: "beyang.liu@gmail.com"},
		},
		"7653ddfbc69a584272a18fe5e675b95025e84bb9": {
			ID:     "7653ddfbc69a584272a18fe5e675b95025e84bb9",
			Author: Author{Name: "Ricky Bobby", Email: "ricky@bobby.com"},
		},
		"d858245d0690b83df437ad830ab1e971d389d68d": {
			ID:     "d858245d0690b83df437ad830ab1e971d389d68d",
			Author: Author{Name: "Sam Hamilton", Email: "sam@salinas.com"},
		},
		"496529633d7c1e8359db63aa3d297359479479ff": {
			ID:     "496529633d7c1e8359db63aa3d297359479479ff",
			Author: Author{Name: "Beyang Liu", Email: "beyang.liu@gmail.com"},
		},
	}

	if !reflect.DeepEqual(expHunks, hunks) {
		t.Errorf("Hunks don't match: %s", strings.Join(pretty.Diff(expHunks, hunks), "\n"))
	}

	if !reflect.DeepEqual(expCommits, commits) {
		t.Errorf("Commits don't match: %s", strings.Join(pretty.Diff(expCommits, commits), "\n"))
	}
}

func TestBlameEmptyFile(t *testing.T) {
	hunks, commits, err := BlameFile(testRepoDir, "__init__.py")
	if err != nil {
		t.Errorf("Failed to blame empty file: %v", err)
	}
	expHunks := []Hunk{{CommitID: "ba4f3f4147a2843eb88712b450ea28ec221f3490", LineStart: 0, LineEnd: 0, CharStart: 0, CharEnd: 0}}
	expCommits := map[string]Commit{
		"ba4f3f4147a2843eb88712b450ea28ec221f3490": {
			ID:     "ba4f3f4147a2843eb88712b450ea28ec221f3490",
			Author: Author{Name: "Beyang Liu", Email: "beyang.liu@gmail.com"},
		},
	}
	if !reflect.DeepEqual(expHunks, hunks) {
		t.Errorf("Hunks don't match: %s", strings.Join(pretty.Diff(expHunks, hunks), "\n"))
	}
	if !reflect.DeepEqual(expCommits, commits) {
		t.Errorf("Commits don't match: %s", strings.Join(pretty.Diff(expCommits, commits), "\n"))
	}
}

func TestBlameQuery(t *testing.T) {
	hunks := []Hunk{
		{CommitID: "0", LineStart: 0, LineEnd: 1, CharStart: 0, CharEnd: 2},
		{CommitID: "1", LineStart: 1, LineEnd: 2, CharStart: 2, CharEnd: 4},
		{CommitID: "2", LineStart: 2, LineEnd: 4, CharStart: 4, CharEnd: 8},
	}
	commits := map[string]Commit{
		"0": {ID: "0", Author: Author{Name: "Bob", Email: "bob@bob.com"}},
		"1": {ID: "0", Author: Author{Name: "Joe", Email: "joe@joe.com"}},
		"2": {ID: "0", Author: Author{Name: "Bob", Email: "bob@bob.com"}},
	}

	testcases := []struct {
		CharStart int
		CharEnd   int
		Result    map[Author]int
	}{
		{
			CharStart: 0,
			CharEnd:   2,
			Result: map[Author]int{
				Author{Name: "Bob", Email: "bob@bob.com"}: 2,
			},
		},
		{
			CharStart: 0,
			CharEnd:   4,
			Result: map[Author]int{
				Author{Name: "Bob", Email: "bob@bob.com"}: 2,
				Author{Name: "Joe", Email: "joe@joe.com"}: 2,
			},
		},
		{
			CharStart: 0,
			CharEnd:   6,
			Result: map[Author]int{
				Author{Name: "Bob", Email: "bob@bob.com"}: 4,
				Author{Name: "Joe", Email: "joe@joe.com"}: 2,
			},
		},
		{
			CharStart: 0,
			CharEnd:   0,
			Result:    map[Author]int{},
		},
		{
			CharStart: 7,
			CharEnd:   8,
			Result: map[Author]int{
				Author{Name: "Bob", Email: "bob@bob.com"}: 1,
			},
		},
	}
	for _, testcase := range testcases {
		result, err := BlameQuery(hunks, commits, testcase.CharStart, testcase.CharEnd)
		if err != nil {
			t.Error(err)
		} else if !reflect.DeepEqual(testcase.Result, result) {
			t.Errorf("On query %d:%d, expected %+v, but got %+v", testcase.CharStart, testcase.CharEnd, testcase.Result, result)
		}
	}

	errorQueries := [][2]int{{-1, -1}, {0, 9}}
	for _, query := range errorQueries {
		if _, err := BlameQuery(hunks, commits, query[0], query[1]); err == nil {
			t.Errorf("On query %d:%d, expected error, but got none", query[0], query[1])
		}
	}
}
