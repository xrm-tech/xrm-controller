package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/xrm-tech/xrm-controller/ovirt"
)

func DoGenerate(request string, siteConfig *ovirt.Site, username, password string, wantStatus int, wantInResp string) error {
	var (
		r           io.Reader
		contentType string
	)
	if siteConfig != nil {
		if body, err := json.Marshal(siteConfig); err == nil {
			r = bytes.NewBuffer(body)
			contentType = "application/json"
		} else {
			return err
		}
	}
	req, _ := http.NewRequest("POST", request, r)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.SetBasicAuth(username, password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("/ovirt/generate/test error = %v", err)
	}
	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus || err != nil {
		return fmt.Errorf("/ovirt/generate/test = %d (%s), error is %v", resp.StatusCode, string(body), err)
	}
	if wantInResp != "" {
		s := string(body)
		if !strings.Contains(s, wantInResp) {
			return fmt.Errorf("/ovirt/generate/test = %q, want contain %q", s, wantInResp)
		}
	}
	return nil
}
