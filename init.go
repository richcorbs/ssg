package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func copyInitDir(source, destination, framework string) error {
	entries, err := initFiles.ReadDir(source)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(destination, 0755); err != nil {
		return err
	}

	// TODO: Handle framework types of "none", "vanjs", and "alpinejs"

	for _, entry := range entries {
		src := filepath.Join(source, entry.Name())
		dst := filepath.Join(destination, entry.Name())

		if entry.IsDir() {
			if err := copyInitDir(src, dst, framework); err != nil {
				return err
			}
		} else {
			if err := copyInitFile(src, dst); err != nil {
				return err
			}
		}
	}

	return nil
}

func copyInitFile(source, destination string) error {
	src, err := initFiles.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}
	return nil
}

func initializeNewProject(framework string) error {
	var destination = "."
	err := copyInitDir("_init", destination, framework)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}
