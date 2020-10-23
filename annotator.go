package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

type manifestAnnotator struct {
	FileName   string
	Annotation string
	Value      string

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
	changed := false
	found := false
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 { // if we can't parse, just write as is
			out.WriteString(line + "\n")
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if key == a.Annotation {
			found = true
			if value != a.Value {
				fmt.Printf("Value does not match, replacing\n")
				fmt.Fprintf(out, "    %s: %q\n", a.Annotation, a.Value)
				changed = true
				continue
			}
		}
		out.WriteString(line + "\n")
	}
	if !found {
		changed = true
		fmt.Fprintf(out, "    %s: %q\n", a.Annotation, a.Value)
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
