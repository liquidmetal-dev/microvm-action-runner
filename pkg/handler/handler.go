package handler

import (
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

type Handler struct {
	flHost        string
	githubPAT     string
	publicKey     string
	webhookSecret string
	payload       payload.Service
	l             *logrus.Entry
}

// TODO refactor input params into opts or something
// New returns a new Handler
func New(log *logrus.Entry, c *config.Config) Handler {
	return Handler{
		flHost:        c.Host,
		githubPAT:     c.APIToken,
		publicKey:     c.SSHPublicKey,
		webhookSecret: c.WebhookSecret,
		payload:       payload.New(c.WebhookSecret),
		l:             log,
	}
}

func (h Handler) HandleWebhookPost(w http.ResponseWriter, r *http.Request) {
	h.l.Debug("webhook received")

	event, err := h.payload.Parse(r)
	if err != nil {
		h.l.Errorf("failed to parse webhook payload: %s", err)
	}

	if event == nil {
		h.l.Debugf("payload type is unknown")
	}

	h.l.Debugf("workflow event found %s", event.WorkflowJob.RunURL)

	switch event.Action {
	case eventQueued:
		h.processQueuedAction(*event)
	case eventCompleted:
		h.processCompletedAction(*event)
	default:
		h.l.Debugf("event type is unknown: %s", event.Action)
	}
}

func (h Handler) processQueuedAction(p github.WorkflowJobPayload) {
	h.l.Infof("proccessing queued action for workflow (job-id: %d) (step-id: %d)", p.WorkflowJob.RunID, p.WorkflowJob.ID)

	fl, err := client.New(h.flHost, "")
	if err != nil {
		h.l.Errorf("failed to create flintlock client: %s", err)
	}

	defer func() {
		if err := fl.Close(); err != nil {
			h.l.Errorf("failed to close connection to flintlock host to %s: %s", h.flHost, err)
		}
	}()

	name := generateName(p)
	mvm, err := microvm.New(h.githubPAT, h.publicKey, name)
	if err != nil {
		h.l.Errorf("failed to generate microvm spec: %s", err)
		return
	}

	h.l.Debugf("creating microvm %s", name)
	created, err := fl.Create(mvm)
	if err != nil {
		h.l.Errorf("failed to create microvm: %s", err)
		return
	}

	h.l.Infof("created microvm, name: %s, uid: %s", name, *created.Microvm.Spec.Uid)
}

func (h Handler) processCompletedAction(p github.WorkflowJobPayload) {
	h.l.Infof("proccessing complete action for workflow (job-id: %d) (step-id: %d)", p.WorkflowJob.RunID, p.WorkflowJob.ID)

	fl, err := client.New(h.flHost, "")
	if err != nil {
		h.l.Errorf("failed to create flintlock client %s", err)
	}

	defer func() {
		if err := fl.Close(); err != nil {
			h.l.Errorf("failed to close connection to flintlock host %s: %s", "address", err)
		}
	}()

	name := generateName(p)
	h.l.Debugf("looking up microvm for action %s/%s", microvm.Namespace, name)
	resp, err := fl.List(name, microvm.Namespace)
	if err != nil {
		h.l.Errorf("failed to list microvms %s", err)
		return
	}

	if len(resp.Microvm) == 0 {
		h.l.Debugf("no microvms found in %s/%s", microvm.Namespace, name)
		return
	}

	// TODO this is only safe if I am totally sure the name is unique...
	uid := resp.Microvm[0].Spec.Uid

	h.l.Debugf("deleting microvm, name: %s, uid: %s", name, *uid)
	_, err = fl.Delete(*uid)
	if err != nil {
		h.l.Errorf("failed to delete microvm: %s", err)
		return
	}

	h.l.Infof("deleted microvm, name: %s, uid: %s", name, *uid)
}

func generateName(p github.WorkflowJobPayload) string {
	return fmt.Sprintf("%s-%d-%d", p.WorkflowJob.NodeID, p.WorkflowJob.ID, p.WorkflowJob.RunID)
}
