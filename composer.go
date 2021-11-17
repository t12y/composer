package main

import (
	"fmt"
	"os"

	"github.com/t12y/composer/composer"
)

const errCode = 1

func main() {
	composerFile := os.Getenv("COMPOSER_FILE")
	if composerFile == "" {
		composerFile = "composer.yml"
	}

	cfg, err := composer.ParseConfig(composerFile)
	if err != nil {
		fmt.Println("Cannot parse config:", err)
		os.Exit(errCode)
	}

	if len(os.Args) <= 1 {
		fmt.Printf("\nUsage: %s SERVICE [SERVICE ...]\n\nAvailable services:\n", os.Args[0])
		for service := range cfg.Services {
			fmt.Println(" -", service)
		}
		fmt.Println("")
		os.Exit(errCode)
	}

	services := os.Args[1:]

	var c *composer.Composer
	if c, err = composer.New(*cfg, services...); err != nil {
		fmt.Println("Error initializing composer:", err)
		os.Exit(errCode)
	}

	if err = c.Run(); err != nil {
		fmt.Println("Error running composer:", err)
		os.Exit(errCode)
	}
}
