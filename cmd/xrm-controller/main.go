package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/msaf1980/go-clipper"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

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

	rootCmd.AddString("dir", "d", "/var/lib/xrm-controller", &xrmcontroller.Cfg.StoreDir, "dir").
		AttachEnv("XRM_CONTROLLER_DIR")
	rootCmd.AddString("listen", "l", ":8080", &xrmcontroller.Cfg.Listen, "listen address").
		AttachEnv("XRM_CONTROLLER_LISTEN")
	// TODO: password protected key, get cert/key files from external storage or env vars
	rootCmd.AddString("key", "k", "", &xrmcontroller.Cfg.TLSKey, "TLS private key").
		AttachEnv("XRM_CONTROLLER_TLS_KEY")
	rootCmd.AddString("cert", "c", "", &xrmcontroller.Cfg.TLSCert, "TLS certificate").
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

	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	xrmcontroller.Cfg.Logger = zerolog.New(os.Stdout)
	xrmcontroller.Cfg.Users = make(map[string]string)
	for _, user := range users {
		username, password, _ := strings.Cut(user, ":")
		if username != "" && password != "" {
			// skip empty
			xrmcontroller.Cfg.Users[username] = password
		}
	}

	if xrmcontroller.Cfg.StoreDir == "" {
		log.Fatal().Msg("store dir can not be empty")
	}
	if !utils.DirExists(xrmcontroller.Cfg.StoreDir) {
		log.Fatal().Str("store_dir", xrmcontroller.Cfg.StoreDir).Msg("store dir not exist")
	}

	xrmcontroller.Cfg.OVirtStoreDir = path.Join(xrmcontroller.Cfg.StoreDir, "ovirt")

	app := xrmcontroller.RouterInit()
	// TODO: implement ssl, basic auth and ip acl
	if xrmcontroller.Cfg.TLSCert != "" && xrmcontroller.Cfg.TLSKey == "" {
		log.Fatal().Err(app.ListenTLS(xrmcontroller.Cfg.Listen, xrmcontroller.Cfg.TLSCert, xrmcontroller.Cfg.TLSKey))
	} else if xrmcontroller.Cfg.TLSCert == "" && xrmcontroller.Cfg.TLSKey == "" {
		log.Fatal().Err(app.Listen(xrmcontroller.Cfg.Listen))
	} else {
		log.Fatal().Str("cert", xrmcontroller.Cfg.TLSCert).Str("key", xrmcontroller.Cfg.TLSKey).Msg("TLS require set key and cert")
	}
}
