package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

type routeEntry struct {
	File     string   `json:"file"`
	Group    string   `json:"group"`
	Prefix   string   `json:"prefix"`
	Path     string   `json:"path"`
	FullPath string   `json:"full_path"`
	Class    string   `json:"class"`
	Methods  []string `json:"methods"`
}

var (
	routeDecoratorRE = regexp.MustCompile(`^\s*@[\w\.]+\.route\((?:"([^"]+)"|'([^']+)')`)
	classRE          = regexp.MustCompile(`^\s*class\s+([A-Za-z0-9_]+)\s*\(`)
	methodRE         = regexp.MustCompile(`^\s*def\s+(get|post|put|patch|delete|head|options)\s*\(`)
)

func main() {
	source := flag.String("source", "../dify/api/controllers", "path to Dify Python controllers")
	output := flag.String("output", "docs/route-manifest.json", "output manifest path")
	flag.Parse()

	entries, err := extractRoutes(*source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "extract routes: %v\n", err)
		os.Exit(1)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal routes: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(*output), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir output dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*output, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}
}

func extractRoutes(root string) ([]routeEntry, error) {
	var entries []routeEntry

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".py" {
			return nil
		}

		group, prefix := controllerGroup(path)
		if group == "" {
			return nil
		}

		fileEntries, err := extractRoutesFromFile(root, path, group, prefix)
		if err != nil {
			return err
		}
		entries = append(entries, fileEntries...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	slices.SortFunc(entries, func(a, b routeEntry) int {
		if a.FullPath == b.FullPath {
			return strings.Compare(a.File, b.File)
		}
		return strings.Compare(a.FullPath, b.FullPath)
	})

	return entries, nil
}

func extractRoutesFromFile(root, path, group, prefix string) ([]routeEntry, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var (
		entries      []routeEntry
		pendingPaths []string
		currentClass string
	)

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		if matches := routeDecoratorRE.FindStringSubmatch(line); len(matches) > 0 {
			route := firstNonEmpty(matches[1], matches[2])
			if route != "" {
				pendingPaths = append(pendingPaths, route)
			}
			continue
		}

		if matches := classRE.FindStringSubmatch(line); len(matches) > 0 {
			currentClass = matches[1]
			for _, routePath := range pendingPaths {
				entries = append(entries, routeEntry{
					File:     relativeFile(root, path),
					Group:    group,
					Prefix:   prefix,
					Path:     routePath,
					FullPath: joinRoute(prefix, routePath),
					Class:    currentClass,
				})
			}
			pendingPaths = nil
			continue
		}

		if matches := methodRE.FindStringSubmatch(line); len(matches) > 0 && currentClass != "" {
			method := strings.ToUpper(matches[1])
			for i := len(entries) - 1; i >= 0; i-- {
				if entries[i].Class != currentClass {
					break
				}
				if !slices.Contains(entries[i].Methods, method) {
					entries[i].Methods = append(entries[i].Methods, method)
				}
			}
		}
	}

	return entries, nil
}

func controllerGroup(path string) (group string, prefix string) {
	switch {
	case strings.Contains(path, "/controllers/console/"):
		return "console", "/console/api"
	case strings.Contains(path, "/controllers/web/"):
		return "web", "/api"
	case strings.Contains(path, "/controllers/files/"):
		return "files", "/files"
	case strings.Contains(path, "/controllers/inner_api/"):
		return "inner_api", "/inner/api"
	case strings.Contains(path, "/controllers/service_api/"):
		return "service_api", "/v1"
	case strings.Contains(path, "/controllers/mcp/"):
		return "mcp", "/mcp"
	case strings.Contains(path, "/controllers/trigger/"):
		return "trigger", "/triggers"
	default:
		return "", ""
	}
}

func relativeFile(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func joinRoute(prefix, route string) string {
	if strings.HasPrefix(route, "/") {
		return prefix + route
	}
	return prefix + "/" + route
}
