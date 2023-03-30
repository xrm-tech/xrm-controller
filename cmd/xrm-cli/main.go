package main

import (
	"fmt"
	"os"
	"path"

	"github.com/gofiber/fiber/v2"
	"github.com/msaf1980/go-clipper"

	"github.com/xrm-tech/xrm-controller/ovirt"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

var (
	BuildVersion string

	app *fiber.App

	// config vars
	storeDir string
	name     string
	typ      xrmType
	verbose  bool

	// generate
	sitesConfig ovirt.GenerateVars
)

func main() {
	// create a new command registry
	registry := clipper.NewRegistry("xrm-cli")

	// cleanup command
	cleanupCmd, _ := registry.Register("cleanup", "cleanup")
	cleanupCmd.AddString("dir", "d", "/var/lib/xrm-controller", &storeDir, "dir")
	cleanupCmd.AddString("name", "n", "", &name, "name")
	cleanupCmd.AddValue("type", "t", newXrmType(xrmOVirt, &typ), false, "type")
	cleanupCmd.AddFlag("verbose", "v", &verbose, "debug logging")

	// generate command
	generateCmd, _ := registry.Register("generate", "generate")
	generateCmd.AddString("dir", "d", "/var/lib/xrm-controller", &storeDir, "dir")
	generateCmd.AddString("name", "n", "", &name, "name")
	generateCmd.AddValue("type", "t", newXrmType(xrmOVirt, &typ), false, "type")
	generateCmd.AddFlag("verbose", "v", &verbose, "debug logging")

	generateCmd.AddString("p_url", "", "", &sitesConfig.PrimaryUrl, "primary site url").AttachEnv("XRM_PRIMARY_URL")
	generateCmd.AddString("p_username", "", "", &sitesConfig.PrimaryUsername, "primary site username").AttachEnv("XRM_PRIMARY_USERNAME")
	generateCmd.AddString("p_password", "", "", &sitesConfig.PrimaryPassword, "primary site username").AttachEnv("XRM_PRIMARY_PASSWORD")

	generateCmd.AddString("s_url", "", "", &sitesConfig.SecondaryUrl, "secondary site url").AttachEnv("XRM_SECONDARY_URL")
	generateCmd.AddString("s_username", "", "", &sitesConfig.SecondaryUsername, "secondary site username").AttachEnv("XRM_SECONDARY_USERNAME")
	generateCmd.AddString("s_password", "", "", &sitesConfig.SecondaryPassword, "secondary site username").AttachEnv("XRM_SECONDARY_PASSWORD")

	// failover command
	failoverCmd, _ := registry.Register("failover", "failover")
	failoverCmd.AddString("dir", "d", "/var/lib/xrm-controller", &storeDir, "dir")
	failoverCmd.AddString("name", "n", "", &name, "name")
	failoverCmd.AddValue("type", "t", newXrmType(xrmOVirt, &typ), false, "type")
	failoverCmd.AddFlag("verbose", "v", &verbose, "debug logging")

	// failback command
	failbackCmd, _ := registry.Register("failover", "failover")
	failbackCmd.AddString("dir", "d", "/var/lib/xrm-controller", &storeDir, "dir")
	failbackCmd.AddString("name", "n", "", &name, "name")
	failbackCmd.AddValue("type", "t", newXrmType(xrmOVirt, &typ), false, "type")
	failbackCmd.AddFlag("verbose", "v", &verbose, "debug logging")

	commandName, err := registry.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	// logger := zerolog.New(os.Stdout)
	// if verbose {
	// 	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// }

	if storeDir == "" {
		fmt.Fprintln(os.Stderr, "store dir can not be empty")
		os.Exit(1)
	}
	if !path.IsAbs(storeDir) {
		pwd, _ := os.Getwd()
		storeDir = path.Join(pwd, storeDir)
	}
	if !utils.DirExists(storeDir) {
		fmt.Fprintf(os.Stderr, "store dir not exist: %q\n", storeDir)
		os.Exit(1)
	}

	switch typ {
	case xrmOVirt:
		oVirtStoreDir := path.Join(storeDir, "ovirt")
		switch commandName {
		case "cleanup":
			if err = ovirt.Cleanup(oVirtStoreDir, name); err == nil {
				fmt.Fprintln(os.Stderr, "success")
			} else {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		case "generate":
			var out string
			fmt.Printf("Primary URL %q Username %q\n", sitesConfig.PrimaryUrl, sitesConfig.PrimaryUsername)
			fmt.Printf("Secondary URL %q Username %q\n", sitesConfig.SecondaryUrl, sitesConfig.SecondaryUsername)

			if err = sitesConfig.Validate(); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			} else if out, err = sitesConfig.Generate(name, oVirtStoreDir); err == nil {
				if out != "" {
					fmt.Fprintln(os.Stdout, out)
				}
				fmt.Fprintln(os.Stdout, "success")
			} else {
				if out != "" {
					fmt.Fprintln(os.Stderr, out)
				}
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		case "failover":
			var out string
			if out, err = ovirt.Failover(oVirtStoreDir, name); err == nil {
				if out != "" {
					fmt.Fprintln(os.Stdout, out)
				}
				fmt.Fprintln(os.Stdout, "success")
			} else {
				if out != "" {
					fmt.Fprintln(os.Stderr, out)
				}
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		case "failback":
			var out string
			if out, err = ovirt.Failback(oVirtStoreDir, name); err == nil {
				if out != "" {
					fmt.Fprintln(os.Stdout, out)
				}
				fmt.Fprintln(os.Stdout, "success")
			} else {
				if out != "" {
					fmt.Fprintln(os.Stderr, out)
				}
				fmt.Fprintln(os.Stderr, err.Error())
				os.Exit(1)
			}
		}
	default:
		fmt.Fprintf(os.Stderr, "type %q not supported\n", typ.String())
	}
}
