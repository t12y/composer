package main

import (
	"flag"
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

	var waitForAll bool
	flag.BoolVar(&waitForAll, "wait", false, "wait for all services to finish")
	flag.Parse()

	if len(os.Args) <= 1 {
		fmt.Printf("\nUsage: %s [options] SERVICE [SERVICE ...]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()

		fmt.Println("\nServices:")
		for service := range cfg.Services {
			fmt.Println(" -", service)
		}
		fmt.Println("")
		os.Exit(errCode)
	}

	services := flag.Args()

	var c *composer.Composer
	if c, err = composer.New(*cfg, services...); err != nil {
		fmt.Println("Error initializing composer:", err)
		os.Exit(errCode)
	}

	if waitForAll {
		err = c.RunAll(services...)
	} else {
		err = c.Run()
	}

	if err != nil {
		fmt.Println("Error running composer:", err)
		os.Exit(errCode)
	}
}
