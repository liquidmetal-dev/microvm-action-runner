package payload

import (
	"errors"
	"net/http"

	"github.com/go-playground/webhooks/v6/github"
)

type Payload interface {
	Parse(r *http.Request) (*github.WorkflowJobPayload, error)
}

type Service struct {
	secret string
}

func New(s string) *Service {
	return &Service{s}
}

func (s Service) Parse(r *http.Request) (*github.WorkflowJobPayload, error) {
	var opt = noOpt()

	if s.secret != "" {
		opt = github.Options.Secret(s.secret)
	}

	hook, err := github.New(opt)
	if err != nil {
		return nil, err
	}

	payload, err := hook.Parse(r, github.WorkflowJobEvent)
	if err != nil {
		return nil, err
	}

	p, ok := payload.(github.WorkflowJobPayload)
	if !ok {
		return nil, errors.New("could not parse WorkflowJobPayload")
	}

	return &p, nil
}

func noOpt() github.Option {
	return func(hook *github.Webhook) error {
		return nil
	}
}
