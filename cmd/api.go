package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/Rambudhi/money-transfer/internal/infrastructure/container"
	"github.com/Rambudhi/money-transfer/internal/pkg/log"
)

const banner = `Money Transfer API`

func RunAPI() {
	initialize()

	fmt.Println(banner)

	app := container.New()
	app.Run()
}

func initialize() {
	log.Init()

	loadTimezone()
}

func loadTimezone() {
	tz := os.Getenv("TZ")
	if tz == "" {
		return
	}

	location, err := time.LoadLocation(tz)
	if err != nil {
		fmt.Printf("error loading location '%s': %v\n", tz, err)
		return
	}

	time.Local = location
	fmt.Printf("location loaded '%s'\n", tz)
}
