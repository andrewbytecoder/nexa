package helmify

import (
	"path/filepath"

	"github.com/nexa/pkg/ctx"
	"github.com/nexa/pkg/helm"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

func GetCmd(ctx *ctx.Ctx) []*cobra.Command {
	var cmds []*cobra.Command
	cmds = append(cmds, newCmd(ctx))

	return cmds
}

// newCmdTcpTerm returns a cobra command for fetching versions
func newCmd(ctx *ctx.Ctx) *cobra.Command {
	h := helm.NewHelmify(ctx)

	cmd := &cobra.Command{
		Use:   "helmify [flags] CHART_NAME",
		Short: "Convert Kubernetes manifests to Helm charts",
		Long:  `nexa psutil [command].`,
		Example: `Helmify parses kubernetes resources from std.in and converts it to a Helm chart.

Examples:
  # Example 1: Convert kustomize output to Helm chart
  kustomize build <kustomize_dir> | helmify mychart

  # Example 2: Convert YAML file to Helm chart
  cat my-app.yaml | helmify mychart

  # Example 3: Scan directory for k8s manifests
  helmify -f ./test_data/dir mychart

  # Example 4: Scan directory recursively
  helmify -f ./test_data/dir -r mychart

  # Example 5: Multiple files and directories
  helmify -f ./test_data/dir -f ./test_data/sample-app.yaml -f ./test_data/dir/another_dir mychart

  # Example 6: All YAML files in a directory
  awk 'FNR==1 && NR!=1  {print "---"}{print}' /my_directory/*.yaml | helmify mychart

Note: CHART_NAME is optional. Default is 'chart'. Can be a directory, e.g., 'deploy/charts/mychart'.`,
		// stop printing usage when the command errors
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			// 默认第一个参数作为chart名称
			if len(args) > 0 {
				h.GetConfig().ChartName = filepath.Base(args[0])
				h.GetConfig().ChartDir = filepath.Dir(args[0])
			}

			err := h.Start()
			if err != nil {
				ctx.Logger().Error("helmify error", zap.Error(err))
				return
			}

		},

		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			// 如果没有提供参数，提示可以输入目录或 chart 名称
			if len(args) == 0 {
				return nil, cobra.ShellCompDirectiveFilterDirs
			}
			// 已经提供了 chart 名称，不再提示
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
	}

	h.ParseFlags(cmd)
	h.Completion(cmd)

	return cmd
}
