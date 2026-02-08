package index

import (
	"bufio"
	"fmt"
	"os"
)

// ShowLine represents a single line of file content with its line number.
type ShowLine struct {
	Number  int    `json:"number"`
	Content string `json:"content"`
}

// ShowResult holds the result of reading a file or a range of lines.
type ShowResult struct {
	Path       string     `json:"path"`
	StartLine  int        `json:"startLine"`
	EndLine    int        `json:"endLine"`
	TotalLines int        `json:"totalLines"`
	Lines      []ShowLine `json:"lines"`
}

// ShowFile reads the file at path and returns its contents. If startLine and
// endLine are both 0, all lines are returned. Otherwise, only lines in the
// range [startLine, endLine] (1-indexed, inclusive) are returned. Binary files
// (containing null bytes in the first 512 bytes) are rejected with an error.
func ShowFile(path string, startLine, endLine int) (*ShowResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Check for binary file.
	header := make([]byte, 512)
	n, err := f.Read(header)
	if err != nil && n == 0 {
		// Empty file is fine.
		if info.Size() == 0 {
			return &ShowResult{
				Path:  path,
				Lines: []ShowLine{},
			}, nil
		}
		return nil, err
	}
	for _, b := range header[:n] {
		if b == 0 {
			return nil, fmt.Errorf("%s appears to be a binary file", path)
		}
	}

	// Seek back to start.
	if _, err := f.Seek(0, 0); err != nil {
		return nil, err
	}

	// Read all lines.
	var allLines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		allLines = append(allLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	totalLines := len(allLines)

	// Determine range.
	start := 1
	end := totalLines
	if startLine != 0 || endLine != 0 {
		if startLine != 0 {
			start = startLine
		}
		if endLine != 0 {
			end = endLine
		}
		if start > end {
			return nil, fmt.Errorf("invalid line range: start (%d) is greater than end (%d)", start, end)
		}
		if start > totalLines {
			return nil, fmt.Errorf("start line %d is beyond end of file (%d lines)", start, totalLines)
		}
		if end > totalLines {
			end = totalLines
		}
	}

	var lines []ShowLine
	for i := start; i <= end; i++ {
		lines = append(lines, ShowLine{
			Number:  i,
			Content: allLines[i-1],
		})
	}
	if lines == nil {
		lines = []ShowLine{}
	}

	return &ShowResult{
		Path:       path,
		StartLine:  start,
		EndLine:    end,
		TotalLines: totalLines,
		Lines:      lines,
	}, nil
}
