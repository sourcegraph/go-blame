package blame

import (
	"code.google.com/p/rog-go/parallel"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
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

	Message string

	// AuthorDate is the date when this commit was originally made. (It may
	// differ from the commit date, which is changed during rebases, etc.)
	AuthorDate time.Time
}

type Author struct {
	Name  string
	Email string
}

func BlameRepository(repoPath, v string, ignorePatterns []string) (map[string][]Hunk, map[string]Commit, error) {
	if isDir(filepath.Join(repoPath, ".hg")) {
		return BlameHgRepository(repoPath, v, ignorePatterns)
	}
	return BlameGitRepository(repoPath, v, ignorePatterns)
}

func BlameFile(repoPath, filePath, v string) ([]Hunk, map[string]Commit, error) {
	if isDir(filepath.Join(repoPath, ".hg")) {
		return BlameHgFile(repoPath, filePath, v)
	}
	return BlameGitFile(repoPath, filePath, v)
}

// isDir returns true if path is an existing directory, and false otherwise.
func isDir(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func listGitRepositoryFiles(repoPath string, v string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-z", "-r", v, "--name-only")
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	lines, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	files := strings.Split(string(lines), "\x00")
	return files, nil
}

func BlameGitRepository(repoPath string, v string, ignorePatterns []string) (map[string][]Hunk, map[string]Commit, error) {
	files, err := listGitRepositoryFiles(repoPath, v)
	if err != nil {
		return nil, nil, err
	}
	return blameFiles(repoPath, files, v, ignorePatterns)
}

func listHgRepositoryFiles(repoPath string, v string) ([]string, error) {
	cmd := exec.Command("hg", "locate", "--print0", "-r", v)
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	lines, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	files := strings.Split(string(lines), "\x00")
	return files, nil
}

func BlameHgRepository(repoPath string, v string, ignorePatterns []string) (map[string][]Hunk, map[string]Commit, error) {
	files, err := listHgRepositoryFiles(repoPath, v)
	if err != nil {
		return nil, nil, err
	}
	return blameFiles(repoPath, files, v, ignorePatterns)
}

func blameFiles(repoPath string, files []string, v string, ignorePatterns []string) (map[string][]Hunk, map[string]Commit, error) {
	hunks := make(map[string][]Hunk)
	commits := make(map[string]Commit)
	var m sync.Mutex
	par := parallel.NewRun(8)
	for _, file := range files {
		file := string(file)
		if file == "" {
			continue
		}

		ignore := false
		for _, pat := range ignorePatterns {
			if strings.Contains(file, pat) {
				ignore = true
				break
			}
		}
		if ignore {
			continue
		}

		par.Do(func() error {
			fileHunks, commits2, err := BlameFile(repoPath, file, v)
			if err != nil {
				return err
			}

			m.Lock()
			defer m.Unlock()
			hunks[file] = fileHunks
			for commitID, commit := range commits2 {
				if _, present := commits[commitID]; !present {
					commits[commitID] = commit
				}
			}
			return nil
		})
	}
	err := par.Wait()
	if err != nil {
		return nil, nil, err
	}

	return hunks, commits, nil
}

// Note: filePath should be absolute or relative to repoPath
func BlameGitFile(repoPath string, filePath string, v string) ([]Hunk, map[string]Commit, error) {
	cmd := exec.Command("git", "blame", "-w", "--porcelain", v, "--", filePath)
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}
	if len(out) < 1 {
		return nil, nil, fmt.Errorf("Expected git output of length at least 1")
	}

	commits := make(map[string]Commit)
	hunks := make([]Hunk, 0)
	remainingLines := strings.Split(string(out[:len(out)-1]), "\n")
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
			if len(email) >= 2 && email[0] == '<' && email[len(email)-1] == '>' {
				email = email[1 : len(email)-1]
			}
			authorTime, err := strconv.ParseInt(strings.Join(strings.Split(remainingLines[3], " ")[1:], " "), 10, 64)
			if err != nil {
				return nil, nil, fmt.Errorf("Failed to parse author-time %q", remainingLines[3])
			}
			summary := strings.Join(strings.Split(remainingLines[9], " ")[1:], " ")
			commit := Commit{
				ID:      commitID,
				Message: summary,
				Author: Author{
					Name:  author,
					Email: email,
				},
				AuthorDate: time.Unix(authorTime, 0),
			}

			if len(remainingLines) >= 13 && strings.HasPrefix(remainingLines[10], "previous ") {
				charOffset += len(remainingLines[12])
				remainingLines = remainingLines[13:]
			} else if len(remainingLines) >= 13 && remainingLines[10] == "boundary" {
				charOffset += len(remainingLines[12])
				remainingLines = remainingLines[13:]
			} else if len(remainingLines) >= 12 {
				charOffset += len(remainingLines[11])
				remainingLines = remainingLines[12:]
			} else if len(remainingLines) == 11 {
				// Empty file
				remainingLines = remainingLines[11:]
			} else {
				return nil, nil, fmt.Errorf("Unexpected number of remaining lines (%d):\n%s", len(remainingLines), "  "+strings.Join(remainingLines, "\n  "))
			}

			commits[commitID] = commit
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

// Note: filePath should be absolute or relative to repoPath
func BlameHgFile(repoPath string, filePath string, v string) ([]Hunk, map[string]Commit, error) {
	cmd := exec.Command("hg", "annotate", "-r", v, "-nduvc", "--", filePath)
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	var currentHunk *Hunk
	var hunks []Hunk
	commits := make(map[string]Commit)
	for i, line := range lines {
		parsed, err := parseHgAnnotateLine(line)
		if err != nil {
			return nil, nil, err
		}
		if parsed == nil {
			continue
		}

		if _, present := commits[parsed.changeset]; !present {
			msg, err := getHgCommitMessage(repoPath, parsed.changeset)
			if err != nil {
				return nil, nil, err
			}
			commits[parsed.changeset] = Commit{
				ID:         parsed.changeset,
				AuthorDate: parsed.date,
				Author:     Author{Name: parsed.authorName, Email: parsed.authorEmail},
				Message:    msg,
			}
		}

		if currentHunk == nil {
			currentHunk = &Hunk{
				CommitID:  parsed.changeset,
				LineStart: 0, CharStart: 0,
			}
		}

		currentHunk.LineEnd = i
		currentHunk.CharEnd += parsed.bytelen

		if currentHunk.CommitID != parsed.changeset || i == len(lines)-1 {
			hunks = append(hunks, *currentHunk)
			currentHunk = &Hunk{
				CommitID:  parsed.changeset,
				LineStart: i + 1, CharStart: currentHunk.CharEnd + 1,
			}
		}
	}

	return hunks, commits, nil
}

func getHgCommitMessage(repoPath string, changeset string) (msg string, err error) {
	cmd := exec.Command("hg", "log", "-r", changeset, "--template", "{desc}")
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

type hgAnnotateLine struct {
	authorName, authorEmail string
	changeset               string
	date                    time.Time
	bytelen                 int
}

var hgAnnotateRE = regexp.MustCompile(`^\s*(.*)\s+(<[^ >]+[ >]?)\s*\d+\s*([0-9a-f]+)\s*([^:]*:[^:]*:[^:]*):(.*)$`)
var hgDateFormat = "Mon Jan 2 15:04:05 2006 -0700"

func parseHgAnnotateLine(line string) (*hgAnnotateLine, error) {
	if line == "" {
		return nil, nil
	}

	parts := hgAnnotateRE.FindStringSubmatch(line)
	if len(parts) < 5 {
		if strings.Contains(line, ": binary file") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to match (got only %d matches) %q", len(parts), line)
	}

	a := new(hgAnnotateLine)

	var datestr string
	a.authorName, a.authorEmail, a.changeset, datestr = strings.TrimSpace(parts[1]), parts[2], parts[3], parts[4]

	a.authorEmail = strings.TrimSpace(strings.Replace(strings.Replace(a.authorEmail, ">", "", -1), "<", "", -1))

	date, err := time.Parse(hgDateFormat, datestr)
	if err != nil {
		return nil, err
	}
	a.date = date

	contents := parts[5]
	if len(contents) > 0 {
		a.bytelen = len(contents) - 1
	} else {
		a.bytelen = len(contents)
	}

	return a, nil
}
