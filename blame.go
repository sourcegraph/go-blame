package blame

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

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

// Note: filePath should be absolute or relative to repoPath
func BlameFile(repoPath string, filePath string) ([]Hunk, map[string]Commit, error) {
	cmd := exec.Command("git", "blame", "--porcelain", "--", filePath)
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}

	commits := make(map[string]Commit)
	hunks := make([]Hunk, 0)
	remainingLines := strings.Split(strings.TrimSpace(string(out)), "\n")
	charOffset := 0
	for len(remainingLines) > 0 {
		// Consume hunk
		hunkHeader := strings.Split(remainingLines[0], " ")
		if len(hunkHeader) != 4 {
			fmt.Printf("Remaining lines: %+v, %d, '%s'\n", remainingLines, len(remainingLines), remainingLines[0])
			return nil, nil, fmt.Errorf("Expected at least 4 parts to hunkHeader, but got: '%s'", hunkHeader)
		}
		commitID := hunkHeader[0]
		lineNoCur, _ := strconv.Atoi(hunkHeader[2])
		nLines, _ := strconv.Atoi(hunkHeader[3])
		hunk := Hunk{
			CommitID:  commitID,
			LineStart: int(lineNoCur) - 1,
			LineEnd:   int(lineNoCur + nLines - 1),
			CharStart: charOffset,
		}

		if _, in := commits[commitID]; in {
			// Already seen commit
			charOffset += len(remainingLines[1])
			remainingLines = remainingLines[2:]
		} else {
			// New commit
			author := strings.Join(strings.Split(remainingLines[1], " ")[1:], " ")
			email := strings.Join(strings.Split(remainingLines[2], " ")[1:], " ")
			commits[commitID] = Commit{
				ID: commitID,
				Author: Author{
					Name:  author,
					Email: email,
				},
			}
			if strings.HasPrefix(remainingLines[10], "previous ") {
				charOffset += len(remainingLines[12])
				remainingLines = remainingLines[13:]
			} else if remainingLines[10] == "boundary" {
				charOffset += len(remainingLines[12])
				remainingLines = remainingLines[13:]
			} else {
				charOffset += len(remainingLines[11])
				remainingLines = remainingLines[12:]
			}
		}

		// Consume remaining lines in hunk
		for i := 1; i < nLines; i++ {
			charOffset += len(remainingLines[1])
			remainingLines = remainingLines[2:]
		}

		hunk.CharEnd = charOffset
		hunks = append(hunks, hunk)
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
