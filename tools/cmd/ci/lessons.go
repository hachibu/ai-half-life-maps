package main

// Lessons: docs/lessons/*.md, one rule per file. `ci lessons` checks every file's format and writes the index of
// rules into AGENTS.md (between <!-- lessons:start --> and <!-- lessons:end -->), grouped by area, so every agent
// sees every rule in the file it reads first. `ci lessons --check` fails if a lesson is malformed or the index is
// out of date.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Areas a lesson can belong to, in the order AGENTS.md lists them.
var lessonAreas = []string{"workflow", "geometry", "lighting", "textures", "entities", "content", "ci", "github", "go"}

// areaNames are the headings AGENTS.md uses for the areas.
var areaNames = map[string]string{"workflow": "Workflow", "geometry": "Geometry", "lighting": "Lighting", "textures": "Textures",
	"entities": "Entities", "content": "Content", "ci": "CI", "github": "GitHub", "go": "Go"}

var (
	errBadLessons = errors.New("lessons need fixing")
	reFrontMatter = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)$`)
	reSection     = regexp.MustCompile(`(?m)^## (.+?)\s*$`)
)

type lesson struct {
	file, title, area string
	tags              []string
}

// parseLesson reads one lesson and lists what is wrong with it.
func parseLesson(path string, maps map[string]bool) (lesson, []string) {
	l := lesson{file: filepath.Base(path)}
	b, err := os.ReadFile(path)
	if err != nil {
		return l, []string{err.Error()}
	}
	m := reFrontMatter.FindStringSubmatch(strings.ReplaceAll(string(b), "\r\n", "\n"))
	if m == nil {
		return l, []string{"no front matter (--- title / area / tags / source ---)"}
	}
	meta := map[string]string{}
	for line := range strings.SplitSeq(m[1], "\n") {
		if k, v, ok := strings.Cut(line, ":"); ok {
			meta[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	l.title, l.area = meta["title"], meta["area"]
	for t := range strings.SplitSeq(strings.Trim(meta["tags"], "[]"), ",") {
		if t = strings.Trim(strings.TrimSpace(t), `'"`); t != "" {
			l.tags = append(l.tags, t)
		}
	}
	var bad []string
	if l.title == "" || strings.HasSuffix(l.title, ".") {
		bad = append(bad, "title must be the rule itself, one line, no trailing period")
	}
	if !slices.Contains(lessonAreas, l.area) {
		bad = append(bad, fmt.Sprintf("area %q is not one of %s", l.area, strings.Join(lessonAreas, ", ")))
	}
	for _, t := range l.tags {
		if !maps[t] {
			bad = append(bad, fmt.Sprintf("tag %q is not a map (tags name the maps a lesson is about; the area covers the rest)", t))
		}
	}
	if meta["source"] == "" {
		bad = append(bad, "source is empty (the PR, map or change it came from)")
	}
	body := m[2]
	heads := reSection.FindAllStringSubmatchIndex(body, -1)
	sections := map[string]string{}
	for i, h := range heads {
		end := len(body)
		if i+1 < len(heads) {
			end = heads[i+1][0]
		}
		sections[strings.ToLower(body[h[2]:h[3]])] = strings.TrimSpace(body[h[1]:end])
	}
	for _, s := range []string{"what happened", "why", "rule"} {
		if sections[s] == "" {
			bad = append(bad, fmt.Sprintf("section '## %s' is missing or empty", strings.ToUpper(s[:1])+s[1:]))
		}
	}
	return l, bad
}

func lessonsIndex(ls []lesson) string {
	var out []string
	for _, area := range lessonAreas {
		var lines []string
		for _, l := range ls {
			if l.area != area {
				continue
			}
			line := fmt.Sprintf("- %s ([details](docs/lessons/%s))", l.title, l.file)
			if len(l.tags) > 0 {
				line = fmt.Sprintf("- %s (%s; [details](docs/lessons/%s))", l.title, strings.Join(l.tags, ", "), l.file)
			}
			lines = append(lines, line)
		}
		if len(lines) > 0 {
			out = append(out, "**"+areaNames[area]+"**", "")
			out = append(out, lines...)
			out = append(out, "")
		}
	}
	return strings.Join(out, "\n")
}

func cmdLessons(root string, check bool) (int, error) {
	maps := map[string]bool{}
	for _, d := range subdirs(filepath.Join(root, "maps")) {
		maps[filepath.Base(d)] = true
	}
	paths, err := filepath.Glob(filepath.Join(root, "docs", "lessons", "*.md"))
	if err != nil {
		return 1, err
	}
	var ls []lesson
	problems := 0
	for _, p := range paths {
		if filepath.Base(p) == "README.md" {
			continue
		}
		l, bad := parseLesson(p, maps)
		for _, b := range bad {
			fmt.Printf("%s: %s\n", filepath.Base(p), b)
			problems++
		}
		ls = append(ls, l)
	}
	if problems > 0 {
		return 1, errBadLessons
	}
	agents := filepath.Join(root, "AGENTS.md")
	b, err := os.ReadFile(agents)
	if err != nil {
		return 1, err
	}
	updated, err := replaceBlock(string(b), "lessons", lessonsIndex(ls), false)
	if err != nil {
		return 1, err
	}
	switch {
	case updated == string(b):
		fmt.Printf("lessons: %d ok, AGENTS.md index up to date\n", len(ls))
	case check:
		fmt.Println("AGENTS.md lessons index is out of date (run `make readme`)")
		return 1, nil
	default:
		if err := os.WriteFile(agents, []byte(updated), 0o644); err != nil {
			return 1, err
		}
		fmt.Printf("lessons: %d ok, AGENTS.md index updated\n", len(ls))
	}
	return 0, nil
}
