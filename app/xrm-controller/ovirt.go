package xrmcontroller

import (
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/xrm-tech/xrm-controller/ovirt"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

func oVirtDelete(c *fiber.Ctx) error {
	if err := ovirt.Delete(c.Params("name"), Cfg.OVirtStoreDir); err != nil {
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}
	return c.Status(http.StatusOK).SendString("success")
}

func oVirtGenerate(c *fiber.Ctx) (err error) {
	var (
		sitesConfig ovirt.GenerateVars
		out         string
	)
	if err = c.BodyParser(&sitesConfig); err != nil {
		Cfg.Logger.Error().Err(err)
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := sitesConfig.Validate(); err != nil {
		Cfg.Logger.Error().Err(err)
		return c.Status(fiber.StatusBadRequest).JSON(utils.ValidatorError(err.(validator.ValidationErrors)))
	}
	name := c.Params("name")

	if out, err = sitesConfig.Generate(name, Cfg.OVirtStoreDir); err == nil {
		// TODO: debug loglevel ??
		Cfg.Logger.Info().Str("out", out)
		return c.Status(http.StatusOK).SendString(out)
	} else {
		Cfg.Logger.Error().Err(err).Str("out", out)
		return fiber.NewError(http.StatusInternalServerError, err.Error()+"\n"+out)
	}
}

func oVirtFailover(c *fiber.Ctx) (err error) {
	var (
		out string
	)
	name := c.Params("name")

	if out, err = ovirt.Failover(name, Cfg.OVirtStoreDir); err == nil {
		// TODO: debug loglevel ??
		Cfg.Logger.Info().Str("out", out)
		return c.Status(http.StatusOK).SendString(out)
	} else {
		Cfg.Logger.Error().Err(err).Str("out", out)
		return fiber.NewError(http.StatusInternalServerError, err.Error()+"\n"+out)
	}
}

func oVirtFailback(c *fiber.Ctx) (err error) {
	var (
		out string
	)
	name := c.Params("name")

	if out, err = ovirt.Failback(name, Cfg.OVirtStoreDir); err == nil {
		// TODO: debug loglevel ??
		Cfg.Logger.Info().Str("out", out)
		return c.Status(http.StatusOK).SendString(out)
	} else {
		Cfg.Logger.Error().Err(err).Str("out", out)
		return fiber.NewError(http.StatusInternalServerError, err.Error()+"\n"+out)
	}
}
