package helm

import (
	"fmt"
	"os"

	"github.com/nexa/pkg/ctx"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/arttor/helmify/pkg/app"
	"github.com/arttor/helmify/pkg/config"
)

type Helmify struct {
	ctx    *ctx.Ctx
	logger *zap.Logger

	cfg config.Config // helmify config
}

func (h *Helmify) ParseFlags(cmd *cobra.Command) {
	cmd.Flags().StringSliceVarP(&h.cfg.Files, "file", "f", []string{}, "File or directory containing k8s manifests.")
	cmd.Flags().BoolVarP(&h.cfg.Verbose, "verbose", "v", false, "Enable verbose output (print WARN & INFO).")
	cmd.Flags().BoolVarP(&h.cfg.VeryVerbose, "very-verbose", "b", false, "Enable very verbose output. Same as verbose but with DEBUG.")
	cmd.Flags().BoolVar(&h.cfg.Crd, "crd-dir", false, "Enable CRD install into 'crds' directory. (cannot be used with 'optional-crds')\nWarning: CRDs placed in 'crds' directory will not be templated by Helm.")
	cmd.Flags().BoolVar(&h.cfg.ImagePullSecrets, "image-pull-secrets", false, "Allows the user to use existing secrets as imagePullSecrets in values.yaml.")
	cmd.Flags().BoolVar(&h.cfg.GenerateDefaults, "generate-defaults", false, "Allows the user to add empty placeholders for typical customization options in values.yaml. Currently covers: topology constraints, node selectors, tolerations.")
	cmd.Flags().BoolVar(&h.cfg.CertManagerAsSubchart, "cert-manager-as-subchart", false, "Allows the user to add cert-manager as a subchart.")
	cmd.Flags().StringVar(&h.cfg.CertManagerVersion, "cert-manager-version", "v1.12.2", "Allows the user to specify cert-manager subchart version. Only useful with --cert-manager-as-subchart.")
	cmd.Flags().BoolVar(&h.cfg.CertManagerInstallCRD, "cert-manager-install-crd", true, "Allows the user to install cert-manager CRD. Only useful with --cert-manager-as-subchart.")
	cmd.Flags().BoolVarP(&h.cfg.FilesRecursively, "recursive", "r", false, "Scan dirs from -f option recursively.")
	cmd.Flags().BoolVar(&h.cfg.OriginalName, "original-name", false, "Use the object's original name instead of adding the chart's release name as the common prefix.")
	cmd.Flags().BoolVar(&h.cfg.PreserveNs, "preserve-ns", false, "Use the object's original namespace instead of adding all the resources to a common namespace.")
	cmd.Flags().BoolVar(&h.cfg.AddWebhookOption, "add-webhook-option", false, "Allows the user to add webhook option in values.yaml.")
	cmd.Flags().BoolVar(&h.cfg.OptionalCRDs, "optional-crds", false, "Enable optional CRD installation through values. (cannot be used with 'crd-dir')")
}

func (h *Helmify) Completion(cmd *cobra.Command) {
	// 为 flags 添加补全功能
	cmd.RegisterFlagCompletionFunc("file", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveFilterFileExt
	})

	cmd.RegisterFlagCompletionFunc("recursive", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("crd-dir", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("image-pull-secrets", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("generate-defaults", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("cert-manager-as-subchart", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("cert-manager-version", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// 提供一些常用的 cert-manager 版本
		versions := []string{
			"v1.12.2",
			"v1.12.1",
			"v1.12.0",
			"v1.11.2",
			"v1.11.1",
			"v1.11.0",
			"v1.10.2",
			"v1.10.1",
			"v1.10.0",
		}
		return versions, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("cert-manager-install-crd", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("original-name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("preserve-ns", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("add-webhook-option", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("optional-crds", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("verbose", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})

	cmd.RegisterFlagCompletionFunc("very-verbose", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"true", "false"}, cobra.ShellCompDirectiveNoFileComp
	})
}

func NewHelmify(ctx *ctx.Ctx) *Helmify {
	return &Helmify{
		ctx:    ctx,
		logger: ctx.Logger(),
	}
}

func (h *Helmify) GetConfig() *config.Config {
	return &h.cfg
}

func (h *Helmify) Start() error {

	if h.cfg.Crd && h.cfg.OptionalCRDs {
		return fmt.Errorf("-crd-dir and -optional-crds cannot be used together")
	}

	stat, err := os.Stdin.Stat()
	if err != nil {
		h.logger.Error("stdin error", zap.Error(err))
		return err
	}

	if len(h.cfg.Files) == 0 && (stat.Mode()&os.ModeCharDevice) != 0 {
		return fmt.Errorf("no data piped in stdin and no input files provided")
	}

	if err = app.Start(os.Stdin, h.cfg); err != nil {
		return err
	}

	return nil
}
