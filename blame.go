package blame

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// type File struct {
// 	Path  string
// 	Hunks []Hunk
// }

type Hunk struct {
	CommitID  string
	LineStart int
	LineEnd   int
	CharStart int
	CharEnd   int
}

type Commit struct {
	ID     string
	Author Author
}

type Author struct {
	Name  string
	Email string
}

func BlameFile(filePath string) ([]*Hunk, map[string]*Commit, error) {
	out, err := exec.Command("git", "blame", "--porcelain", "--", filePath).Output()
	if err != nil {
		return nil, nil, err
	}

	commits := make(map[string]*Commit)
	hunks := make([]*Hunk, 0)
	remainingLines := strings.Split(string(out), "\n")
	for len(remainingLines) > 0 {
		// Consume hunk
		hunkHeader := strings.Split(remainingLines[0], " ")
		if len(hunkHeader) != 4 {
			return nil, nil, fmt.Errorf("Expected at least 4 parts to hunkHeader, but got: '%s'", hunkHeader)
		}
		commitID := hunkHeader[0]
		lineNoCur, _ := strconv.Atoi(hunkHeader[2])
		nLines, _ := strconv.Atoi(hunkHeader[3])
		hunk := Hunk{
			CommitID:  commitID,
			LineStart: int(lineNoCur),
			LineEnd:   int(lineNoCur + nLines),
			// TODO: LineEnd
			// TODO: CharStart/End
		}

		if _, in := commits[commitID]; in {
			// Already seen commit
			remainingLines = remainingLines[2:]
		} else {
			// New commit
			author := strings.Join(strings.Split(remainingLines[1], " ")[1:], " ")
			commits[commitID] = &Commit{
				ID: commitID,
				Author: Author{
					Name: author,
				},
			}

			if strings.HasPrefix(remainingLines[10], "previous") {
				remainingLines = remainingLines[13:]
			} else {
				remainingLines = remainingLines[12:]
			}
		}

		// Consume remaining lines in hunk
		for i := 1; i < nLines; i++ {
			remainingLines = remainingLines[2:]
		}

		hunks = append(hunks, &hunk)
	}

	return hunks, commits, nil
}

// func BlameRepository(repoPath string) ([]*File, []*Commit, error) {
// 	return nil, nil
// }

// type Blamer struct {
// }

// func NewBlamer(blamedFiles []*File) *Blamer {

// }

// func BlameQuery(start int, end int) {
// 	// TODO
// }
