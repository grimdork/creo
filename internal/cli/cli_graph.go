package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/grimdork/climate/arg"
	"github.com/grimdork/climate/fx"
	"github.com/grimdork/creo/internal/fiat"
	"github.com/grimdork/creo/internal/runner"
	"github.com/grimdork/creo/internal/targets"
)

// RunGraph renders the dependency graph in the requested format (tree/dot/svg).
func RunGraph(opt *arg.Options) error {
	format := opt.GetString("graph")
	if !runner.ValidGraphFormat(format) {
		return fmt.Errorf("--graph must be 'tree', 'dot', or 'svg'")
	}

	fiatPath, ok := fiat.FindFiat(opt.GetString("file"))
	if !ok {
		return fmt.Errorf("no fiat file found or file inaccessible")
	}
	dir := filepath.Dir(fiatPath)

	file, err := fiat.Parse(fiatPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", fiatPath, err)
	}
	if bd := opt.GetString("output"); bd != "" {
		InjectBuildDir(file, bd)
	}
	if err := targets.Apply(file); err != nil {
		return fmt.Errorf("applying defaults to %s: %w", fiatPath, err)
	}

	out, err := runner.RenderGraph(file, dir, format, opt.GetBool("status"))
	if err != nil {
		return err
	}
	fx.Fprint(os.Stdout, "{}", out)
	return nil
}
