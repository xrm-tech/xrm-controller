package ovirt

import (
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type HttpError struct {
	Code int
}

func (e HttpError) Error() string {
	return "request failed with " + strconv.Itoa(e.Code)
}

func getCaUrl(ovirtUrl string) string {
	var out strings.Builder
	out.Grow(len(ovirtUrl) + 32)
	_, u, _ := strings.Cut(ovirtUrl, "://")
	out.WriteString("http://")
	addr, _, _ := strings.Cut(u, "/")
	out.WriteString(addr)
	out.WriteString("/ovirt-engine/services/pki-resource?resource=ca-certificate&format=X509-PEM-CA")

	return out.String()
}

func saveCaFile(ovirtUrl string, caFile string) error {
	caUrl := getCaUrl(ovirtUrl)
	client := http.Client{
		Timeout: time.Second * 10,
	}
	resp, err := client.Get(caUrl)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return HttpError{resp.StatusCode}
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return os.WriteFile(caFile, b, 0640)
}

var nameRe = regexp.MustCompile(`^[a-zA-Z_\-0-9]+$`)

func validateName(name string) bool {
	return name != "template" && nameRe.MatchString(name)
}
