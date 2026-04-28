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
	"strings"
	"time"

	"github.com/nexa/pkg/ctx"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
		Use:          "query",
		Short:        "query Prometheus instant query API",
		Example:      `nexa prometheus query --namespace monitoring --query 'up==1'`,
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

			res, err := promQuery(ctx, baseURL, query)
			if err != nil {
				return err
			}
			return renderPromResult(os.Stdout, res, limit)
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

	return []*cobra.Command{cmd}
}

func newKubeClient(kubeconfig string) (*kubernetes.Clientset, error) {
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
	return kubernetes.NewForConfig(cfg)
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

func promQuery(ctx context.Context, baseURL string, q string) (*promAPIResponse, error) {
	u, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, err
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/query"
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

func renderPromResult(w io.Writer, res *promAPIResponse, limit int) error {
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

	t := tablewriter.NewWriter(w)
	t.Header([]string{"Metric", "Labels", "Value"})

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
		_ = t.Append([]string{metricName, labels, val})
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
