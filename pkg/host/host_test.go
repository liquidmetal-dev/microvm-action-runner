package host_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/host"
)

func Test_HostAssign(t *testing.T) {
	g := NewWithT(t)

	var (
		host1     = "host1"
		host2     = "host2"
		runnerOne = "runner1"
		runnerTwo = "runner2"
	)

	tt := []struct {
		name           string
		hosts          []string
		runners        []string
		expectedErr    bool
		expectedResult string
	}{
		{
			name:           "when the lists of hosts is empty, returns error",
			hosts:          []string{},
			runners:        []string{runnerOne},
			expectedErr:    true,
			expectedResult: "",
		},
		{
			name:           "when there is only one host in the pool, returns that host",
			hosts:          []string{host1},
			runners:        []string{runnerOne},
			expectedErr:    false,
			expectedResult: host1,
		},
		{
			name:           "when there are two hosts in the pool, returns the host less in use",
			hosts:          []string{host1, host2},
			runners:        []string{runnerOne, runnerTwo},
			expectedErr:    false,
			expectedResult: host1,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			manager := host.New(tc.hosts)

			for _, r := range tc.runners {
				assigned, err := manager.Assign(r)
				if tc.expectedErr {
					g.Expect(err).To(HaveOccurred())
					return
				}

				found, ok := manager.AssignedMap[r]
				g.Expect(ok).To(BeTrue())
				g.Expect(found).To(Equal(assigned))
				g.Expect(manager.HostCount[found]).To(Equal(1))
			}
		})
	}
}

func Test_HostLookup(t *testing.T) {
	g := NewWithT(t)

	var (
		host1     = "host1"
		runnerOne = "runner1"
	)

	manager := host.New([]string{host1})
	assigned, err := manager.Assign(runnerOne)
	g.Expect(err).NotTo(HaveOccurred())

	found, err := manager.Lookup(runnerOne)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(found).To(Equal(assigned))
}

func Test_HostLookupFails(t *testing.T) {
	g := NewWithT(t)

	var (
		host1     = "host1"
		runnerOne = "runner1"
	)

	manager := host.New([]string{host1})
	_, err := manager.Lookup(runnerOne)
	g.Expect(err).To(HaveOccurred())
}

func Test_HostUnassign(t *testing.T) {
	g := NewWithT(t)

	var (
		host1      = "host1"
		runnerName = "runner1"
	)

	manager := host.New([]string{host1})
	assigned, err := manager.Assign(runnerName)
	g.Expect(err).NotTo(HaveOccurred())

	manager.Unassign(runnerName)
	_, ok := manager.AssignedMap[runnerName]
	g.Expect(ok).To(BeFalse())
	g.Expect(manager.HostCount[assigned]).To(Equal(0))
}
