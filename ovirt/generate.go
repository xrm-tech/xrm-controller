package ovirt

import (
	"errors"
	"time"

	ovirtsdk4 "github.com/ovirt/go-ovirt"
)

type GenerateVars struct {
	Url      string
	Username string
	Password string
	CaFile   string
	Insecure bool
}

func (g GenerateVars) Generate(name, varsFile, ansiblePlayFile string) (err error) {
	if err = g.validateOvirtCon(); err != nil {
		return err
	}

	// 	extra_vars = "site={0} username={1} password={2} ca={3} var_file={4}".\
	// 	format(site, username, password, ca_file, var_file)
	// command = [
	// 	"ansible-playbook", ansible_play_file,
	// 	"-t", dr_tag,
	// 	"-e", extra_vars,
	// 	"-vvvvv"
	// ]
	// log.info("Executing command %s", ' '.join(map(str, command)))
	// if log_file is not None and log_file != '':
	// 	self._log_to_file(log_file, command)
	// else:
	// 	self._log_to_console(command, log)

	// if not os.path.isfile(var_file):
	// 	log.error("Can not find output file in '%s'.", var_file)
	// 	self._print_error(log)
	// 	sys.exit()
	// log.info("Var file location: '%s'", var_file)
	// self._print_success(log)

	return errors.New("not implemented")
}

func (g GenerateVars) validateOvirtCon() error {
	builder := ovirtsdk4.NewConnectionBuilder().
		URL(g.Url).
		Username(g.Username).
		Password(g.Password).
		Insecure(g.Insecure).
		Timeout(time.Second * 10)

	if g.CaFile != "" {
		builder = builder.CAFile(g.CaFile)
	}

	if conn, err := builder.Build(); err == nil {
		defer conn.Close()

		if err = conn.Test(); err == nil {
			_, err = conn.SystemService().DataCentersService().List().Send()
		}

		return err
	} else {
		return err
	}
}
