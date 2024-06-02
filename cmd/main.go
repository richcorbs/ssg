package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/joho/godotenv"
)

//go:embed _init
var initFiles embed.FS

type Layout struct {
	Name string
	Path string
}

type Snippet struct {
	Name string
	Path string
}

type FrontMatter struct {
	Layout string `yaml:"layout"`
}

var PORT = "8080"

const DIST = "dist"
const SRC = "src"
const DEFAULT_LAYOUT = "default.html"

// src
// └── assets
// │   ├── css
// │   │   ├── pico.colors.min.css
// │   │   ├── pico.min.css
// │   │   └── styles.css
// │   ├── images
// │   │   └── logo.png
// │   └── js
// │       └── app.js
// ├── snippets
// │   └── Test.html
// ├── layouts
// │   ├── default.html
// │   └── layout.html
// └── pages
//     ├── about.html
//     ├── index.html
//     └── markdown.md

var clients = make(map[chan string]bool)
var layouts []Layout
var snippets []Snippet
var dependencies = make(map[string][]string)

func main() {
	var doBuild bool
	flag.BoolVar(&doBuild, "build", false, "build the site")
	var doDev bool
	flag.BoolVar(&doDev, "dev", false, "build the site and run the dev server")
	var doInit bool
	flag.BoolVar(&doInit, "init", false, "scaffold a site in the current directory")
	var jsFramework string
	flag.StringVar(&jsFramework, "js", "none", "on init, which javascript framework do you want? none, vanjs (default), or alpinejs")
	var doDeploy bool
	flag.BoolVar(&doDeploy, "deploy", false, "deploy built site via scp")
	flag.Parse()

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Loading .env file failed: %s", err)
	}
	if doBuild {
		err := build(false)
		if err != nil {
			log.Fatalf("Build failed: %s", err)
		}
	} else if doDeploy {
		err := build(false)
		if err != nil {
			log.Fatalf("Deploy failed: %s", err)
		}
		deploy()
	} else if doInit {
		err := initializeNewProject(jsFramework)
		if err != nil {
			log.Fatalf("Init failed: %s", err)
		}
	} else if doDev {
		err := build(false)
		if err != nil {
			log.Fatalf("Could not build: %s", err)
		}
		go fileWatcher()
		http.HandleFunc("/", requestHandler)
		http.HandleFunc("/sssg-hot-reload", hotReloadHandler)
		ln, err := net.Listen("tcp", ":"+PORT)
		if err != nil {
			if strings.Contains(err.Error(), "address already in use") {
				for port := 8080; port < 65535; port++ {
					ln, err = net.Listen("tcp", ":"+fmt.Sprint(port))
					if err == nil {
						PORT = fmt.Sprint(port)
						break
					}
				}
			} else {
				log.Fatal(err)
			}
		}
		fmt.Printf("Server started on port %v\n", PORT)
		log.Fatal(http.Serve(ln, nil))
	} else {
		flag.Usage()
		fmt.Println("\nUsage: sssgo option")
		fmt.Println("\n  where option is one of the following:")
		fmt.Println("\n  build    Build the site")
		fmt.Println("  deploy   Build and then deploy the site")
		fmt.Println("  dev      Build the site and start the dev server")
		fmt.Println("  init     Scaffold out a project folder structure and files if they don't already exist")
	}
}
