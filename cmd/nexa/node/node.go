package node

import (
	"fmt"
	"os"
	"regexp"
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

type nodeCollectorFlags struct {
	disableDefaults bool
	forceEnable     map[string]*bool
	forceDisable    map[string]*bool
}

type nodePostFilterFlags struct {
	fsMountInclude string
	fsMountExclude string
	fsTypeInclude  string
	fsTypeExclude  string

	netdevInclude string
	netdevExclude string

	diskInclude string
	diskExclude string
}

func Cmd(cctx *ctx.Ctx) []*cobra.Command {
	reg := nodecollector.NewDefaultRegistry(cctx)

	var rf nodeRenderFlags
	var collectOnly []string
	var exclude []string
	cf := nodeCollectorFlags{
		forceEnable:  map[string]*bool{},
		forceDisable: map[string]*bool{},
	}
	var pf nodePostFilterFlags

	cmd := &cobra.Command{
		Use:          "node",
		Short:        "node metrics collectors (node_exporter-like)",
		Long:         "Collect node (machine) metrics with pluggable collectors and render as tables.",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if runtime.GOOS != "linux" {
				return fmt.Errorf("nexa node collectors are currently implemented for linux; current GOOS=%s", runtime.GOOS)
			}

			enabledSet := computeEnabledSet(reg, cf)

			// `nexa node <collector>`
			if len(args) == 1 {
				name := args[0]
				if !reg.Has(name) {
					return fmt.Errorf("unknown collector: %s (try: nexa node list)", name)
				}
				if _, ok := enabledSet[name]; !ok {
					return fmt.Errorf("collector %s is disabled by flags (try enabling with --collector.%s)", name, name)
				}
				families, err := reg.Collect(name)
				if err != nil {
					return err
				}
				families, err = applyPostFilters(families, pf)
				if err != nil {
					return err
				}
				sort.Slice(families, func(i, j int) bool { return families[i].Name < families[j].Name })
				return render.PrintMetricFamilies(os.Stdout, families, render.Options{ShowSamples: rf.samples, Limit: rf.limit, Humanize: rf.human})
			}

			// `nexa node --collect ...` or `nexa node --exclude ...`
			if len(collectOnly) > 0 || len(exclude) > 0 {
				return allCmd(reg, &rf, &collectOnly, &exclude, enabledSet, pf).RunE(cmd, args)
			}

			return cmd.Help()
		},
	}

	cmd.PersistentFlags().BoolVar(&rf.samples, "samples", false, "print per-sample time series rows")
	cmd.PersistentFlags().IntVar(&rf.limit, "limit", 2000, "max output rows in --samples mode (protects console)")
	cmd.PersistentFlags().BoolVar(&rf.human, "human-readable", true, "human readable output (bytes, seconds, big integers)")
	cmd.PersistentFlags().StringArrayVar(&collectOnly, "collect", nil, "collect only these collectors (repeatable; mutual exclusive with --exclude)")
	cmd.PersistentFlags().StringArrayVar(&exclude, "exclude", nil, "exclude these collectors (repeatable; mutual exclusive with --collect)")
	cmd.PersistentFlags().BoolVar(&cf.disableDefaults, "collector.disable-defaults", false, "disable all collectors by default (enable explicitly with --collector.<name>)")

	// Subset of upstream include/exclude flags applied as post-filters on gathered metrics.
	cmd.PersistentFlags().StringVar(&pf.fsMountInclude, "collector.filesystem.mount-points-include", "", "regexp of filesystem mountpoints to include (post-filter; mutually exclusive with exclude)")
	cmd.PersistentFlags().StringVar(&pf.fsMountExclude, "collector.filesystem.mount-points-exclude", "", "regexp of filesystem mountpoints to exclude (post-filter; mutually exclusive with include)")
	cmd.PersistentFlags().StringVar(&pf.fsTypeInclude, "collector.filesystem.fs-types-include", "", "regexp of filesystem types to include (post-filter; mutually exclusive with exclude)")
	cmd.PersistentFlags().StringVar(&pf.fsTypeExclude, "collector.filesystem.fs-types-exclude", "", "regexp of filesystem types to exclude (post-filter; mutually exclusive with include)")
	cmd.PersistentFlags().StringVar(&pf.netdevInclude, "collector.netdev.device-include", "", "regexp of net devices to include (post-filter; mutually exclusive with exclude)")
	cmd.PersistentFlags().StringVar(&pf.netdevExclude, "collector.netdev.device-exclude", "", "regexp of net devices to exclude (post-filter; mutually exclusive with include)")
	cmd.PersistentFlags().StringVar(&pf.diskInclude, "collector.diskstats.device-include", "", "regexp of disk devices to include (post-filter; mutually exclusive with exclude)")
	cmd.PersistentFlags().StringVar(&pf.diskExclude, "collector.diskstats.device-exclude", "", "regexp of disk devices to exclude (post-filter; mutually exclusive with include)")

	for _, name := range reg.Names() {
		n := name
		en := false
		dis := false
		cf.forceEnable[n] = &en
		cf.forceDisable[n] = &dis
		cmd.PersistentFlags().BoolVar(cf.forceEnable[n], "collector."+n, false, "enable collector "+n)
		cmd.PersistentFlags().BoolVar(cf.forceDisable[n], "no-collector."+n, false, "disable collector "+n)
	}

	cmd.AddCommand(listCmd(reg))
	cmd.AddCommand(allCmd(reg, &rf, &collectOnly, &exclude, nil, pf))
	// NOTE: Cobra subcommand names must be literal; we keep the collector runner on root args.

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

func allCmd(reg *nodecollector.Registry, rf *nodeRenderFlags, collectOnly *[]string, exclude *[]string, enabledSet map[string]struct{}, pf nodePostFilterFlags) *cobra.Command {
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
			var notEnabled []string
			var errs []string

			selected := reg.DefaultCollectorsLinuxEnabledByDefault()
			if enabledSet != nil {
				tmp := make([]string, 0, len(selected))
				for _, n := range selected {
					if _, ok := enabledSet[n]; ok {
						tmp = append(tmp, n)
					}
				}
				selected = tmp
			}
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
				if enabledSet != nil {
					if _, ok := enabledSet[name]; !ok {
						notEnabled = append(notEnabled, name)
						continue
					}
				}
				f, err := reg.Collect(name)
				if err != nil {
					errs = append(errs, fmt.Sprintf("%s: %v", name, err))
					continue
				}
				ff, ferr := applyPostFilters(f, pf)
				if ferr != nil {
					errs = append(errs, fmt.Sprintf("%s(filter): %v", name, ferr))
					continue
				}
				families = append(families, ff...)
			}

			sort.Slice(families, func(i, j int) bool { return families[i].Name < families[j].Name })

			if err := render.PrintMetricFamilies(os.Stdout, families, render.Options{ShowSamples: rf.samples, Limit: rf.limit, Humanize: rf.human}); err != nil {
				return err
			}

			if len(notEnabled) > 0 {
				sort.Strings(notEnabled)
				fmt.Fprintln(os.Stdout)
				fmt.Fprintf(os.Stdout, "Not enabled collectors: %s\n", strings.Join(notEnabled, ", "))
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

func computeEnabledSet(reg *nodecollector.Registry, cf nodeCollectorFlags) map[string]struct{} {
	enabled := map[string]struct{}{}
	if !cf.disableDefaults {
		for _, n := range reg.DefaultCollectorsLinuxEnabledByDefault() {
			enabled[n] = struct{}{}
		}
	}
	for name, v := range cf.forceEnable {
		if v != nil && *v {
			enabled[name] = struct{}{}
		}
	}
	for name, v := range cf.forceDisable {
		if v != nil && *v {
			delete(enabled, name)
		}
	}
	return enabled
}

func applyPostFilters(families []nodecollector.MetricFamily, pf nodePostFilterFlags) ([]nodecollector.MetricFamily, error) {
	if pf.fsMountInclude != "" && pf.fsMountExclude != "" {
		return nil, fmt.Errorf("collector.filesystem.mount-points-include and mount-points-exclude are mutually exclusive")
	}
	if pf.fsTypeInclude != "" && pf.fsTypeExclude != "" {
		return nil, fmt.Errorf("collector.filesystem.fs-types-include and fs-types-exclude are mutually exclusive")
	}
	if pf.netdevInclude != "" && pf.netdevExclude != "" {
		return nil, fmt.Errorf("collector.netdev.device-include and device-exclude are mutually exclusive")
	}
	if pf.diskInclude != "" && pf.diskExclude != "" {
		return nil, fmt.Errorf("collector.diskstats.device-include and device-exclude are mutually exclusive")
	}

	var (
		fsMountInc *regexp.Regexp
		fsMountExc *regexp.Regexp
		fsTypeInc  *regexp.Regexp
		fsTypeExc  *regexp.Regexp
		netInc     *regexp.Regexp
		netExc     *regexp.Regexp
		diskInc    *regexp.Regexp
		diskExc    *regexp.Regexp
		err        error
	)
	if pf.fsMountInclude != "" {
		fsMountInc, err = regexp.Compile(pf.fsMountInclude)
		if err != nil {
			return nil, err
		}
	}
	if pf.fsMountExclude != "" {
		fsMountExc, err = regexp.Compile(pf.fsMountExclude)
		if err != nil {
			return nil, err
		}
	}
	if pf.fsTypeInclude != "" {
		fsTypeInc, err = regexp.Compile(pf.fsTypeInclude)
		if err != nil {
			return nil, err
		}
	}
	if pf.fsTypeExclude != "" {
		fsTypeExc, err = regexp.Compile(pf.fsTypeExclude)
		if err != nil {
			return nil, err
		}
	}
	if pf.netdevInclude != "" {
		netInc, err = regexp.Compile(pf.netdevInclude)
		if err != nil {
			return nil, err
		}
	}
	if pf.netdevExclude != "" {
		netExc, err = regexp.Compile(pf.netdevExclude)
		if err != nil {
			return nil, err
		}
	}
	if pf.diskInclude != "" {
		diskInc, err = regexp.Compile(pf.diskInclude)
		if err != nil {
			return nil, err
		}
	}
	if pf.diskExclude != "" {
		diskExc, err = regexp.Compile(pf.diskExclude)
		if err != nil {
			return nil, err
		}
	}

	filterSample := func(f nodecollector.MetricFamily, s nodecollector.Sample) bool {
		switch {
		case strings.HasPrefix(f.Name, "node_filesystem_"):
			mp := labelValue(s.Labels, "mountpoint")
			fs := labelValue(s.Labels, "fstype")
			if fsMountInc != nil && !fsMountInc.MatchString(mp) {
				return false
			}
			if fsMountExc != nil && fsMountExc.MatchString(mp) {
				return false
			}
			if fsTypeInc != nil && !fsTypeInc.MatchString(fs) {
				return false
			}
			if fsTypeExc != nil && fsTypeExc.MatchString(fs) {
				return false
			}
		case strings.HasPrefix(f.Name, "node_network_"):
			dev := labelValue(s.Labels, "device")
			if netInc != nil && !netInc.MatchString(dev) {
				return false
			}
			if netExc != nil && netExc.MatchString(dev) {
				return false
			}
		case strings.HasPrefix(f.Name, "node_disk_"):
			dev := labelValue(s.Labels, "device")
			if diskInc != nil && !diskInc.MatchString(dev) {
				return false
			}
			if diskExc != nil && diskExc.MatchString(dev) {
				return false
			}
		}
		return true
	}

	out := make([]nodecollector.MetricFamily, 0, len(families))
	for _, f := range families {
		nf := f
		if len(nf.Samples) > 0 {
			nf.Samples = nf.Samples[:0]
			for _, s := range f.Samples {
				if filterSample(f, s) {
					nf.Samples = append(nf.Samples, s)
				}
			}
		}
		// TODO: extend to Histograms/Summaries for relevant collectors.
		out = append(out, nf)
	}
	return out, nil
}

func labelValue(labels []nodecollector.Label, key string) string {
	for _, l := range labels {
		if l.Name == key {
			return l.Value
		}
	}
	return ""
}

