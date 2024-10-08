package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/russross/blackfriday/v2"
)

func broadcast(message string) {
	for messageChan := range clients {
		messageChan <- message
	}
}

func build(reload bool) error {
	startTime := time.Now()
	fmt.Println("Building...")

	err := initializeSnippets()
	if err != nil {
		log.Fatal("Error initializing snippets", err)
	}

	err = initializeLayouts()
	if err != nil {
		log.Fatal("Error initializing layouts:", err)
	}

	err = initializeDependencies()
	if err != nil {
		log.Fatal("Error initializing dependencies:", err)
	}

	_, err = os.Stat(DIST)
	if err == nil {
		err = os.RemoveAll(DIST)
		if err != nil {
			fmt.Println("Error:", err)
		}
	}

	err = os.Mkdir(DIST, 0744)
	if err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Println("Creating directory structure...")
	err = filepath.Walk(SRC, buildDirs)
	if err != nil {
		fmt.Println("Error:", err)
	}

	err = filepath.Walk(SRC, buildPages)
	if err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Printf("Build complete: %s\n", time.Since(startTime))

	if reload {
		fmt.Println("Reloading browser...")
		broadcast("RELOAD")
	}
	return nil
}

func buildDirs(srcPath string, info os.FileInfo, err error) error {
	distPath := replaceAWithB(srcPath, "src/", DIST+"/")
	distPath = replaceAWithB(distPath, "src", DIST)
	distPath = replaceAWithB(distPath, "/pages", "")

	if info.IsDir() && !strings.HasPrefix(srcPath, SRC+"/snippets") && !strings.HasPrefix(srcPath, SRC+"/layouts") {
		fmt.Printf("  %s -> %s\n", srcPath, distPath)
		_, err := os.Stat(distPath)
		if err != nil {
			err = os.Mkdir(distPath, 0755)
			if err != nil {
				fmt.Println("Error:", err)
			}
		}
	} else if info.IsDir() {
		fmt.Println("  Skipping", srcPath)
	}

	return nil
}

func buildPages(srcPath string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Println("Error:", err)
	}

	if strings.HasPrefix(srcPath, SRC+"/snippets/") || strings.HasPrefix(srcPath, SRC+"/layouts/") {
		fmt.Println("  Skipping", srcPath)
		return nil
	}

	var wg sync.WaitGroup

	if !info.IsDir() {
		wg.Add(1)
		go buildPage(srcPath, &wg)
	}

	wg.Wait()

	return nil
}

func buildPage(srcPath string, wg *sync.WaitGroup) {
	defer wg.Done()

	distPath := replaceAWithB(srcPath, "src/", "dist/")
	distPath = replaceAWithB(distPath, "src", "dist")
	distPath = replaceAWithB(distPath, "/pages", "")

	data, err := os.ReadFile(srcPath)
	if err != nil {
		fmt.Println("Error:", err)
	}

	var destPath string
	var wrappedData []byte

	switch {
	case strings.HasSuffix(distPath, ".md"):
		// parse markdown to html
		data = blackfriday.Run(data)
		destPath = replaceAWithB(distPath, ".md", ".html")
	case strings.HasSuffix(distPath, ".html"):
		destPath = distPath
	default:
		// assets files: css, js, etc
		wrappedData = data
		destPath = distPath
	}

	if strings.HasSuffix(destPath, ".html") {
		dataWithSnippets := processSnippets(data)
		wrappedData = wrapHtmlInLayout(dataWithSnippets)
	}

	fmt.Printf("  %s -> %s\n", srcPath, destPath)

	err = os.WriteFile(destPath, wrappedData, 0644)
	if err != nil {
		fmt.Println("Error:", err)
	}
}

func initializeSnippets() error {
	fmt.Println("Initializing snippets...")

	snippets = []Snippet{}

	if _, err := os.Stat(SRC + "/snippets"); os.IsNotExist(err) {
		fmt.Println("Skipping snippets initialization. Directory does not exist.")
		return nil
	}

	err := filepath.Walk(SRC+"/snippets", func(path string, info os.FileInfo, err error) error {
		var snippet Snippet
		if !info.IsDir() {
			base := filepath.Base(path)
			snippet.Name = strings.TrimSuffix(base, filepath.Ext(base))
			snippet.Path = path
			snippets = append(snippets, snippet)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func initializeDependencies() error {
	fmt.Println("Initializing dependencies...")

	dependencies = make(map[string][]string)

	filepath.Walk(SRC, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if strings.Contains(path, "src/snippets/") || strings.Contains(path, "src/layouts/") {
			// TODO: refactor this to consider nested snippets
			// TODO: refactor this to consider snippets in a layout. Would that mean cascading dependencies?
			return nil
		}

		file := filepath.Base(path)
		ext := filepath.Ext(file)

		if ext == ".html" || ext == ".md" {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			foundLayout := false

			for _, layout := range layouts {
				if strings.HasPrefix(string(content), "<"+layout.Name+"Layout>") && (strings.HasSuffix(string(content), "</"+layout.Name+"Layout>") || strings.HasSuffix(string(content), "</"+layout.Name+"Layout>\n")) {
					foundLayout = true
					if !sliceContains(path, dependencies[layout.Path]) {
						dependencies[layout.Path] = append(dependencies[layout.Path], path)
					}
				}
			}

			if foundLayout == false {
				if !sliceContains(path, dependencies[DEFAULT_LAYOUT]) {
					dependencies[DEFAULT_LAYOUT] = append(dependencies[DEFAULT_LAYOUT], path)
				}
			}

			for _, snippet := range snippets {
				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				if strings.Contains(string(content), "<"+snippet.Name+"></"+snippet.Name+">") {
					if !sliceContains(path, dependencies[snippet.Path]) {
						dependencies[snippet.Path] = append(dependencies[snippet.Path], path)
					}
				}
			}
		}
		return nil
	})
	return nil
}

func initializeLayouts() error {
	fmt.Println("Initializing layouts...")

	layouts = []Layout{}

	if _, err := os.Stat(SRC + "/layouts"); os.IsNotExist(err) {
		return fmt.Errorf("%v/layouts not found", SRC)
	}
	err := filepath.Walk(SRC+"/layouts", func(path string, info os.FileInfo, err error) error {
		var layout Layout
		if !info.IsDir() {
			base := filepath.Base(path)
			layout.Name = strings.TrimSuffix(base, filepath.Ext(base))
			layout.Path = path
			layouts = append(layouts, layout)
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func processSnippets(data []byte) []byte {
	fmt.Println("Processing snippets...")
	content := string(data)
	for _, snippet := range snippets {
		snippetString := "<" + snippet.Name + "></" + snippet.Name + ">"
		if strings.Contains(content, snippetString) {
			snippetContent, err := os.ReadFile(snippet.Path)
			if err != nil {
				fmt.Println(err)
			}

			content = replaceAWithB(content, snippetString, string(snippetContent))
		}
	}
	return []byte(content)
}

func wrapHtmlInLayout(data []byte) []byte {
	fmt.Println("Wrapping in layout...")
	defaultLayoutPath := SRC + "/layouts/Default.html"

	var wrappedData string
	var unwrappedData string
	var rawLayout []byte
	var layout string
	var err error
	var openTag string
	var closeTag string

	for _, layout := range layouts {
		openTag = "<" + layout.Name + "Layout>"
		closeTag = "</" + layout.Name + "Layout>"
		if strings.HasPrefix(string(data), openTag) && strings.Contains(string(data), closeTag) {
			layoutPath := layout.Path
			_, err := os.Stat(layoutPath)
			if err == nil {
				rawLayout, err = os.ReadFile(layoutPath)
				if err != nil {
					fmt.Println("Error:", err)
				}
			}
			unwrappedData = replaceAWithB(string(data), openTag, "")
			unwrappedData = replaceAWithB(unwrappedData, closeTag, "")
		}
	}

	if len(rawLayout) == 0 {
		unwrappedData = replaceAWithB(string(data), "<DefaultLayout>", "")
		unwrappedData = replaceAWithB(unwrappedData, "</DefaultLayout>", "")
		rawLayout, err = os.ReadFile(defaultLayoutPath)
		if err != nil {
			fmt.Println("Error:", err)
		}
	}

	layout = string(rawLayout)
	wrappedData = replaceAWithB(layout, "__CONTENT__", unwrappedData)

	return []byte(wrappedData)
}
