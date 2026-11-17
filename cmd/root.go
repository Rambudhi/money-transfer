package cmd

import (
	"os"
)

func StartServer() {
	if len(os.Args) > 1 && os.Args[1] == "worker" {
		RunWorker()
		return
	}

	RunAPI()
}
