package prometheus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nexa/pkg/ctx"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func Cmd(cctx *ctx.Ctx) []*cobra.Command {
	_ = cctx

	var (
		namespace  string
		query      string
		address    string
		service    string
		selector   string
		portName   string
		kubeconfig string
		timeout    time.Duration
		limit      int
	)

	cmd := &cobra.Command{
		Use:          "prometheus",
		Short:        "discover Prometheus in a namespace and query metrics",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	queryCmd := &cobra.Command{
		Use:          "query [promql]",
		Short:        "query Prometheus instant query API",
		Example:      "nexa prometheus query -n monitoring 'up==1'\n  nexa prometheus query -n monitoring --query 'up==1'",
		SilenceUsage: true,
		Args:         cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			q := query
			if len(args) == 1 {
				// Positional promql takes precedence for shell-friendliness.
				// If both are provided, fail fast to avoid ambiguity.
				if cmd.Flags().Changed("query") {
					return fmt.Errorf("provide query either as positional arg or via --query, not both")
				}
				q = args[0]
			}
			if strings.TrimSpace(q) == "" {
				return fmt.Errorf("empty query")
			}

			baseURL := address
			if baseURL == "" {
				cli, err := newKubeClient(kubeconfig)
				if err != nil {
					return err
				}
				host, port, err := discoverPrometheus(ctx, cli, namespace, service, selector, portName)
				if err != nil {
					return err
				}
				baseURL = fmt.Sprintf("http://%s", net.JoinHostPort(host, port))
			}

			res, err := promQuery(ctx, baseURL, q)
			if err != nil {
				return err
			}
			return renderPromResult(ctx, os.Stdout, baseURL, q, res, limit)
		},
	}

	cmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "namespace to discover Prometheus in")
	cmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig (optional; defaults to in-cluster or ~/.kube/config)")

	queryCmd.Flags().StringVar(&query, "query", "up==1", "PromQL query to execute (instant query)")
	queryCmd.Flags().StringVar(&address, "address", "", "Prometheus base URL, e.g. http://10.247.96.18:9090 (skip discovery)")
	queryCmd.Flags().StringVar(&service, "service", "", "Prometheus Service name (optional, prefer if known)")
	queryCmd.Flags().StringVar(&selector, "selector", "", "Service label selector to find Prometheus, e.g. app=prometheus")
	queryCmd.Flags().StringVar(&portName, "port-name", "", "Service port name to use (optional)")
	queryCmd.Flags().DurationVar(&timeout, "timeout", 10*time.Second, "overall timeout for discovery and query")
	queryCmd.Flags().IntVar(&limit, "limit", 2000, "max output rows (protects console)")

	cmd.AddCommand(queryCmd)

	var allNamespaces bool
	monitorCmd := &cobra.Command{
		Use:          "monitor",
		Short:        "list ServiceMonitors and PodMonitors (prometheus-operator)",
		Example:      "nexa prometheus monitor -A\n  nexa prometheus monitor -n base-services",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			cfg, err := newKubeRESTConfig(kubeconfig)
			if err != nil {
				return err
			}
			dyn, err := dynamic.NewForConfig(cfg)
			if err != nil {
				return err
			}

			ns := namespace
			if allNamespaces {
				ns = ""
			}

			sm, err := listMonitors(ctx, dyn, ns, schema.GroupVersionResource{
				Group:    "monitoring.coreos.com",
				Version:  "v1",
				Resource: "servicemonitors",
			})
			if err != nil {
				return err
			}
			pm, err := listMonitors(ctx, dyn, ns, schema.GroupVersionResource{
				Group:    "monitoring.coreos.com",
				Version:  "v1",
				Resource: "podmonitors",
			})
			if err != nil {
				return err
			}

			if err := renderMonitorTable(os.Stdout, "ServiceMonitor", sm, limit); err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout)
			if err := renderMonitorTable(os.Stdout, "PodMonitor", pm, limit); err != nil {
				return err
			}
			fmt.Fprintln(os.Stdout)
			fmt.Fprintf(os.Stdout, "Total: ServiceMonitor=%d, PodMonitor=%d\n", len(sm), len(pm))
			return nil
		},
	}
	monitorCmd.Flags().BoolVarP(&allNamespaces, "all-namespaces", "A", false, "list across all namespaces")
	cmd.AddCommand(monitorCmd)

	listCmd := &cobra.Command{
		Use:          "list",
		Short:        "list all metric names from Prometheus",
		Example:      "nexa prometheus list -n base-services",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			cli, err := newKubeClient(kubeconfig)
			if err != nil {
				return err
			}
			host, port, err := discoverPrometheus(ctx, cli, namespace, service, selector, portName)
			if err != nil {
				return err
			}
			baseURL := fmt.Sprintf("http://%s", net.JoinHostPort(host, port))

			names, err := promMetricNames(ctx, baseURL)
			if err != nil {
				return err
			}
			return renderMetricNames(os.Stdout, names, limit)
		},
	}
	cmd.AddCommand(listCmd)

	targetsCmd := &cobra.Command{
		Use:          "targets",
		Short:        "show Prometheus scrape targets (like /targets page)",
		Example:      "nexa prometheus targets --address 'http://10.161.42.222:30090/targets'\n  nexa prometheus targets -n base-services",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
			defer cancel()

			baseURL := address
			if baseURL == "" {
				cli, err := newKubeClient(kubeconfig)
				if err != nil {
					return err
				}
				host, port, err := discoverPrometheus(ctx, cli, namespace, service, selector, portName)
				if err != nil {
					return err
				}
				baseURL = fmt.Sprintf("http://%s", net.JoinHostPort(host, port))
			}

			active, dropped, err := promTargetsRows(ctx, baseURL)
			if err != nil {
				return err
			}
			// When we can access K8s, detect "scrape pools with no active targets" (Prometheus API doesn't list them).
			var expectedPools []string
			if cfg, err := newKubeRESTConfig(kubeconfig); err == nil {
				if dyn, derr := dynamic.NewForConfig(cfg); derr == nil {
					expectedPools = expectedScrapePools(ctx, dyn, namespace)
				}
			}

			return renderTargets(os.Stdout, active, dropped, expectedPools, limit)
		},
	}
	// Allow direct proxy testing (e.g. http://.../targets) without discovery.
	targetsCmd.Flags().StringVar(&address, "address", "", "Prometheus base URL or /targets page URL (skip discovery)")
	cmd.AddCommand(targetsCmd)

	return []*cobra.Command{cmd}
}

func newKubeClient(kubeconfig string) (*kubernetes.Clientset, error) {
	cfg, err := newKubeRESTConfig(kubeconfig)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func newKubeRESTConfig(kubeconfig string) (*rest.Config, error) {
	cfg, err := rest.InClusterConfig()
	if err != nil {
		loading := clientcmd.NewDefaultClientConfigLoadingRules()
		if kubeconfig != "" {
			loading.ExplicitPath = kubeconfig
		}
		overrides := &clientcmd.ConfigOverrides{}
		cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loading, overrides)
		cfg, err = cc.ClientConfig()
		if err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

type monitorRow struct {
	Namespace string
	Name      string
	Age       string
	Endpoints int
	Selector  string
}

func listMonitors(ctx context.Context, dyn dynamic.Interface, namespace string, gvr schema.GroupVersionResource) ([]monitorRow, error) {
	var (
		list *unstructured.UnstructuredList
		err  error
	)
	if namespace != "" {
		list, err = dyn.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = dyn.Resource(gvr).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}

	rows := make([]monitorRow, 0, len(list.Items))
	for _, it := range list.Items {
		rows = append(rows, monitorRow{
			Namespace: it.GetNamespace(),
			Name:      it.GetName(),
			Age:       formatAge(it),
			Endpoints: countMonitorEndpoints(&it),
			Selector:  formatMonitorSelector(&it),
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Namespace == rows[j].Namespace {
			return rows[i].Name < rows[j].Name
		}
		return rows[i].Namespace < rows[j].Namespace
	})
	return rows, nil
}

func formatAge(u unstructured.Unstructured) string {
	ct := u.GetCreationTimestamp()
	if ct.IsZero() {
		return ""
	}
	d := time.Since(ct.Time)
	if d < time.Minute {
		return strconv.Itoa(int(d.Seconds())) + "s"
	}
	if d < time.Hour {
		return strconv.Itoa(int(d.Minutes())) + "m"
	}
	if d < 24*time.Hour {
		return strconv.Itoa(int(d.Hours())) + "h"
	}
	return strconv.Itoa(int(d.Hours()/24)) + "d"
}

func countMonitorEndpoints(u *unstructured.Unstructured) int {
	spec, ok := u.Object["spec"].(map[string]any)
	if !ok {
		return 0
	}
	// ServiceMonitor: spec.endpoints
	if eps, ok := spec["endpoints"]; ok {
		if arr, ok := eps.([]any); ok {
			return len(arr)
		}
	}
	// PodMonitor: spec.podMetricsEndpoints
	if eps, ok := spec["podMetricsEndpoints"]; ok {
		if arr, ok := eps.([]any); ok {
			return len(arr)
		}
	}
	return 0
}

func formatMonitorSelector(u *unstructured.Unstructured) string {
	spec, ok := u.Object["spec"].(map[string]any)
	if !ok {
		return ""
	}
	sel, ok := spec["selector"].(map[string]any)
	if !ok {
		return ""
	}
	ml, ok := sel["matchLabels"].(map[string]any)
	if !ok || len(ml) == 0 {
		return ""
	}
	keys := make([]string, 0, len(ml))
	for k := range ml {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, ml[k]))
	}
	return strings.Join(parts, ",")
}

func renderMonitorTable(w io.Writer, title string, rows []monitorRow, limit int) error {
	if limit <= 0 {
		limit = 2000
	}
	fmt.Fprintf(w, "%s (%d)\n", title, len(rows))
	t := tablewriter.NewWriter(w)
	t.Header([]string{"NAMESPACE", "NAME", "ENDPOINTS", "SELECTOR", "AGE"})

	printed := 0
	for _, r := range rows {
		if printed >= limit {
			break
		}
		_ = t.Append([]string{r.Namespace, r.Name, strconv.Itoa(r.Endpoints), r.Selector, r.Age})
		printed++
	}
	if err := t.Render(); err != nil {
		return err
	}
	if len(rows) > printed {
		fmt.Fprintf(w, "(truncated to %d rows; use --limit)\n", printed)
	}
	return nil
}

func discoverPrometheus(ctx context.Context, cli *kubernetes.Clientset, namespace, serviceName, selector, portName string) (host string, port string, err error) {
	if namespace == "" {
		namespace = "default"
	}

	var svc *corev1.Service
	if serviceName != "" {
		s, gerr := cli.CoreV1().Services(namespace).Get(ctx, serviceName, metav1.GetOptions{})
		if gerr != nil {
			return "", "", gerr
		}
		svc = s
	} else {
		sel := strings.TrimSpace(selector)
		if sel == "" {
			// Common labels used by kube-prometheus-stack / prometheus-operator / standalone deployments.
			// We'll try a few selectors in order, stopping at the first match.
			candidates := []string{
				"app.kubernetes.io/name=prometheus",
				"app=prometheus",
				"app.kubernetes.io/component=prometheus",
				"operated-prometheus=true",
			}
			for _, c := range candidates {
				list, lerr := cli.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: c})
				if lerr != nil {
					continue
				}
				if len(list.Items) > 0 {
					svc = &list.Items[0]
					break
				}
			}
		} else {
			list, lerr := cli.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: sel})
			if lerr != nil {
				return "", "", lerr
			}
			if len(list.Items) == 0 {
				return "", "", fmt.Errorf("no Service matched selector %q in namespace %q", sel, namespace)
			}
			svc = &list.Items[0]
		}
	}

	if svc == nil {
		return "", "", fmt.Errorf("unable to discover Prometheus Service in namespace %q (try --service or --selector or --address)", namespace)
	}

	p, perr := pickServicePort(svc, portName)
	if perr != nil {
		return "", "", perr
	}

	// Prefer Service ClusterIP when available (works inside cluster).
	if svc.Spec.ClusterIP != "" && svc.Spec.ClusterIP != "None" {
		return svc.Spec.ClusterIP, fmt.Sprintf("%d", p.Port), nil
	}

	// Fallback to Endpoints (pod IPs), useful for headless Services.
	ep, eerr := cli.CoreV1().Endpoints(namespace).Get(ctx, svc.Name, metav1.GetOptions{})
	if eerr != nil {
		return "", "", eerr
	}
	for _, ss := range ep.Subsets {
		pp := pickEndpointPort(ss.Ports, portName, int(p.Port))
		if pp == 0 {
			continue
		}
		for _, addr := range ss.Addresses {
			if addr.IP == "" {
				continue
			}
			return addr.IP, fmt.Sprintf("%d", pp), nil
		}
	}

	return "", "", fmt.Errorf("found Service %q but could not resolve an address/port (no ClusterIP; endpoints empty)", svc.Name)
}

func pickServicePort(svc *corev1.Service, portName string) (corev1.ServicePort, error) {
	if svc == nil {
		return corev1.ServicePort{}, errors.New("nil service")
	}
	if len(svc.Spec.Ports) == 0 {
		return corev1.ServicePort{}, fmt.Errorf("service %q has no ports", svc.Name)
	}
	if portName != "" {
		for _, p := range svc.Spec.Ports {
			if p.Name == portName {
				return p, nil
			}
		}
		return corev1.ServicePort{}, fmt.Errorf("service %q has no port named %q", svc.Name, portName)
	}
	// Prefer typical Prometheus port.
	for _, p := range svc.Spec.Ports {
		if p.Port == 9090 {
			return p, nil
		}
	}
	return svc.Spec.Ports[0], nil
}

func pickEndpointPort(ports []corev1.EndpointPort, portName string, servicePort int) int32 {
	if len(ports) == 0 {
		return 0
	}
	if portName != "" {
		for _, p := range ports {
			if p.Name == portName && p.Port != 0 {
				return p.Port
			}
		}
	}
	if servicePort != 0 {
		for _, p := range ports {
			if int(p.Port) == servicePort {
				return p.Port
			}
		}
	}
	for _, p := range ports {
		if p.Port == 9090 {
			return p.Port
		}
	}
	if ports[0].Port == 0 {
		return 0
	}
	return ports[0].Port
}

type promAPIResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
	Data      struct {
		ResultType string           `json:"resultType"`
		Result     []promVectorItem `json:"result"`
	} `json:"data"`
}

type promVectorItem struct {
	Metric map[string]string `json:"metric"`
	// value is [ <unix_time>, "<sample_value>" ]
	Value []any `json:"value"`
}

type promLabelValuesResponse struct {
	Status    string   `json:"status"`
	ErrorType string   `json:"errorType,omitempty"`
	Error     string   `json:"error,omitempty"`
	Data      []string `json:"data"`
}

func promAPIURL(baseURL string, apiPath string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return nil, err
	}
	// If caller passes something like http://host:port/targets (proxy), drop path and point to API.
	u.Path = apiPath
	u.RawQuery = ""
	u.Fragment = ""
	return u, nil
}

type promTargetsResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType,omitempty"`
	Error     string `json:"error,omitempty"`
	Data      struct {
		ActiveTargets []struct {
			ScrapePool         string            `json:"scrapePool"`
			ScrapeURL          string            `json:"scrapeUrl"`
			Labels             map[string]string `json:"labels"`
			DiscoveredLabels   map[string]string `json:"discoveredLabels"`
			Health             string            `json:"health"`
			LastScrape         string            `json:"lastScrape"`
			LastScrapeDuration float64           `json:"lastScrapeDuration"`
			ScrapeInterval     string            `json:"scrapeInterval"`
			ScrapeTimeout      string            `json:"scrapeTimeout"`
			LastError          string            `json:"lastError"`
		} `json:"activeTargets"`
		DroppedTargets []struct {
			DiscoveredLabels map[string]string `json:"discoveredLabels"`
		} `json:"droppedTargets"`
	} `json:"data"`
}

func promQuery(ctx context.Context, baseURL string, q string) (*promAPIResponse, error) {
	u, err := promAPIURL(baseURL, "/api/v1/query")
	if err != nil {
		return nil, err
	}
	qq := u.Query()
	qq.Set("query", q)
	u.RawQuery = qq.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("prometheus http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out promAPIResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.Status != "success" {
		if out.Error != "" {
			return nil, fmt.Errorf("prometheus query failed: %s (%s)", out.Error, out.ErrorType)
		}
		return nil, fmt.Errorf("prometheus query failed: status=%s", out.Status)
	}
	return &out, nil
}

func promTargetsLastErrorByKey(ctx context.Context, baseURL string) (map[string]string, error) {
	u, err := promAPIURL(baseURL, "/api/v1/targets")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("prometheus http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out promTargetsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.Status != "success" {
		if out.Error != "" {
			return nil, fmt.Errorf("prometheus targets failed: %s (%s)", out.Error, out.ErrorType)
		}
		return nil, fmt.Errorf("prometheus targets failed: status=%s", out.Status)
	}

	m := make(map[string]string, len(out.Data.ActiveTargets))
	for _, t := range out.Data.ActiveTargets {
		k := upTargetKey(t.Labels)
		if k == "" {
			continue
		}
		if t.LastError != "" {
			m[k] = t.LastError
		}
	}
	return m, nil
}

type promTargetRow struct {
	ScrapePool string
	Job        string
	Instance   string
	Health     string
	LastError  string
	ScrapeURL  string
	Namespace  string
	Pod        string
	Service    string
	Endpoint   string
	LastScrape string
	Duration   string
	Interval   string
	Timeout    string
	Labels     string
	Discovered string
}

func promTargetsRows(ctx context.Context, baseURL string) (active []promTargetRow, dropped []string, err error) {
	u, err := promAPIURL(baseURL, "/api/v1/targets")
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("prometheus http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out promTargetsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, nil, err
	}
	if out.Status != "success" {
		if out.Error != "" {
			return nil, nil, fmt.Errorf("prometheus targets failed: %s (%s)", out.Error, out.ErrorType)
		}
		return nil, nil, fmt.Errorf("prometheus targets failed: status=%s", out.Status)
	}

	active = make([]promTargetRow, 0, len(out.Data.ActiveTargets))
	for _, t := range out.Data.ActiveTargets {
		lbl := formatMap(t.Labels, []string{})
		disc := formatMap(t.DiscoveredLabels, []string{})
		active = append(active, promTargetRow{
			ScrapePool: t.ScrapePool,
			Job:        t.Labels["job"],
			Instance:   t.Labels["instance"],
			Health:     t.Health,
			LastError:  t.LastError,
			ScrapeURL:  t.ScrapeURL,
			Namespace:  t.Labels["namespace"],
			Pod:        t.Labels["pod"],
			Service:    t.Labels["service"],
			Endpoint:   t.Labels["endpoint"],
			LastScrape: t.LastScrape,
			Duration:   fmt.Sprintf("%.3fs", t.LastScrapeDuration),
			Interval:   t.ScrapeInterval,
			Timeout:    t.ScrapeTimeout,
			Labels:     lbl,
			Discovered: disc,
		})
	}
	sort.Slice(active, func(i, j int) bool {
		if active[i].ScrapePool == active[j].ScrapePool {
			if active[i].Job == active[j].Job {
				return active[i].Instance < active[j].Instance
			}
			return active[i].Job < active[j].Job
		}
		return active[i].ScrapePool < active[j].ScrapePool
	})

	dropped = make([]string, 0, len(out.Data.DroppedTargets))
	for _, t := range out.Data.DroppedTargets {
		// Prometheus exposes only discoveredLabels here; try to surface something meaningful.
		if t.DiscoveredLabels == nil {
			continue
		}
		// prefer job/instance if present
		job := t.DiscoveredLabels["job"]
		inst := t.DiscoveredLabels["instance"]
		scrapePool := t.DiscoveredLabels["__scrape_pool__"]
		switch {
		case scrapePool != "" && job != "" && inst != "":
			dropped = append(dropped, fmt.Sprintf("%s job=%s instance=%s", scrapePool, job, inst))
		case job != "" && inst != "":
			dropped = append(dropped, fmt.Sprintf("job=%s instance=%s", job, inst))
		case scrapePool != "":
			dropped = append(dropped, scrapePool)
		default:
			// fallback to a compact summary
			if u := t.DiscoveredLabels["__address__"]; u != "" {
				dropped = append(dropped, "__address__="+u)
			}
		}
	}
	sort.Strings(dropped)
	return active, dropped, nil
}

func upTargetKey(labels map[string]string) string {
	if labels == nil {
		return ""
	}
	// job+instance should be unique for Prometheus target labelsets; add a few common k8s labels to be safer.
	parts := []string{
		"job=" + labels["job"],
		"instance=" + labels["instance"],
	}
	for _, k := range []string{"namespace", "pod", "service", "endpoint", "metrics_path"} {
		if v, ok := labels[k]; ok && v != "" {
			parts = append(parts, k+"="+v)
		}
	}
	return strings.Join(parts, "|")
}

func promMetricNames(ctx context.Context, baseURL string) ([]string, error) {
	u, err := promAPIURL(baseURL, "/api/v1/label/__name__/values")
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("prometheus http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out promLabelValuesResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.Status != "success" {
		if out.Error != "" {
			return nil, fmt.Errorf("prometheus label values failed: %s (%s)", out.Error, out.ErrorType)
		}
		return nil, fmt.Errorf("prometheus label values failed: status=%s", out.Status)
	}
	sort.Strings(out.Data)
	return out.Data, nil
}

func renderPromResult(ctx context.Context, w io.Writer, baseURL string, query string, res *promAPIResponse, limit int) error {
	if res == nil {
		return errors.New("nil response")
	}
	if limit <= 0 {
		limit = 2000
	}

	if res.Data.ResultType != "vector" {
		// Keep it simple for now; users can still see raw structure.
		b, _ := json.MarshalIndent(res, "", "  ")
		_, _ = fmt.Fprintf(w, "Unsupported resultType=%q (only vector is rendered as table)\n%s\n", res.Data.ResultType, string(b))
		return nil
	}

	// When target disappears, `up` series can be absent (not 0). In that case, `up==0` returns empty,
	// but Prometheus UI still shows scrape pools with 0 active targets. Provide a helpful fallback.
	if len(res.Data.Result) == 0 && isUpEqualsZero(query) && ctx != nil && baseURL != "" {
		if err := renderDownTargetsFallback(ctx, w, baseURL, limit); err == nil {
			return nil
		}
		// if fallback fails, continue to render empty table like before
	}

	t := tablewriter.NewWriter(w)
	t.Header([]string{"Metric", "Labels", "Value", "Desc"})

	// Only used for `up==0` style queries (or any query that returns up=0 rows).
	var lastErr map[string]string

	printed := 0
	truncated := false
	for _, it := range res.Data.Result {
		if printed >= limit {
			truncated = true
			break
		}
		metricName := it.Metric["__name__"]
		labels := formatLabels(it.Metric)
		val := formatPromValue(it.Value)
		desc := ""
		if metricName == "up" {
			if fv, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
				if fv >= 1 {
					desc = "UP"
				} else {
					if lastErr == nil && ctx != nil && baseURL != "" {
						m, _ := promTargetsLastErrorByKey(ctx, baseURL)
						lastErr = m
					}
					if lastErr != nil {
						if e, ok := lastErr[upTargetKey(it.Metric)]; ok && strings.TrimSpace(e) != "" {
							desc = e
						} else {
							desc = "DOWN (no lastError)"
						}
					} else {
						desc = "DOWN"
					}
				}
			}
		}
		_ = t.Append([]string{metricName, labels, val, desc})
		printed++
	}

	if err := t.Render(); err != nil {
		return err
	}
	if truncated {
		_, _ = fmt.Fprintf(w, "\n(truncated to %d rows; use --limit)\n", limit)
	}
	return nil
}

func isUpEqualsZero(q string) bool {
	s := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(q)), " ", "")
	return s == "up==0" || s == "up==0.0" || s == "up==false"
}

func renderDownTargetsFallback(ctx context.Context, w io.Writer, baseURL string, limit int) error {
	active, dropped, err := promTargetsRows(ctx, baseURL)
	if err != nil {
		return err
	}

	down := make([]promTargetRow, 0)
	for _, t := range active {
		if strings.EqualFold(t.Health, "down") {
			down = append(down, t)
		}
	}

	if len(down) == 0 && len(dropped) == 0 {
		return nil
	}

	if len(down) > 0 {
		fmt.Fprintf(w, "Active targets with health=down (%d)\n", len(down))
		tw := tablewriter.NewWriter(w)
		tw.Header([]string{"ScrapePool", "Job", "Instance", "Namespace", "Pod", "Service", "Endpoint", "Desc"})
		printed := 0
		for _, r := range down {
			if printed >= limit {
				break
			}
			desc := r.LastError
			if strings.TrimSpace(desc) == "" {
				desc = "DOWN (no lastError)"
			}
			_ = tw.Append([]string{r.ScrapePool, r.Job, r.Instance, r.Namespace, r.Pod, r.Service, r.Endpoint, desc})
			printed++
		}
		if err := tw.Render(); err != nil {
			return err
		}
		if len(down) > printed {
			fmt.Fprintf(w, "(truncated to %d rows; use --limit)\n", printed)
		}
	}

	if len(dropped) > 0 {
		if len(down) > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "Dropped targets (no active targets matched) (%d)\n", len(dropped))
		tw := tablewriter.NewWriter(w)
		tw.Header([]string{"Target"})
		printed := 0
		for _, s := range dropped {
			if printed >= limit {
				break
			}
			_ = tw.Append([]string{s})
			printed++
		}
		if err := tw.Render(); err != nil {
			return err
		}
		if len(dropped) > printed {
			fmt.Fprintf(w, "(truncated to %d rows; use --limit)\n", printed)
		}
	}
	return nil
}

func renderMetricNames(w io.Writer, names []string, limit int) error {
	if limit <= 0 {
		limit = 2000
	}
	t := tablewriter.NewWriter(w)
	t.Header([]string{"Metric"})

	printed := 0
	for _, n := range names {
		if printed >= limit {
			break
		}
		_ = t.Append([]string{n})
		printed++
	}
	if err := t.Render(); err != nil {
		return err
	}
	if len(names) > printed {
		fmt.Fprintf(w, "\n(truncated to %d rows; use --limit)\n", printed)
	}
	return nil
}

func formatLabels(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		if k == "__name__" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sb := strings.Builder{}
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(m[k])
	}
	return sb.String()
}

func formatMap(m map[string]string, excludeKeys []string) string {
	if len(m) == 0 {
		return ""
	}
	ex := map[string]struct{}{}
	for _, k := range excludeKeys {
		ex[k] = struct{}{}
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		if _, ok := ex[k]; ok {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(m[k])
	}
	return sb.String()
}

func formatPromValue(v []any) string {
	// [ <unix_time>, "<sample_value>" ]
	if len(v) != 2 {
		b, _ := json.Marshal(v)
		return string(b)
	}
	if s, ok := v[1].(string); ok {
		return s
	}
	b, _ := json.Marshal(v[1])
	return string(b)
}

func renderTargets(w io.Writer, active []promTargetRow, dropped []string, expectedPools []string, limit int) error {
	if limit <= 0 {
		limit = 2000
	}

	// Scrape pool summary (closer to /targets UI grouping).
	type poolAgg struct {
		Total int
		Up    int
		Down  int
		Other int
		// keep the latest timestamp string for readability (Prometheus returns RFC3339Nano).
		LastScrape string
	}
	pools := map[string]*poolAgg{}
	for _, t := range active {
		p := t.ScrapePool
		if p == "" {
			p = "(unknown)"
		}
		agg := pools[p]
		if agg == nil {
			agg = &poolAgg{}
			pools[p] = agg
		}
		agg.Total++
		switch strings.ToLower(t.Health) {
		case "up":
			agg.Up++
		case "down":
			agg.Down++
		default:
			agg.Other++
		}
		if t.LastScrape != "" && t.LastScrape > agg.LastScrape {
			agg.LastScrape = t.LastScrape
		}
	}
	poolNames := make([]string, 0, len(pools))
	for k := range pools {
		poolNames = append(poolNames, k)
	}
	sort.Strings(poolNames)

	fmt.Fprintf(w, "Scrape pools (with active targets): %d\n", len(poolNames))
	ts := tablewriter.NewWriter(w)
	ts.Header([]string{"ScrapePool", "Active", "Up", "Down", "Other", "LastScrape"})
	printed := 0
	for _, p := range poolNames {
		if printed >= limit {
			break
		}
		agg := pools[p]
		_ = ts.Append([]string{
			p,
			strconv.Itoa(agg.Total),
			strconv.Itoa(agg.Up),
			strconv.Itoa(agg.Down),
			strconv.Itoa(agg.Other),
			agg.LastScrape,
		})
		printed++
	}
	if err := ts.Render(); err != nil {
		return err
	}
	if len(poolNames) > printed {
		fmt.Fprintf(w, "(truncated to %d rows; use --limit)\n", printed)
	}

	// Pools that exist in ServiceMonitor/PodMonitor but have no active targets.
	if len(expectedPools) > 0 {
		expSet := map[string]struct{}{}
		for _, p := range expectedPools {
			expSet[p] = struct{}{}
		}
		for _, p := range poolNames {
			delete(expSet, p)
		}
		missing := make([]string, 0, len(expSet))
		for p := range expSet {
			missing = append(missing, p)
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			fmt.Fprintln(w)
			fmt.Fprintf(w, "Scrape pools with NO active targets (%d)\n", len(missing))
			tm := tablewriter.NewWriter(w)
			tm.Header([]string{"ScrapePool", "State"})
			printed = 0
			for _, p := range missing {
				if printed >= limit {
					break
				}
				_ = tm.Append([]string{p, "NO ACTIVE TARGETS"})
				printed++
			}
			if err := tm.Render(); err != nil {
				return err
			}
			if len(missing) > printed {
				fmt.Fprintf(w, "(truncated to %d rows; use --limit)\n", printed)
			}
		}
	}

	fmt.Fprintln(w)
	fmt.Fprintf(w, "Active targets: total=%d\n", len(active))
	tw := tablewriter.NewWriter(w)
	tw.Header([]string{"ScrapePool", "Health", "Job", "Instance", "Namespace", "Pod", "Service", "Endpoint", "LastScrape", "Duration", "Interval", "Timeout", "ScrapeURL", "LastError"})

	printed = 0
	for _, r := range active {
		if printed >= limit {
			break
		}
		le := r.LastError
		if strings.TrimSpace(le) == "" && strings.EqualFold(r.Health, "down") {
			le = "DOWN (no lastError)"
		}
		_ = tw.Append([]string{r.ScrapePool, r.Health, r.Job, r.Instance, r.Namespace, r.Pod, r.Service, r.Endpoint, r.LastScrape, r.Duration, r.Interval, r.Timeout, r.ScrapeURL, le})
		printed++
	}
	if err := tw.Render(); err != nil {
		return err
	}
	if len(active) > printed {
		fmt.Fprintf(w, "(truncated to %d rows; use --limit)\n", printed)
	}

	if len(dropped) > 0 {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Dropped targets (%d)\n", len(dropped))
		td := tablewriter.NewWriter(w)
		td.Header([]string{"Target"})
		printed = 0
		for _, s := range dropped {
			if printed >= limit {
				break
			}
			_ = td.Append([]string{s})
			printed++
		}
		if err := td.Render(); err != nil {
			return err
		}
		if len(dropped) > printed {
			fmt.Fprintf(w, "(truncated to %d rows; use --limit)\n", printed)
		}
	}

	return nil
}

func expectedScrapePools(ctx context.Context, dyn dynamic.Interface, namespace string) []string {
	// We mirror Prometheus operator scrape pool naming:
	// - serviceMonitor/<namespace>/<name>/<endpointIndex>
	// - podMonitor/<namespace>/<name>/<endpointIndex>
	// If namespace is empty, we list across all namespaces.
	ctx = withBackground(ctx)

	pools := map[string]struct{}{}

	addMonitor := func(kind string, ns string, name string, endpoints int) {
		if ns == "" || name == "" || endpoints <= 0 {
			return
		}
		for i := 0; i < endpoints; i++ {
			pools[fmt.Sprintf("%s/%s/%s/%d", kind, ns, name, i)] = struct{}{}
		}
	}

	// ServiceMonitors
	sms, err := listUnstructured(ctx, dyn, namespace, schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "servicemonitors",
	})
	if err == nil {
		for _, it := range sms {
			addMonitor("serviceMonitor", it.GetNamespace(), it.GetName(), countMonitorEndpoints(&it))
		}
	}

	// PodMonitors
	pms, err := listUnstructured(ctx, dyn, namespace, schema.GroupVersionResource{
		Group:    "monitoring.coreos.com",
		Version:  "v1",
		Resource: "podmonitors",
	})
	if err == nil {
		for _, it := range pms {
			addMonitor("podMonitor", it.GetNamespace(), it.GetName(), countMonitorEndpoints(&it))
		}
	}

	out := make([]string, 0, len(pools))
	for p := range pools {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func withBackground(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	return ctx
}

func listUnstructured(ctx context.Context, dyn dynamic.Interface, namespace string, gvr schema.GroupVersionResource) ([]unstructured.Unstructured, error) {
	var (
		list *unstructured.UnstructuredList
		err  error
	)
	if namespace != "" {
		list, err = dyn.Resource(gvr).Namespace(namespace).List(ctx, metav1.ListOptions{})
	} else {
		list, err = dyn.Resource(gvr).List(ctx, metav1.ListOptions{})
	}
	if err != nil {
		return nil, err
	}
	return list.Items, nil
}
