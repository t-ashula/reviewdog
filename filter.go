package reviewdog

import (
	"path/filepath"
	"strings"

	"github.com/haya14busa/reviewdog/diff"
)

// FilteredCheck represents CheckResult with filtering info.
type FilteredCheck struct {
	*CheckResult
	InDiff   bool
	LnumDiff int
}

// FilterCheck filters check results by diff. It doesn't not drop check which
// is not in diff but set FilteredCheck.InDiff field false.
func FilterCheck(results []*CheckResult, diff []*diff.FileDiff, strip int, wd string) []*FilteredCheck {
	checks := make([]*FilteredCheck, 0, len(results))

	addedlines := addedDiffLines(diff, strip)

	for _, result := range results {
		check := &FilteredCheck{CheckResult: result}

		addedline := addedlines.Get(result.Path, result.Lnum)
		if filepath.IsAbs(result.Path) && wd != "" {
			relPath, err := filepath.Rel(wd, result.Path)
			if err == nil {
				result.Path = relPath
			}
		}
		result.Path = cleanPath(result.Path)
		if addedline != nil {
			check.InDiff = true
			check.LnumDiff = addedline.LnumDiff
		}

		checks = append(checks, check)
	}

	return checks
}

func cleanPath(path string) string {
	p := filepath.Clean(path)
	if p == "." {
		return ""
	}
	return p
}

// addedLine represents added line in diff.
type addedLine struct {
	Path     string // path to new file
	Lnum     int    // the line number in the new file
	LnumDiff int    // the line number of the diff (Same as Lnumdiff of diff.Line)
	Content  string // line content
}

// posToAddedLine is a hash table of normalized path to line number to addedLine.
type posToAddedLine map[string]map[int]*addedLine

func (p posToAddedLine) Get(path string, lnum int) *addedLine {
	npath, err := normalizePath(path)
	if err != nil {
		return nil
	}
	ltodiff, ok := p[npath]
	if !ok {
		return nil
	}
	diffline, ok := ltodiff[lnum]
	if !ok {
		return nil
	}
	return diffline
}

// addedDiffLines traverse []*diff.FileDiff and returns posToAddedLine.
func addedDiffLines(filediffs []*diff.FileDiff, strip int) posToAddedLine {
	r := make(posToAddedLine)
	for _, filediff := range filediffs {
		path := filediff.PathNew
		ltodiff := make(map[int]*addedLine)
		if strip > 0 {
			ps := strings.Split(filepath.ToSlash(filediff.PathNew), "/")
			if len(ps) > strip {
				path = filepath.Join(ps[strip:]...)
			}
		}
		np, err := normalizePath(path)
		if err != nil {
			// FIXME(haya14busa): log or return error?
			continue
		}
		path = np

		for _, hunk := range filediff.Hunks {
			for _, line := range hunk.Lines {
				if line.Type == diff.LineAdded {
					ltodiff[line.LnumNew] = &addedLine{
						Path:     path,
						Lnum:     line.LnumNew,
						LnumDiff: line.LnumDiff,
						Content:  line.Content,
					}
				}
			}
		}
		r[path] = ltodiff
	}
	return r
}

func normalizePath(p string) (string, error) {
	if !filepath.IsAbs(p) {
		path, err := filepath.Abs(p)
		if err != nil {
			return "", err
		}
		p = path
	}
	return filepath.ToSlash(p), nil
}
