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
	"gopkg.in/yaml.v2"
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

		if ext == ".html" {
			var layout string
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			frontMatter, _ := parseFrontMatter(string(content))
			if len(frontMatter.Layout) > 0 {
				layout = frontMatter.Layout
			} else if layout == "" {
				layout = DEFAULT_LAYOUT
			}

			if layout != "" {
				layoutPath := SRC + "/layouts/" + layout
				if !sliceContains(path, dependencies[layoutPath]) {
					dependencies[layoutPath] = append(dependencies[layoutPath], path)
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

func parseFrontMatter(content string) (FrontMatter, string) {
	var frontMatter FrontMatter
	parts := strings.SplitN(content, "---", 3)

	if len(parts) != 3 {
		return frontMatter, content
	}
	if err := yaml.Unmarshal([]byte(parts[1]), &frontMatter); err != nil {
		return frontMatter, parts[2]
	}
	return frontMatter, parts[2]
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
	defaultLayoutPath := SRC + "/layouts/" + DEFAULT_LAYOUT
	frontMatter, body := parseFrontMatter(string(data))

	var wrappedData string
	var rawLayout []byte
	var layout string
	var err error

	if len(frontMatter.Layout) > 0 {
		layoutPath := SRC + "/layouts/" + frontMatter.Layout
		_, err := os.Stat(layoutPath)
		if err == nil {
			rawLayout, err = os.ReadFile(layoutPath)
			if err != nil {
				fmt.Println("Error:", err)
			}
		} else {
			rawLayout, err = os.ReadFile(defaultLayoutPath)
			if err != nil {
				fmt.Println("Error:", err)
			}
		}
	} else {
		rawLayout, err = os.ReadFile(defaultLayoutPath)
		if err != nil {
			fmt.Println("Error:", err)
		}
	}

	layout = string(rawLayout)
	wrappedData = replaceAWithB(layout, "__CONTENT__", string(body))

	return []byte(wrappedData)
}
