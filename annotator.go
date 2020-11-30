package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

type manifestAnnotator struct {
	FileName       string
	Annotation     string
	SkipAnnotation string
	Value          string

	Kind         string
	GroupVersion string
	Name         string
	Namespace    string
}

type manifest struct {
	name      string
	namespace string

	start int
	end   int
}

func (a *manifestAnnotator) Run() error {
	lines, err := readLines(a.FileName)
	if err != nil {
		return err
	}
	output := &bytes.Buffer{}
	currentManifest := []string{}
	for _, line := range lines {
		if strings.HasPrefix(line, "---") {
			a.processManifest(currentManifest, output)
			currentManifest = []string{}
		} else {
			currentManifest = append(currentManifest, line)
			continue
		}
		output.WriteString(line + "\n")
	}
	changed := a.processManifest(currentManifest, output)
	if changed {
		if err = ioutil.WriteFile(a.FileName, output.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (a *manifestAnnotator) processManifest(lines []string, out *bytes.Buffer) bool {

	changed := false
	// first determine kind and apiVersion
	var kind, groupVersion string
	for _, line := range lines {
		if strings.HasPrefix(line, "kind:") {
			kind = strings.TrimSpace(strings.TrimPrefix(line, "kind:"))
			continue
		}
		if strings.HasPrefix(line, "apiVersion:") {
			groupVersion = strings.TrimSpace(strings.TrimPrefix(line, "apiVersion:"))
		}
	}

	inMetadata := false
	metadataLines := []string{}
	metadataProcessed := false
	for _, line := range lines {
		if inMetadata {
			if !strings.HasPrefix(line, "  ") {
				changed = a.processMetadata(metadataLines, kind, groupVersion, out)
				inMetadata = false
				metadataProcessed = true
			} else {
				metadataLines = append(metadataLines, line)
				continue
			}
		}
		if strings.HasPrefix(line, "metadata:") {
			inMetadata = true
		}
		out.WriteString(line + "\n")
	}
	if !metadataProcessed && len(metadataLines) > 0 {
		changed = a.processMetadata(metadataLines, kind, groupVersion, out)
	}
	return changed
}

func (a *manifestAnnotator) processMetadata(lines []string, kind, groupVersion string, out *bytes.Buffer) bool {
	// Determine information about the current manifest
	changed := false
	var name, namespace string
	for _, line := range lines {
		if strings.HasPrefix(line, "  name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "  name:"))
		}
		if strings.HasPrefix(line, "  namespace:") {
			namespace = strings.TrimSpace(strings.TrimPrefix(line, "  namespace:"))
		}
	}
	skipProcessing := (len(a.Kind) > 0 && a.Kind != kind) ||
		(len(a.GroupVersion) > 0 && a.GroupVersion != groupVersion) ||
		(len(a.Name) > 0 && a.Name != name) ||
		(len(a.Namespace) > 0 && a.Namespace != namespace)

	if skipProcessing {
		for _, line := range lines {
			out.WriteString(line + "\n")
		}
		return changed
	}

	annotationLines := []string{}
	inAnnotations := false
	annotationsProcessed := false
	for _, line := range lines {
		if inAnnotations {
			if !strings.HasPrefix(line, "    ") {
				changed = a.processAnnotations(annotationLines, out)
				inAnnotations = false
				annotationsProcessed = true
			} else {
				annotationLines = append(annotationLines, line)
				continue
			}
		}
		if strings.HasPrefix(line, "  annotations:") {
			inAnnotations = true
		}
		out.WriteString(line + "\n")
	}
	if len(annotationLines) == 0 { // annotations were never found
		changed = true
		out.WriteString("  annotations:\n")
	}
	if !annotationsProcessed {
		changed = a.processAnnotations(annotationLines, out)
	}
	return changed
}

type annotationSorter struct {
	lines []string
}

func (s *annotationSorter) Len() int {
	return len(s.lines)
}

// Less reports whether the element with
// index i should sort before the element with index j.
func (s *annotationSorter) Less(i, j int) bool {
	partsI := strings.SplitN(s.lines[i], ":", 2)
	partsJ := strings.SplitN(s.lines[j], ":", 2)
	if partsI[0] == partsJ[0] {
		return partsI[1] < partsJ[1]
	}
	return partsI[0] < partsJ[0]
}

// Swap swaps the elements with indexes i and j.
func (s *annotationSorter) Swap(i, j int) {
	tmp := s.lines[i]
	s.lines[i] = s.lines[j]
	s.lines[j] = tmp
}

func sortAnnotations(lines []string) {
	sort.Sort(&annotationSorter{lines: lines})
}

func includesAnnotation(lines []string, annotation string) bool {
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		key := strings.TrimSpace(parts[0])
		if key == annotation {
			return true
		}
	}
	return false
}

func (a *manifestAnnotator) processAnnotations(lines []string, out *bytes.Buffer) bool {
	changed := false
	shouldSkip := false

	if includesAnnotation(lines, a.SkipAnnotation) {
		shouldSkip = true
	}

	if !shouldSkip && !includesAnnotation(lines, a.Annotation) {
		changed = true
		lines = append(lines, fmt.Sprintf("    %s: %q", a.Annotation, a.Value))
	}

	if changed {
		sortAnnotations(lines)
	}
	for _, line := range lines {
		out.WriteString(line + "\n")
	}
	return changed
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
