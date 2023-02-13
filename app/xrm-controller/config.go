package xrmcontroller

import (
	"github.com/go-playground/validator/v10"
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
}

var (
	Cfg      Config
	Validate = validator.New()
)

func RouterInit(logger *zerolog.Logger) (app *fiber.App) {
	app = fiber.New(fiber.Config{
		JSONEncoder: json.Marshal,
		JSONDecoder: Decode,
	})

	app.Use(fiberlog.New(fiberlog.Config{
		Logger: logger,
		Next: func(ctx *fiber.Ctx) bool {
			return false
		},
		LogUsername: "username",
	}))

	// enable basic auth
	app.Use(basicauth.New(basicauth.Config{Users: Cfg.Users}))

	// OVirt
	app.Get("/ovirt/cleanup/:name<regex([a-zA-Z_\\-0-9]+)>", oVirtCleanup)
	app.Post("/ovirt/generate/:name<regex([a-zA-Z_\\-0-9]+)>", oVirtGenerate)

	return
}
