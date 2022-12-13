package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/webhooks/v6/github"
	"github.com/sirupsen/logrus"
	"github.com/warehouse-13/hammertime/pkg/client"

	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/config"
	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/microvm"
	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/payload"
)

const (
	eventQueued    = "queued"
	eventCompleted = "completed"
)

type ClientFunc func(string) (client.FlintlockClient, error)

func NewFlintClient(addr string) (client.FlintlockClient, error) {
	return client.New(addr, "")
}

// TODO unexported?
type Handler struct {
	HandlerParams
	payload payload.Service
}

// HandlerParams groups the init opts for a New Handler object
type HandlerParams struct {
	*config.Config
	Client ClientFunc
	L      *logrus.Entry
}

// New returns a new Handler
func New(p HandlerParams) (Handler, error) {
	if p.Client == nil {
		return Handler{}, errors.New("func to generate FlintlockClient not provided")
	}

	return Handler{
		HandlerParams: p,
		payload:       payload.New(p.WebhookSecret),
	}, nil
}

// HandleWebhookPost will respond to calls to the /webhook endpoint
// It will Parse the payload and will proceed if the payload contains a
// WorkflowJobPayload. From there it will either act on "queued" or "completed"
// events. Anything else is ignored.
func (h Handler) HandleWebhookPost(w http.ResponseWriter, r *http.Request) {
	h.L.Debug("webhook received")

	event, err := h.payload.Parse(r)
	if err != nil {
		h.L.Errorf("failed to parse webhook payload: %s", err)
	}

	if event == nil {
		h.L.Debug("payload type is unknown")
	}

	h.L.Debugf("workflow event found %s", event.WorkflowJob.RunURL)

	switch event.Action {
	case eventQueued:
		h.processQueuedAction(*event)
	case eventCompleted:
		h.processCompletedAction(*event)
	default:
		h.L.Debugf("event type is unknown: %s", event.Action)
	}
}

func (h Handler) processQueuedAction(p github.WorkflowJobPayload) {
	h.L.Infof("proccessing queued action for workflow (job-id: %d) (step-id: %d)", p.WorkflowJob.RunID, p.WorkflowJob.ID)

	fl, err := h.Client(h.Host)
	if err != nil {
		h.L.Errorf("failed to create flintlock client: %s", err)
	}

	defer func() {
		if err := fl.Close(); err != nil {
			h.L.Errorf("failed to close connection to flintlock host to %s: %s", h.Host, err)
		}
	}()

	name := generateName(p)
	mvm, err := microvm.New(h.APIToken, h.SSHPublicKey, name)
	if err != nil {
		h.L.Errorf("failed to generate microvm spec: %s", err)
		return
	}

	h.L.Debugf("creating microvm %s", name)
	created, err := fl.Create(mvm)
	if err != nil {
		h.L.Errorf("failed to create microvm: %s", err)
		return
	}

	h.L.Infof("created microvm, name: %s, uid: %s", name, *created.Microvm.Spec.Uid)
}

func (h Handler) processCompletedAction(p github.WorkflowJobPayload) {
	h.L.Infof("proccessing complete action for workflow (job-id: %d) (step-id: %d)", p.WorkflowJob.RunID, p.WorkflowJob.ID)

	fl, err := h.Client(h.Host)
	if err != nil {
		h.L.Errorf("failed to create flintlock client %s", err)
	}

	defer func() {
		if err := fl.Close(); err != nil {
			h.L.Errorf("failed to close connection to flintlock host %s: %s", "address", err)
		}
	}()

	name := generateName(p)
	h.L.Debugf("looking up microvm for action %s/%s", microvm.Namespace, name)
	resp, err := fl.List(name, microvm.Namespace)
	if err != nil {
		h.L.Errorf("failed to list microvms %s", err)
		return
	}

	if len(resp.Microvm) == 0 {
		h.L.Debugf("no microvms found in %s/%s", microvm.Namespace, name)
		return
	}

	// TODO this is only safe if I am totally sure the name is unique...
	uid := resp.Microvm[0].Spec.Uid

	h.L.Debugf("deleting microvm, name: %s, uid: %s", name, *uid)
	if _, err := fl.Delete(*uid); err != nil {
		h.L.Errorf("failed to delete microvm: %s", err)
		return
	}

	h.L.Infof("deleted microvm, name: %s, uid: %s", name, *uid)
}

func generateName(p github.WorkflowJobPayload) string {
	return fmt.Sprintf("%s-%d-%d", p.WorkflowJob.NodeID, p.WorkflowJob.ID, p.WorkflowJob.RunID)
}
