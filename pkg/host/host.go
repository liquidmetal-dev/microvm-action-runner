package host

import (
	"errors"
	"fmt"
	"sort"
)

// Manager is an object which assigns, records and unassigns the hosts to each runner
type Manager struct {
	hosts []string
	// AssignedMap is a record of each runner and its assigned host
	AssignedMap map[string]string
	// HostCount is a counter for each host to keep track of which is most in use
	HostCount map[string]int
}

// New returns a new HostManager
func New(hosts []string) *Manager {
	var (
		hc = map[string]int{}
		am = map[string]string{}
	)

	for _, h := range hosts {
		hc[h] = 0
	}

	return &Manager{
		hosts:       hosts,
		HostCount:   hc,
		AssignedMap: am,
	}
}

// Assign will very naively find the "least busy" host to schedule an runner onto.
// The record is stored in memory and thus will not survive restarting the service,
// so if you start an action, kill the service, then restart it, the tool will
// not be able to discover where it was scheduled. Cleanup will then
// have to be manual.
func (m *Manager) Assign(name string) (string, error) {
	var host string
	switch len(m.hosts) {
	case 0:
		// technically this will never happen since at least one host is required by
		// the command, but just in case...
		return "", errors.New("no host found")
	case 1:
		host = m.hosts[0]
	default:
		host = m.findAvailableHost()
	}

	m.saveHost(host, name)

	return host, nil
}

// Lookup will find the host assigned to the runner.
func (m *Manager) Lookup(name string) (string, error) {
	h, ok := m.AssignedMap[name]
	if !ok {
		return "", fmt.Errorf("host for runner %s not found", name)
	}

	return h, nil
}

// Unassign will remove the record of the runner from the Manager
func (m *Manager) Unassign(name string) {
	h, _ := m.Lookup(name)
	delete(m.AssignedMap, name)
	m.HostCount[h]--
}

func (m *Manager) findAvailableHost() string {
	keys := make([]string, 0, len(m.HostCount))

	for key := range m.HostCount {
		keys = append(keys, key)
	}

	sort.SliceStable(keys, func(i, j int) bool {
		return m.HostCount[keys[i]] < m.HostCount[keys[j]]
	})

	return keys[0]
}

func (m *Manager) saveHost(host, runner string) {
	m.AssignedMap[runner] = host
	m.HostCount[host]++
}
