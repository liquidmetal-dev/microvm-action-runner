package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/webhooks/v6/github"
	"github.com/sirupsen/logrus"
	"github.com/warehouse-13/hammertime/pkg/client"

	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/config"
	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/host"
	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/microvm"
	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/payload"
)

const (
	eventQueued    = "queued"
	eventCompleted = "completed"
)

type ClientFunc func(string) (client.FlintlockClient, error)

// NewFlintClient is a wrapper around client.New to disguise the fact that this
// service does not give the option for auth right now
func NewFlintClient(addr string) (client.FlintlockClient, error) {
	noAuthYet := ""
	return client.New(addr, noAuthYet)
}

type handler struct {
	Params
}

// Params groups the init opts for a New handler object
type Params struct {
	*config.Config
	Client  ClientFunc
	Payload payload.Payload
	// TODO interface instead?
	HostManager *host.Manager
	L           *logrus.Entry
}

// New returns a new handler
func New(p Params) (handler, error) {
	if p.Client == nil {
		return handler{}, errors.New("func to generate FlintlockClient not provided")
	}

	if p.L == nil {
		return handler{}, errors.New("logger not provided")
	}

	if p.Payload == nil {
		return handler{}, errors.New("payload interface not fulfilled")
	}

	if p.HostManager == nil {
		return handler{}, errors.New("host manager not provided")
	}

	return handler{
		Params: p,
	}, nil
}

// HandleWebhookPost will respond to calls to the /webhook endpoint
// It will Parse the payload and will proceed if the payload contains a
// WorkflowJobPayload. From there it will either act on "queued" or "completed"
// events. Anything else is ignored.
func (h handler) HandleWebhookPost(w http.ResponseWriter, r *http.Request) {
	h.L.Debug("webhook received")

	event, err := h.Payload.Parse(r)
	if err != nil {
		h.L.Errorf("%d failed to parse webhook payload: %s", http.StatusInternalServerError, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if event == nil {
		h.L.Debug("payload type is unknown")
		w.WriteHeader(http.StatusOK)
		return
	}

	h.L.Debugf("workflow event found %s", event.WorkflowJob.RunURL)

	if !h.shouldRun(event.WorkflowJob.Labels) {
		h.L.Debug("required labels not set, ignoring event")
		w.WriteHeader(http.StatusOK)
		return
	}

	switch event.Action {
	case eventQueued:
		if err := h.processQueuedAction(*event); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case eventCompleted:
		if err := h.processCompletedAction(*event); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	default:
		h.L.Debugf("event type is unknown: %s", event.Action)
	}

	w.WriteHeader(http.StatusOK)
}

func (h handler) processQueuedAction(p github.WorkflowJobPayload) error {
	h.L.Infof("proccessing queued action for workflow (job-id: %d) (step-id: %d)", p.WorkflowJob.RunID, p.WorkflowJob.ID)

	name := generateName(p)

	host, err := h.HostManager.Assign(name)
	if err != nil {
		h.L.Errorf("failed to assign host to runner: %s", err)
		return err
	}

	fl, err := h.Client(host)
	if err != nil {
		h.L.Errorf("failed to create flintlock client: %s", err)
		return err
	}

	defer func() {
		if err := fl.Close(); err != nil {
			h.L.Errorf("failed to close connection to flintlock host to %s: %s", host, err)
		}
	}()

	mvm, err := microvm.New(
		microvm.UserdataCfg{
			GithubToken: h.APIToken,
			PublicKey:   h.SSHPublicKey,
			User:        h.Username,
			Repo:        h.Repository,
			Id:          name,
			Labels:      h.Labels,
		},
	)
	if err != nil {
		h.L.Errorf("failed to generate microvm spec: %s", err)
		return err
	}

	h.L.Debugf("creating microvm %s", name)
	created, err := fl.Create(mvm)
	if err != nil {
		h.L.Errorf("failed to create microvm: %s", err)
		return err
	}

	h.L.Infof("created microvm, name: %s, uid: %s", name, *created.Microvm.Spec.Uid)

	return nil
}

func (h handler) processCompletedAction(p github.WorkflowJobPayload) error {
	h.L.Infof("proccessing complete action for workflow (job-id: %d) (step-id: %d)", p.WorkflowJob.RunID, p.WorkflowJob.ID)

	name := generateName(p)

	host, err := h.HostManager.Lookup(name)
	if err != nil {
		h.L.Errorf("failed to look up host for runner: %s", err)
		return err
	}

	fl, err := h.Client(host)
	if err != nil {
		h.L.Errorf("failed to create flintlock client %s", err)
		return err
	}

	defer func() {
		if err := fl.Close(); err != nil {
			h.L.Errorf("failed to close connection to flintlock host %s: %s", "address", err)
		}
	}()

	h.L.Debugf("looking up microvm for action %s/%s", microvm.Namespace, name)
	resp, err := fl.List(name, microvm.Namespace)
	if err != nil {
		h.L.Errorf("failed to list microvms %s", err)
		return err
	}

	if len(resp.Microvm) == 0 {
		h.L.Debugf("no microvms found in %s/%s", microvm.Namespace, name)
		return err
	}

	// TODO this is only safe if I am totally sure the name is unique...
	uid := resp.Microvm[0].Spec.Uid

	h.L.Debugf("deleting microvm, name: %s, uid: %s", name, *uid)
	if _, err := fl.Delete(*uid); err != nil {
		h.L.Errorf("failed to delete microvm: %s", err)
		return err
	}

	h.L.Infof("deleted microvm, name: %s, uid: %s", name, *uid)

	h.HostManager.Unassign(name)

	return nil
}

func (h handler) shouldRun(jobLabels []string) bool {
	seen := map[string]struct{}{}
	for _, l := range jobLabels {
		seen[l] = struct{}{}
	}

	for _, l := range h.Labels {
		if _, ok := seen[l]; ok {
			return true
		}
	}

	return false
}

func generateName(p github.WorkflowJobPayload) string {
	return fmt.Sprintf("%s-%d-%d", p.WorkflowJob.NodeID, p.WorkflowJob.ID, p.WorkflowJob.RunID)
}
