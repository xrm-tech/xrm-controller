package xrmcontroller

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/xrm-tech/xrm-controller/ovirt"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

func oVirtCleanup(c *fiber.Ctx) error {
	if err := ovirt.Cleanup(c.Params("name"), Cfg.OVirtStoreDir); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Status{Message: err.Error()})
	}
	status := Status{Message: "success"}
	return c.Status(http.StatusOK).JSON(&status)
}

func oVirtGenerate(c *fiber.Ctx) (err error) {
	var (
		sitesConfig ovirt.GenerateVars
		out         string
	)
	if err = c.BodyParser(&sitesConfig); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := sitesConfig.Validate(); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidatorError(err.(validator.ValidationErrors)))
	}
	name := c.Params("name")

	if out, err = sitesConfig.Generate(name, Cfg.OVirtStoreDir); err == nil {
		// TODO: debug loglevel ??
		Cfg.Logger.Info().Str("out", out)
		return c.Status(http.StatusOK).JSON(Status{Message: "success"})
	} else {
		Cfg.Logger.Error().Err(err).Str("out", out)
		return c.Status(http.StatusInternalServerError).JSON(Status{Message: err.Error()})
	}
}
