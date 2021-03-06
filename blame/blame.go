package blame

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

var Log *log.Logger

func logf(s string, v ...interface{}) {
	if Log != nil {
		Log.Printf(s, v...)
	}
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
	paths := strings.Split(string(lines), "\x00")

	// Directories listed here are git submodules (otherwise only files are
	// listed). Omit these because we can't `git blame` them.
	var files []string
	for _, f := range paths {
		if !isDir(filepath.Join(repoPath, f)) {
			files = append(files, f)
		}
	}

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

type hgRepoAnnotatOutputFormat struct {
	Commits map[string]Commit
	Hunks   map[string][]Hunk
}

func BlameHgRepository(repoPath string, v string, ignorePatterns []string) (map[string][]Hunk, map[string]Commit, error) {
	// write script to temp file
	tmpfile, err := ioutil.TempFile("", "hg-repo-annotate.py")
	if err != nil {
		return nil, nil, err
	}
	defer os.Remove(tmpfile.Name())
	_, err = io.WriteString(tmpfile, hgRepoAnnotatePy)
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.Command("python", tmpfile.Name(), repoPath, v)
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	in := bufio.NewReader(stdout)
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	var data hgRepoAnnotatOutputFormat
	err = json.NewDecoder(in).Decode(&data)
	if err != nil {
		return nil, nil, err
	}
	return data.Hunks, data.Commits, nil
}

func blameFiles(repoPath string, files []string, v string, ignorePatterns []string) (map[string][]Hunk, map[string]Commit, error) {
	hunks := make(map[string][]Hunk)
	commits := make(map[string]Commit)
	var m sync.Mutex
	t0 := time.Now()
	for i, file := range files {
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

		tSleep := time.Millisecond * 50
		time.Sleep(tSleep)
		logf("[% 4d/%d %.1f%% %s/file] BlameFile %s %s", i, len(files), float64(i)/float64(len(files))*100, time.Since(t0.Add(tSleep))/time.Duration(i+1), repoPath, file)

		fileHunks, commits2, err := BlameFile(repoPath, file, v)
		if err != nil {
			return nil, nil, err
		}

		func() {
			m.Lock()
			defer m.Unlock()
			hunks[file] = fileHunks
			for commitID, commit := range commits2 {
				if _, present := commits[commitID]; !present {
					commits[commitID] = commit
				}
			}
		}()
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
		// go 1.8.5 changed the behavior of `git blame` on empty files.
		// previously, it returned a boundary commit. now, it returns nothing.
		// TODO(sqs) TODO(beyang): make `git blame` return the boundary commit
		// on an empty file somehow, or come up with some other workaround.
		st, err := os.Stat(filepath.Join(repoPath, filePath))
		if err == nil && st.Size() == 0 {
			return nil, nil, nil
		}
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
	// write script to temp file
	tmpfile, err := ioutil.TempFile("", "hg-repo-annotate.py")
	if err != nil {
		return nil, nil, err
	}
	defer os.Remove(tmpfile.Name())
	_, err = io.WriteString(tmpfile, hgRepoAnnotatePy)
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.Command("python", tmpfile.Name(), repoPath, v, filePath)
	cmd.Dir = repoPath
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	in := bufio.NewReader(stdout)
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	var data hgRepoAnnotatOutputFormat
	err = json.NewDecoder(in).Decode(&data)
	if err != nil {
		return nil, nil, err
	}
	return data.Hunks[filePath], data.Commits, nil
}
