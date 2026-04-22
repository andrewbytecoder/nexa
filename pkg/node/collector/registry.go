package collector

import (
	"context"
	"fmt"
	"sort"
)

type Registry struct {
	collectors map[string]Collector
	status     map[string]CollectorStatus

	// linuxEnabledByDefault mirrors the node_exporter README "Enabled by default" names for Linux,
	// but in this project we may implement them gradually.
	linuxEnabledByDefault []string
}

func NewRegistry() *Registry {
	return &Registry{
		collectors: make(map[string]Collector),
		status:     make(map[string]CollectorStatus),
	}
}

func (r *Registry) RegisterImplemented(c Collector) {
	r.collectors[c.Name()] = c
	r.status[c.Name()] = CollectorStatus{
		Name:        c.Name(),
		Description: c.Describe(),
		Implemented: true,
	}
}

func (r *Registry) RegisterPlaceholder(name, desc string) {
	r.status[name] = CollectorStatus{
		Name:        name,
		Description: desc,
		Implemented: false,
	}
}

func (r *Registry) SetLinuxEnabledByDefault(names []string) {
	r.linuxEnabledByDefault = append([]string(nil), names...)
}

func (r *Registry) DefaultCollectorsLinuxEnabledByDefault() []string {
	return append([]string(nil), r.linuxEnabledByDefault...)
}

func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.status))
	for name := range r.status {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func (r *Registry) Has(name string) bool {
	_, ok := r.status[name]
	return ok
}

func (r *Registry) Status(name string) CollectorStatus {
	if st, ok := r.status[name]; ok {
		return st
	}
	return CollectorStatus{Name: name}
}

func (r *Registry) Collect(name string) ([]MetricFamily, error) {
	st, ok := r.status[name]
	if !ok {
		return nil, fmt.Errorf("unknown collector: %s", name)
	}
	if !st.Implemented {
		return nil, fmt.Errorf("collector %s not implemented", name)
	}
	c, ok := r.collectors[name]
	if !ok {
		return nil, fmt.Errorf("collector %s is marked implemented but missing", name)
	}
	return c.Collect(context.Background())
}

