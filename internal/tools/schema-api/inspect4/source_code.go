package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

type codeBlock struct {
	lines      []string
	origin     string
	resource   string
	fixedLines []string
}

func (c *codeBlock) addLine(line string) {
	c.lines = append(c.lines, line)
	c.origin += "\n" + line
}

type SourceCodeFile struct {
	file    string
	content string
	lines   []string
	blocks  []*codeBlock
}

func loadSourceCode(path string, content string) *SourceCodeFile {
	if content == "" {
		bs, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		content = string(bs)
	}
	return &SourceCodeFile{
		file:    path,
		content: content,
		lines:   strings.Split(content, "\n"),
	}
}

func (s *SourceCodeFile) analysisBlocks() {
	block := &codeBlock{}
	for _, line := range s.lines {
		if strings.HasPrefix(line, "resource \"azurerm_") {
			if len(block.lines) > 0 {
				s.blocks = append(s.blocks, block)
			}
			resourceType := strings.Trim(strings.Split(line, " ")[1], "\"")
			block = &codeBlock{
				resource: resourceType,
			}
			block.addLine(line)
		} else if strings.HasPrefix(line, "}") && block.resource != "" {
			block.addLine(line)
			s.blocks = append(s.blocks, block)
			block = &codeBlock{}
		} else {
			// append to previous block if the empty line is just below that block
			if line == "" && len(block.lines) == 0 && len(s.blocks) > 0 {
				s.blocks[len(s.blocks)-1].addLine(line)
			} else {
				block.addLine(line)
			}
		}
	}

	if len(block.lines) > 0 {
		s.blocks = append(s.blocks, block)
	}
}

func (s *SourceCodeFile) lastGoBlockOf(idx int) *codeBlock {
	if idx < len(s.blocks) {
		for j := idx - 1; j >= 0; j-- {
			if s.blocks[j].resource == "" {
				return s.blocks[j]
			}
		}
	}
	return nil
}

func (s *SourceCodeFile) contextOf(idx int) []*codeBlock {
	if idx < 0 || idx > len(s.blocks) {
		return nil
	}

	var res []*codeBlock
	for j := idx - 1; j >= 0; j-- {
		if s.blocks[j].resource == "" {
			break
		}
		res = append(res, s.blocks[j])
		if len(res) > 2 {
			break
		}
	}
	slices.Reverse(res)
	res = append(res, s.blocks[idx])
	for j := idx + 1; j < len(s.blocks); j++ {
		if s.blocks[j].resource == "" {
			break
		}
		res = append(res, s.blocks[j])
		if len(res) > 5 {
			break
		}
	}
	return res
}

func contextHasResource(context []*codeBlock, targetResourceType string) bool {
	for _, block := range context {
		if block.resource == targetResourceType {
			return true
		}
	}
	return false
}

func (s *SourceCodeFile) fixedCode() string {
	newLines := make([]string, 0, len(s.lines)+100)
	for _, block := range s.blocks {
		if len(block.fixedLines) > 0 {
			newLines = append(newLines, block.fixedLines...)
		} else {
			newLines = append(newLines, block.lines...)
		}
	}
	result := strings.Join(newLines, "\n")
	if strings.HasSuffix(s.content, "\n") && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result
}

var internalDir string

func init() {
	_, file, _, _ := runtime.Caller(0)
	fd := filepath.Dir
	internalDir = fd(fd(fd(fd(file))))
}

func findAllTestFiles(dir string) []string {
	if dir == "" {
		dir = filepath.Join(internalDir, "services")
	}
	var files []string
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(d.Name(), "_test.go") {
			files = append(files, path)
		}
		return nil
	})
	return files
}
