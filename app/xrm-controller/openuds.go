package xrmcontroller

import (
	"net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/rs/zerolog"
	openuds "github.com/xrm-tech/xrm-controller/openuds"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

func openUDSDelete(c *fiber.Ctx) error {
	if err := openuds.Delete(c.Params("name"), Cfg.OpenUDSStoreDir); err != nil {
		return fiber.NewError(http.StatusInternalServerError, err.Error())
	}
	return c.Status(http.StatusOK).SendString("success")
}

func openUDSGenerate(c *fiber.Ctx) (err error) {
	var (
		sitesConfig openuds.GenerateVars
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

	if Cfg.Logger.GetLevel() == zerolog.DebugLevel || Cfg.Logger.GetLevel() == zerolog.TraceLevel {
		c.Context().SetUserValue("req_body", utils.UnsafeString(bodyPasswordCleanup(c.Request().Body())))
	}

	if out, err = openuds.Generate(name, Cfg.OpenUDSStoreDir, sitesConfig); err == nil {
		return c.Status(http.StatusOK).SendString(out)
	} else {
		return fiber.NewError(http.StatusInternalServerError, err.Error()+"\n"+out)
	}
}

func openUDSFailover(c *fiber.Ctx) (err error) {
	var (
		out string
	)
	name := c.Params("name")

	// TODO (SECURITY): cleanup token from out
	if out, err = openuds.Failover(name, Cfg.OVirtStoreDir); err == nil {
		// TODO: debug loglevel ??
		return c.Status(http.StatusOK).SendString(out)
	} else {
		return fiber.NewError(http.StatusInternalServerError, err.Error()+"\n"+out)
	}
}
