package node

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"

	"github.com/nexa/pkg/ctx"
	nodecollector "github.com/nexa/pkg/node/collector"
	"github.com/nexa/pkg/node/render"
	"github.com/spf13/cobra"
)

type nodeRenderFlags struct {
	samples bool
	limit   int
	human   bool
}

func Cmd(cctx *ctx.Ctx) []*cobra.Command {
	reg := nodecollector.NewDefaultRegistry(cctx)

	var rf nodeRenderFlags
	var collectOnly []string
	var exclude []string

	cmd := &cobra.Command{
		Use:          "node",
		Short:        "node metrics collectors (node_exporter-like)",
		Long:         "Collect node (machine) metrics with pluggable collectors and render as tables.",
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolVar(&rf.samples, "samples", false, "print per-sample time series rows")
	cmd.PersistentFlags().IntVar(&rf.limit, "limit", 2000, "max output rows in --samples mode (protects console)")
	cmd.PersistentFlags().BoolVar(&rf.human, "human-readable", true, "human readable output (bytes, seconds, big integers)")
	cmd.PersistentFlags().StringArrayVar(&collectOnly, "collect", nil, "collect only these collectors (repeatable; mutual exclusive with --exclude)")
	cmd.PersistentFlags().StringArrayVar(&exclude, "exclude", nil, "exclude these collectors (repeatable; mutual exclusive with --collect)")

	cmd.AddCommand(listCmd(reg))
	cmd.AddCommand(allCmd(reg, &rf, &collectOnly, &exclude))
	cmd.AddCommand(collectorCmd(reg, &rf))

	return []*cobra.Command{cmd}
}

func listCmd(reg *nodecollector.Registry) *cobra.Command {
	return &cobra.Command{
		Use:          "list",
		Short:        "list supported collectors",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			names := reg.Names()
			sort.Strings(names)
			rows := make([]nodecollector.CollectorStatus, 0, len(names))
			for _, name := range names {
				rows = append(rows, reg.Status(name))
			}
			return render.PrintCollectorList(os.Stdout, rows)
		},
	}
}

func allCmd(reg *nodecollector.Registry, rf *nodeRenderFlags, collectOnly *[]string, exclude *[]string) *cobra.Command {
	return &cobra.Command{
		Use:          "all",
		Short:        "run default collectors (implemented only)",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "linux" {
				return fmt.Errorf("nexa node collectors are currently implemented for linux; current GOOS=%s", runtime.GOOS)
			}

			if len(*collectOnly) > 0 && len(*exclude) > 0 {
				return fmt.Errorf("combined --collect and --exclude are not allowed")
			}

			var families []nodecollector.MetricFamily
			var notImplemented []string
			var errs []string

			selected := reg.DefaultCollectorsLinuxEnabledByDefault()
			if len(*collectOnly) > 0 {
				selected = append([]string(nil), (*collectOnly)...)
			} else if len(*exclude) > 0 {
				exset := map[string]struct{}{}
				for _, n := range *exclude {
					exset[n] = struct{}{}
				}
				tmp := make([]string, 0, len(selected))
				for _, n := range selected {
					if _, ok := exset[n]; ok {
						continue
					}
					tmp = append(tmp, n)
				}
				selected = tmp
			}

			for _, name := range selected {
				st := reg.Status(name)
				if !st.Implemented {
					notImplemented = append(notImplemented, name)
					continue
				}
				f, err := reg.Collect(name)
				if err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", name, err))
					continue
				}
				families = append(families, f...)
			}

			sort.Slice(families, func(i, j int) bool { return families[i].Name < families[j].Name })

			if err := render.PrintMetricFamilies(os.Stdout, families, render.Options{ShowSamples: rf.samples, Limit: rf.limit, Humanize: rf.human}); err != nil {
				return err
			}

			if len(notImplemented) > 0 {
				sort.Strings(notImplemented)
				fmt.Fprintln(os.Stdout)
				fmt.Fprintf(os.Stdout, "Not implemented collectors: %s\n", strings.Join(notImplemented, ", "))
			}
			if len(errs) > 0 {
				fmt.Fprintln(os.Stdout)
				fmt.Fprintln(os.Stdout, "Errors:")
				for _, e := range errs {
					fmt.Fprintf(os.Stdout, "- %s\n", e)
				}
			}

			return nil
		},
	}
}

func collectorCmd(reg *nodecollector.Registry, rf *nodeRenderFlags) *cobra.Command {
	return &cobra.Command{
		Use:          "<collector>",
		Short:        "run one collector",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			names := reg.Names()
			sort.Strings(names)
			out := make([]string, 0, len(names))
			for _, n := range names {
				if strings.HasPrefix(n, toComplete) {
					out = append(out, n)
				}
			}
			return out, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "linux" {
				return fmt.Errorf("nexa node collectors are currently implemented for linux; current GOOS=%s", runtime.GOOS)
			}

			name := args[0]
			if !reg.Has(name) {
				return fmt.Errorf("unknown collector: %s (try: nexa node list)", name)
			}
			families, err := reg.Collect(name)
			if err != nil {
				return err
			}
			sort.Slice(families, func(i, j int) bool { return families[i].Name < families[j].Name })
			return render.PrintMetricFamilies(os.Stdout, families, render.Options{ShowSamples: rf.samples, Limit: rf.limit, Humanize: rf.human})
		},
	}
}

