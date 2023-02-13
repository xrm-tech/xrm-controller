package xrmcontroller

import (
	"net/http"
	"os"
	"path"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/xrm-tech/xrm-controller/ovirt"
)

func oVirtCleanup(c *fiber.Ctx) error {
	path := path.Join(Cfg.OVirtStoreDir, c.Params("name"), "disaster_recovery_vars.yml")
	if err := ovirt.Cleanup(path); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(Status{Message: err.Error()})
	}
	status := Status{Message: "success"}
	return c.Status(http.StatusOK).JSON(&status)
}

func oVirtGenerate(c *fiber.Ctx) error {
	var siteConfig ovirt.Site
	err := c.BodyParser(&siteConfig)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(err.Error())
	}
	if err := Validate.Struct(&siteConfig); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ValidatorError(err.(validator.ValidationErrors)))
	}
	name := c.Params("name")
	var caFile string
	if siteConfig.Ca != "" {
		caFile = "/tmp/xrm-ovirt" + name + ".ca"
		if err := os.WriteFile(caFile, []byte(siteConfig.Ca), 0644); err != nil {
			return c.Status(http.StatusInternalServerError).JSON(Status{Message: err.Error()})
		}
	}

	// path := path.Join(ovirtStoreDir, c.Params("name"), "disaster_recovery_vars.yml")
	g := ovirt.GenerateVars{
		Url:      siteConfig.Url,
		Username: siteConfig.Username,
		Password: siteConfig.Password,
		CaFile:   caFile,
		Insecure: siteConfig.Insecure,
	}
	if err = g.Generate(name, "", ""); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(Status{Message: err.Error()})
	}
	return c.Status(http.StatusOK).JSON(Status{Message: "success"})
}
