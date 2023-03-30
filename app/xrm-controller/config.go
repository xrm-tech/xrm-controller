package xrmcontroller

import (
	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/msaf1980/fiberlog"
	"github.com/rs/zerolog"
)

type Config struct {
	StoreDir      string
	OVirtStoreDir string
	Listen        string
	TLSCert       string
	TLSKey        string
	Users         map[string]string
	Logger        zerolog.Logger
}

var (
	Cfg Config
)

func RouterInit() (app *fiber.App) {
	app = fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: Decode,
	})

	app.Use(fiberlog.New(fiberlog.Config{
		Logger: &Cfg.Logger,
		Next: func(ctx *fiber.Ctx) bool {
			return false
		},
		LogUsername: "username",
	}))

	// enable basic auth
	app.Use(basicauth.New(basicauth.Config{Users: Cfg.Users}))

	// OVirt
	app.Get("/ovirt/cleanup/:name", oVirtCleanup)
	app.Post("/ovirt/generate/:name", oVirtGenerate)
	app.Get("/ovirt/failover/:name", oVirtFailover)
	app.Get("/ovirt/failback/:name", oVirtFailback)

	return
}
