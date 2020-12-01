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

func (a *manifestAnnotator) processAnnotations(lines []string, out *bytes.Buffer) bool {
	annotations := parseAnnotations(lines)
	if !annotations.Includes(a.Annotation) && !annotations.Includes(a.SkipAnnotation) {
		annotations.Add(a.Annotation, a.Value)
		annotations.Sort()
		annotations.Write(out)
		return true
	}
	annotations.Write(out)
	return false
}

type annotation struct {
	key   string
	lines []string
}

type annotations []annotation

func (a annotations) Len() int {
	return len(a)
}

func (a annotations) Less(i, j int) bool {
	return a[i].key < a[j].key
}

func (a annotations) Swap(i, j int) {
	tmp := a[i]
	a[i] = a[j]
	a[j] = tmp
}

func (a annotations) Sort() {
	sort.Sort(a)
}

func (a annotations) Includes(key string) bool {
	for _, aa := range a {
		if aa.key == key {
			return true
		}
	}
	return false
}

func (a annotations) Write(out *bytes.Buffer) {
	for _, aa := range a {
		for _, line := range aa.lines {
			out.WriteString(line + "\n")
		}
	}
}

func (a *annotations) Add(key, value string) {
	*a = append(*a, newAnnotation(key, value))
}

func newAnnotation(key, value string) annotation {
	return annotation{
		key:   key,
		lines: []string{fmt.Sprintf("    %s: %s", key, value)},
	}
}

func parseAnnotations(lines []string) annotations {
	var currentAnnotation *annotation
	result := annotations{}
	for _, line := range lines {
		if strings.HasPrefix(line, "      ") {
			if currentAnnotation != nil {
				currentAnnotation.lines = append(currentAnnotation.lines, line)
			}
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) < 2 {
			continue
		}
		if currentAnnotation != nil {
			result = append(result, *currentAnnotation)
		}
		currentAnnotation = &annotation{
			key:   strings.TrimSpace(parts[0]),
			lines: []string{line},
		}
	}
	if currentAnnotation != nil {
		result = append(result, *currentAnnotation)
	}
	return result
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
