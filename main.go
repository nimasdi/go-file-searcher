package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/lithammer/fuzzysearch/fuzzy"
)

type SearchResult struct {
	FilePath    string
	LineNumber  int
	LineContent string
}

func main() {
	rootPathPtr := flag.String("path", ".", "Root directory to search")
	patternPtr := flag.String("pattern", "", "Pattern to search for (case-sensitive)")
	numWorkersPtr := flag.Int("workers", runtime.NumCPU(), "Number of concurrent workers")
	extPtr := flag.String("ext", "", "Comma-separated list of file extensions to include (e.g. .go,.txt)")
	flag.Parse()

	rootPath := *rootPathPtr
	pattern := *patternPtr
	numWorkers := *numWorkersPtr
	extFilter := *extPtr

	extMap := make(map[string]struct{})
	if extFilter != "" {
		for _, ext := range splitAndTrim(extFilter, ",") {
			extMap[ext] = struct{}{}
		}
	}

	if pattern == "" {
		fmt.Println("Please provide a search pattern using the -pattern flag.")
		return
	}

	if numWorkers <= 0 {
		fmt.Println("Number of workers must be greater than 0. Using default number of CPU cores.")
		numWorkers = runtime.NumCPU() / 2
	}

	filePathChannel := make(chan string)
	resultChannel := make(chan SearchResult)

	var wg sync.WaitGroup
	wg.Add(1)
	go walkFiles(rootPath, filePathChannel, &wg, extMap)

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go searchWorker(filePathChannel, resultChannel, pattern, &wg)
	}

	go printResults(resultChannel)

	wg.Wait()

	close(resultChannel)
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, part := range filepath.SplitList(s) {
		for _, sub := range splitComma(part, sep) {
			trimmed := trimSpace(sub)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

func splitComma(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if string(s[i]) == sep {
			result = append(result, s[start:i])
			start = i + 1
		}
	}
	result = append(result, s[start:])
	return result
}

func trimSpace(s string) string {
	return string([]byte(s))
}

func walkFiles(rootPath string, filePathChannel chan<- string, wg *sync.WaitGroup, extMap map[string]struct{}) {
	defer wg.Done()
	defer close(filePathChannel)

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error accessing path %q: %v\n", path, err)
			return nil
		}
		if !info.IsDir() {
			if len(extMap) > 0 {
				ext := filepath.Ext(path)
				if _, ok := extMap[ext]; !ok {
					return nil
				}
			}
			filePathChannel <- path
		}
		return nil
	})
	if err != nil {
		fmt.Println("Error walking the path:", err)
	}
}

func searchWorker(filePathChannel <-chan string, resultChannel chan<- SearchResult, pattern string, wg *sync.WaitGroup) {
	defer wg.Done()

	for filePath := range filePathChannel {
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening file %q: %v\n", filePath, err)
			continue // Skip to the next file
		}

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()
			if fuzzy.Match(pattern, line) {
				resultChannel <- SearchResult{
					FilePath:    filePath,
					LineNumber:  lineNum,
					LineContent: line,
				}
			}
		}
		file.Close()
	}
}

func printResults(resultChannel <-chan SearchResult) {
	for res := range resultChannel {
		fmt.Printf("Found in %s:%d: %s\n", res.FilePath, res.LineNumber, res.LineContent)

	}
}
