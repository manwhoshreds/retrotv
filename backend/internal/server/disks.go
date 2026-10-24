package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
)

type Smartctl struct {
	Smartctl struct {
		ExitStatus int `json:"exit_status"`
		Messages   []struct {
			String   string `json:"string"`
			Severity string `json:"severity"`
		} `json:"messages"`
	} `json:"smartctl"`

	Device struct {
		Name string `json:"name"`
	} `json:"device"`

	ModelName       string `json:"model_name"`
	SerialNumber    string `json:"serial_number"`
	FirmwareVersion string `json:"firmware_version"`

	SmartStatus *struct {
		Passed bool `json:"passed"`
	} `json:"smart_status"`

	Temperature struct {
		Current int `json:"current"`
	} `json:"temperature"`

	PowerOnTime struct {
		Hours int `json:"hours"`
	} `json:"power_on_time"`

	AtaSmartAttributes struct {
		Table []struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Value  int    `json:"value"`
			Worst  int    `json:"worst"`
			Thresh int    `json:"thresh"`
			Raw    struct {
				Value int64 `json:"value"`
			} `json:"raw"`
		} `json:"table"`
	} `json:"ata_smart_attributes"`
}

type Attribute struct {
	Name      string `json:"name"`
	Value     int64  `json:"value"`
	Threshold string `json:"status"`
}

type Report struct {
	Device           string      `json:"device"`
	Model            string      `json:"model"`
	SerialNumber     string      `json:"serial_number"`
	Firmware         string      `json:"firmware"`
	OverallSMART     string      `json:"overall_smart_status"`
	Health           string      `json:"health"`
	TemperatureC     int         `json:"temperature_c"`
	PowerOnHours     int         `json:"power_on_hours"`
	ImportantMetrics []Attribute `json:"important_metrics"`
	Issues           []string    `json:"issues"`
}

func findAttribute(attrs []struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Value  int    `json:"value"`
	Worst  int    `json:"worst"`
	Thresh int    `json:"thresh"`
	Raw    struct {
		Value int64 `json:"value"`
	} `json:"raw"`
}, name string) (int64, bool) {

	for _, a := range attrs {
		if a.Name == name {
			return a.Raw.Value, true
		}
	}
	return 0, false
}

func (s *Server) handleDisksInfo(w http.ResponseWriter, r *http.Request) {
	device, err := smartDevicePath(r.URL.Query().Get("device"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	cmd := exec.CommandContext(r.Context(), "smartctl", "-a", "-j", device)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	if err != nil && stdout.Len() == 0 {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			message := strings.TrimSpace(stderr.String())
			if message == "" {
				message = exitErr.Error()
			}
			writeError(w, http.StatusBadGateway, fmt.Sprintf("smartctl could not inspect %s: %s", device, message))
			return
		}
		writeError(w, http.StatusServiceUnavailable, fmt.Sprintf("smartctl is unavailable: %v", err))
		return
	}

	var smart Smartctl
	if err := json.Unmarshal(stdout.Bytes(), &smart); err != nil {
		writeError(w, http.StatusBadGateway, fmt.Sprintf("smartctl returned invalid JSON for %s: %v", device, err))
		return
	}

	report := Report{
		Device:       device,
		Model:        smart.ModelName,
		SerialNumber: smart.SerialNumber,
		Firmware:     smart.FirmwareVersion,
		TemperatureC: smart.Temperature.Current,
		PowerOnHours: smart.PowerOnTime.Hours,
		Health:       "Unknown",
		OverallSMART: "UNKNOWN",
		Issues:       []string{},
	}
	if smart.Device.Name != "" {
		report.Device = smart.Device.Name
	}

	if smart.SmartStatus != nil {
		if smart.SmartStatus.Passed {
			report.OverallSMART = "PASSED"
			report.Health = "Healthy"
		} else {
			report.OverallSMART = "FAILED"
			report.Health = "Critical"
			report.Issues = append(report.Issues, "SMART overall status failed")
		}
	}
	for _, message := range smart.Smartctl.Messages {
		if strings.EqualFold(message.Severity, "error") && strings.TrimSpace(message.String) != "" {
			report.Issues = append(report.Issues, strings.TrimSpace(message.String))
		}
	}

	check := func(name string) {
		if value, ok := findAttribute(smart.AtaSmartAttributes.Table, name); ok {

			status := "OK"

			switch name {

			case "Reallocated_Sector_Ct":
				if value > 0 {
					status = "WARNING"
					promoteDiskHealth(&report, "Warning")
					report.Issues = append(report.Issues,
						fmt.Sprintf("%d reallocated sectors", value))
				}

			case "Current_Pending_Sector":
				if value > 0 {
					status = "CRITICAL"
					promoteDiskHealth(&report, "Critical")
					report.Issues = append(report.Issues,
						fmt.Sprintf("%d pending sectors", value))
				}

			case "Offline_Uncorrectable":
				if value > 0 {
					status = "CRITICAL"
					promoteDiskHealth(&report, "Critical")
					report.Issues = append(report.Issues,
						fmt.Sprintf("%d offline uncorrectable sectors", value))
				}

			case "Reported_Uncorrect":
				if value > 0 {
					status = "WARNING"
					promoteDiskHealth(&report, "Warning")
					report.Issues = append(report.Issues,
						fmt.Sprintf("%d reported uncorrectable errors", value))
				}

			case "UDMA_CRC_Error_Count":
				if value > 0 {
					status = "WARNING"
					promoteDiskHealth(&report, "Warning")
					report.Issues = append(report.Issues,
						fmt.Sprintf("%d CRC errors (possible SATA cable issue)", value))
				}
			}

			report.ImportantMetrics = append(report.ImportantMetrics, Attribute{
				Name:      name,
				Value:     value,
				Threshold: status,
			})
		}
	}

	check("Reallocated_Sector_Ct")
	check("Current_Pending_Sector")
	check("Offline_Uncorrectable")
	check("Reported_Uncorrect")
	check("UDMA_CRC_Error_Count")

	if report.PowerOnHours > 40000 && report.Health == "Healthy" {
		report.Health = "Warning"
		report.Issues = append(report.Issues,
			fmt.Sprintf("Drive has %d power-on hours", report.PowerOnHours))
	}

	writeJSON(w, http.StatusOK, report)
}

func promoteDiskHealth(report *Report, health string) {
	severity := map[string]int{"Unknown": 0, "Healthy": 1, "Warning": 2, "Critical": 3}
	if severity[health] > severity[report.Health] {
		report.Health = health
	}
}

func smartDevicePath(value string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "/dev/sda", nil
	}
	cleaned := filepath.Clean(strings.TrimSpace(value))
	if !strings.HasPrefix(cleaned, "/dev/") || cleaned == "/dev" || strings.ContainsAny(cleaned, "\x00\r\n") {
		return "", fmt.Errorf("device must be a path below /dev")
	}
	return cleaned, nil
}
