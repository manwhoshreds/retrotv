package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleDisksInfoReturnsServiceUnavailableWhenSmartctlIsMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	request := httptest.NewRequest(http.MethodGet, "/api/system/disks", nil)
	response := httptest.NewRecorder()

	(&Server{}).handleDisksInfo(response, request)

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "smartctl is unavailable") {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestHandleDisksInfoAcceptsSmartctlJSONWithNonzeroExitStatus(t *testing.T) {
	dir := t.TempDir()
	command := filepath.Join(dir, "smartctl")
	script := `#!/bin/sh
printf '%s' '{"smartctl":{"exit_status":8},"device":{"name":"/dev/test"},"model_name":"Test Drive","smart_status":{"passed":true},"temperature":{"current":31},"power_on_time":{"hours":42}}'
exit 8
`
	if err := os.WriteFile(command, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	request := httptest.NewRequest(http.MethodGet, "/api/system/disks?device=/dev/test", nil)
	response := httptest.NewRecorder()

	(&Server{}).handleDisksInfo(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"overall_smart_status":"PASSED"`) || !strings.Contains(response.Body.String(), `"model":"Test Drive"`) {
		t.Fatalf("body = %s", response.Body.String())
	}
}

func TestHandleDisksInfoRejectsDeviceOutsideDev(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/system/disks?device=/etc/passwd", nil)
	response := httptest.NewRecorder()

	(&Server{}).handleDisksInfo(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}
