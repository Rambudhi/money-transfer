package cmd

import (
	"fmt"

	"github.com/Rambudhi/money-transfer/internal/infrastructure/container"
	"github.com/Rambudhi/money-transfer/internal/pkg/log"
)

const workerBanner = `Money Transfer Worker`

func RunWorker() {
	fmt.Println(workerBanner)

	log.Init()

	app := container.NewWorker()
	app.Run()
}
