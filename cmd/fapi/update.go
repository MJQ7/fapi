package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"fapi/internal/config"
	"fapi/internal/updates"
)

// applyUpdateTimeout limits how long installing an update may take.
const applyUpdateTimeout = 9 * time.Minute

// applyUpdate runs `fapi apply-update [--data-dir <folder>]`: the .deb and
// .rpm's root helper, which installs an update fapi downloaded. The
// fapi-update service runs it, as root, when fapi asks for an update (see
// packaging/fapi-update.path and internal/updates/apply.go).
func applyUpdate(arguments []string) error {
	flags := flag.NewFlagSet("apply-update", flag.ContinueOnError)
	dataDirFlag := flags.String("data-dir", "", "fapi's data folder, holding the downloaded update")
	err := flags.Parse(arguments)
	if err != nil {
		return err // flag has already printed what was wrong
	}
	if flags.NArg() > 0 {
		return fmt.Errorf("apply-update: unexpected argument %q", flags.Arg(0))
	}
	if runtime.GOOS != "linux" || os.Geteuid() != 0 {
		return errors.New("apply-update installs .deb and .rpm updates, so it only runs as root on Linux")
	}

	dataDir, err := config.DataDir(*dataDirFlag)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), applyUpdateTimeout)
	defer cancel()
	return updates.ApplyRequest(ctx, dataDir, version, packageName())
}
