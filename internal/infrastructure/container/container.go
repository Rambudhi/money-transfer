package container

import (
	"time"

	"go.uber.org/fx"
)

func New() *fx.App {
	return newApp(APIModule)
}

func NewWorker() *fx.App {
	return newApp(WorkerModule)
}

func newApp(module fx.Option) *fx.App {
	return fx.New(
		module,
		fx.StartTimeout(30*time.Second),
		fx.StopTimeout(30*time.Second),
	)
}
