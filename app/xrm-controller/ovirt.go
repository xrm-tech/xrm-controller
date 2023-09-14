package xrmcontroller

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
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
		c.Context().SetUserValue("req_body", utils.UnsafeString(bodyPasswordCleanup(c.Request().Body())))
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}

	if err := sitesConfig.Validate(); err != nil {
		c.Context().SetUserValue("req_body", utils.UnsafeString(bodyPasswordCleanup(c.Request().Body())))
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	name := c.Params("name")

	ovirt.StripStorageDomains(sitesConfig.StorageDomains)

	if Cfg.Logger.GetLevel() == zerolog.DebugLevel || Cfg.Logger.GetLevel() == zerolog.TraceLevel {
		c.Context().SetUserValue("req_body", utils.UnsafeString(bodyPasswordCleanup(c.Request().Body())))
	}

	_, out, err = sitesConfig.Generate(name, Cfg.OVirtStoreDir)

	if err == nil {
		return c.Status(http.StatusOK).SendString(out)
	} else {
		return fiber.NewError(http.StatusInternalServerError, err.Error()+"\n"+out)
	}
}

func oVirtFailover(c *fiber.Ctx) (err error) {
	var (
		out string
	)
	name := c.Params("name")

	// TODO (SECURITY): cleanup token from out
	if out, err = ovirt.Failover(name, Cfg.OVirtStoreDir); err == nil {
		// TODO: debug loglevel ??
		return c.Status(http.StatusOK).SendString(out)
	} else {
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
		return c.Status(http.StatusOK).SendString(out)
	} else {
		return fiber.NewError(http.StatusInternalServerError, err.Error()+"\n"+out)
	}
}
