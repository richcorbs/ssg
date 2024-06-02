package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

func fileWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	fmt.Println("Watching for changes...")

	err = watchPath(watcher, SRC)
	if err != nil {
		log.Fatal("Error watching path:", err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			startTime := time.Now()
			if !ok {
				return
			}

			var wg sync.WaitGroup
			defer wg.Done()

			// fmt.Println("Event:", event)

			var fileInfo os.FileInfo
			var err error
			if event.Op&fsnotify.Remove != fsnotify.Remove {
				fileInfo, err = os.Stat(event.Name)
			}

			if err != nil {
				fmt.Println("Error:", err)
				return
			} else if event.Op&fsnotify.Create == fsnotify.Create && fileInfo.IsDir() {
				// CREATE DIRECTORY PATH
				watchPath(watcher, event.Name)

				fmt.Println("Creating directory structure:", event.Name)
				err = filepath.Walk(event.Name, buildDirs)
				if err != nil {
					fmt.Println("Error building directories:", err)
				}
			} else if event.Op&fsnotify.Create == fsnotify.Create && strings.HasPrefix(event.Name, "src/assets") {
				// CREATE ASSET
				wg.Add(1)
				go buildPage(event.Name, &wg)
			} else if event.Op&fsnotify.Create == fsnotify.Create && strings.HasPrefix(event.Name, "src/layouts") {
				// CREATE LAYOUT
				err = initializeDependencies()
				if err != nil {
					log.Fatal("Error initializing dependencies:", err)
				}

				for _, path := range dependencies[event.Name] {
					wg.Add(1)
					go buildPage(path, &wg)
				}
			} else if event.Op&fsnotify.Create == fsnotify.Create && strings.HasPrefix(event.Name, "src/pages") {
				// CREATE PAGE
				err = initializeDependencies()
				if err != nil {
					log.Fatal("Error initializing dependencies:", err)
				}

				wg.Add(1)
				go buildPage(event.Name, &wg)
			} else if event.Op&fsnotify.Create == fsnotify.Create && strings.HasPrefix(event.Name, "src/snippets") {
				// CREATE SNIPPET
				err = initializeSnippets()
				if err != nil {
					log.Fatal("Error initializing dependencies:", err)
				}

				err = initializeDependencies()
				if err != nil {
					log.Fatal("Error initializing dependencies:", err)
				}

				for _, path := range dependencies[event.Name] {
					wg.Add(1)
					go buildPage(path, &wg)
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove && strings.HasPrefix(event.Name, "src/assets") {
				// DELETE ASSET
				distPath := event.Name
				distPath = replaceAWithB(distPath, "src/", "dist/")
				fmt.Println("Deleting from dist:", distPath)
				_, err := os.Stat(distPath)
				if err == nil {
					err = os.RemoveAll(distPath)
					if err != nil {
						fmt.Println("Error deleting:", distPath, err)
					}
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove && strings.HasPrefix(event.Name, "src/layouts") {
				// DELETE LAYOUT
				err = initializeLayouts()
				if err != nil {
					log.Fatal("Error initializing layouts:", err)
				}

				err = initializeDependencies()
				if err != nil {
					log.Fatal("Error initializing dependencies:", err)
				}

				for _, path := range dependencies[event.Name] {
					wg.Add(1)
					go buildPage(path, &wg)
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove && strings.HasPrefix(event.Name, "src/pages") {
				// DELETE PAGE
				err = initializeDependencies()
				if err != nil {
					log.Fatal("Error initializing dependencies:", err)
				}

				distPath := event.Name
				distPath = replaceAWithB(distPath, "src/", "dist/")
				distPath = replaceAWithB(distPath, "pages/", "")
				distPath = replaceAWithB(distPath, ".md", ".html")
				fmt.Println("Deleting from dist:", distPath)
				_, err := os.Stat(distPath)
				if err == nil {
					err := os.RemoveAll(distPath)
					if err != nil {
						fmt.Println("Error deleting:", distPath, err)
					}
				}
			} else if event.Op&fsnotify.Remove == fsnotify.Remove && strings.HasPrefix(event.Name, "src/snippets") {
				// DELETE SNIPPET
				err = initializeSnippets()
				if err != nil {
					log.Fatal("Error initializing snippets:", err)
				}

				for _, path := range dependencies[event.Name] {
					wg.Add(1)
					go buildPage(path, &wg)
				}

				err = initializeDependencies()
				if err != nil {
					log.Fatal("Error initializing dependencies:", err)
				}
			} else if event.Op&fsnotify.Write == fsnotify.Write && strings.HasPrefix(event.Name, "src/assets") {
				// UPDATE ASSET
				wg.Add(1)
				go buildPage(event.Name, &wg)
			} else if event.Op&fsnotify.Write == fsnotify.Write && strings.HasPrefix(event.Name, "src/layouts") {
				// UPDATE LAYOUT
				err = initializeDependencies()
				if err != nil {
					log.Fatal("Error initializing dependencies:", err)
				}

				for _, path := range dependencies[event.Name] {
					wg.Add(1)
					go buildPage(path, &wg)
				}
			} else if event.Op&fsnotify.Write == fsnotify.Write && strings.HasPrefix(event.Name, "src/pages") {
				// UPDATE PAGE
				err = initializeDependencies()
				if err != nil {
					log.Fatal("Error initializing dependencies:", err)
				}

				wg.Add(1)
				go buildPage(event.Name, &wg)

				for _, path := range dependencies[event.Name] {
					fmt.Println(path)
					wg.Add(1)
					go buildPage(path, &wg)
				}
			} else if event.Op&fsnotify.Write == fsnotify.Write && strings.HasPrefix(event.Name, "src/snippets") {
				// UPDATE SNIPPET
				err := initializeSnippets()
				if err != nil {
					log.Fatal("Error initializing snippets:", err)
				}

				err = initializeDependencies()
				if err != nil {
					log.Fatal("Error initializing dependencies:", err)
				}

				for _, path := range dependencies[event.Name] {
					wg.Add(1)
					go buildPage(path, &wg)
				}
			}

			wg.Wait()
			fmt.Printf("Re-build complete: %s\n", time.Since(startTime))
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Error:", err)
		}
		broadcast("RELOAD")
	}
}

func hotReloadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Client connected")
	messageChan := make(chan string)
	clients[messageChan] = true
	defer func() {
		delete(clients, messageChan)
		close(messageChan)
	}()

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Set headers to enable SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Listen for messages on the messageChan and send them to the client
	for {
		select {
		case message := <-messageChan:
			fmt.Fprintf(w, "data: %s\n\n", message)
			flusher.Flush()
		case <-r.Context().Done():
			fmt.Println("Client disconnected")
			return
		}
	}
}

func requestHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method is not supported", http.StatusNotFound)
		return
	}

	path := r.URL.Path

	if strings.HasSuffix(path, "/") {
		// Look for /index.html
		_, err := http.Dir(DIST).Open(path + "index.html")
		if err == nil {
			path += "index.html"
		}
	}

	// Check if file exists
	contentBytes, err := os.ReadFile(DIST + path)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	content := string(contentBytes)
	var contentWithSSE string

	fmt.Println("Request for", r.URL.Path)

	switch {
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css")
		contentWithSSE = content
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "text/javascript")
		contentWithSSE = content
	case strings.HasSuffix(path, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
		contentWithSSE = content
	case strings.HasSuffix(path, ".png"):
		w.Header().Set("Content-Type", "image/png")
		contentWithSSE = content
	default:
		hotReloadScript := `
            <script>
                let eventSource = new EventSource("/sssg-hot-reload");
                eventSource.onmessage = (event) => { window.location.reload() };
                eventSource.onerror = (event) => { console.log('ERROR', JSON.stringify(event, null, 2)) };
                eventSource.onopen = (event) => { console.log('OPEN', JSON.stringify(event, null, 2)) };
                eventSource.onclose = (event) => { console.log('CLOSED', JSON.stringify(event, null, 2)) };
            </script>
        `

		contentWithSSE = replaceAWithB(content, "</body>", hotReloadScript+"</body>")

		w.Header().Set("Content-Type", "text/html")
	}
	http.ServeContent(w, r, r.URL.Path, time.Now(), strings.NewReader(contentWithSSE))
}

func watchPath(watcher *fsnotify.Watcher, path string) error {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})
	return err
}

// ORIGINAL NAIVE RELOAD ON FILE CHANGE CODE THAT REBUILT *everything*!
// if event.Op&fsnotify.Write == fsnotify.Write ||
// 	event.Op&fsnotify.Create == fsnotify.Create ||
// 	event.Op&fsnotify.Remove == fsnotify.Remove {
// 	fmt.Println("Content changed: ", event)
// 	fmt.Printf("Dependencies for %v: %v\n", event.Name, dependencies[event.Name])
// 	err = build(true)
// 	if err != nil {
// 		fmt.Println(err)
// 	}
// }
