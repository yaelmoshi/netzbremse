package collector

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	"github.com/sm-moshi/netzbremse/internal/config"
	"github.com/sm-moshi/netzbremse/internal/model"
	"github.com/sm-moshi/netzbremse/internal/postgres"
)

func Run(ctx context.Context, cfg config.Measurement) (model.Measurement, error) {
	fields := strings.Fields(cfg.Command)
	if len(fields) == 0 {
		return model.Measurement{}, fmt.Errorf("NETZBREMSE_SPEEDTEST_COMMAND is empty")
	}

	commandCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(commandCtx, fields[0], fields[1:]...)
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		if stderr.Len() > 0 {
			log.Printf("speedtest stderr: %s", strings.TrimSpace(stderr.String()))
		}
		return model.Measurement{}, fmt.Errorf("run %q: %w", cfg.Command, err)
	}

	measurement, parseErr := postgres.ParseMeasurementPayload(output, time.Now().UTC(), cfg.Endpoint)
	if parseErr == nil && !measurement.Success && stderr.Len() > 0 {
		log.Printf("speedtest reported failure; browser diagnostics:\n%s", strings.TrimSpace(stderr.String()))
	}
	return measurement, parseErr
}
