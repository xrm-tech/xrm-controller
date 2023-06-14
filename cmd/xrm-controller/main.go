package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/msaf1980/go-clipper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	xrm "github.com/xrm-tech/xrm-controller/app/xrm-controller"
	xrmcontroller "github.com/xrm-tech/xrm-controller/app/xrm-controller"
	"github.com/xrm-tech/xrm-controller/pkg/utils"
)

var (
	BuildVersion string
	users        []string
	debug        bool
)

func main() {
	// create a new command registry
	registry := clipper.NewRegistry("xrm-controller")
	rootCmd, _ := registry.Register("", "") // root command

	rootCmd.AddString("dir", "d", "/var/lib/xrm-controller", &xrm.Cfg.StoreDir, "dir").
		AttachEnv("XRM_CONTROLLER_DIR")
	rootCmd.AddString("listen", "l", ":8080", &xrm.Cfg.Listen, "listen address").
		AttachEnv("XRM_CONTROLLER_LISTEN")
	// TODO: password protected key, get cert/key files from external storage or env vars
	rootCmd.AddString("key", "k", "", &xrm.Cfg.TLSKey, "TLS private key").
		AttachEnv("XRM_CONTROLLER_TLS_KEY")
	rootCmd.AddString("cert", "c", "", &xrm.Cfg.TLSCert, "TLS certificate").
		AttachEnv("XRM_CONTROLLER_TLS_CERT")
	// TODO: may be API key for simplify or crypted passwords for security
	// no default password, it's security hole
	rootCmd.AddStringArray("user", "u", []string{}, &users, "users (username1:password1,...)").
		AttachEnv("XRM_CONTROLLER_USERS")
	rootCmd.AddVersionHelper("version", "v", registry.Description, BuildVersion)
	rootCmd.AddFlag("debug", "", &debug, "debug logging").
		AttachEnv("XRM_CONTROLLER_DEBUG")

	if _, err := registry.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	xrm.Cfg.Logger = zerolog.New(os.Stdout)
	xrm.Cfg.Users = make(map[string]string)
	for _, user := range users {
		username, password, _ := strings.Cut(user, ":")
		if username != "" && password != "" {
			// skip empty
			xrm.Cfg.Users[username] = password
		}
	}

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if xrm.Cfg.StoreDir == "" {
		log.Fatal().Msg("store dir can not be empty")
	}
	if !utils.DirExists(xrm.Cfg.StoreDir) {
		log.Fatal().Str("store_dir", xrm.Cfg.StoreDir).Msg("store dir not exist")
	}

	xrm.Cfg.OVirtStoreDir = path.Join(xrm.Cfg.StoreDir, "ovirt")

	app := xrmcontroller.RouterInit()
	// TODO: implement ssl, basic auth and ip acl
	if xrm.Cfg.TLSCert != "" && xrm.Cfg.TLSKey == "" {
		log.Fatal().Err(app.ListenTLS(xrm.Cfg.Listen, xrm.Cfg.TLSCert, xrm.Cfg.TLSKey))
	} else if xrm.Cfg.TLSCert == "" && xrm.Cfg.TLSKey == "" {
		log.Fatal().Err(app.Listen(xrm.Cfg.Listen))
	} else {
		log.Fatal().Str("cert", xrm.Cfg.TLSCert).Str("key", xrm.Cfg.TLSKey).Msg("TLS require set key and cert")
	}
}
