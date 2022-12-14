package handler_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/webhooks/v6/github"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/warehouse-13/hammertime/pkg/client"
	"github.com/weaveworks-liquidmetal/flintlock/api/services/microvm/v1alpha1"
	"github.com/weaveworks-liquidmetal/flintlock/api/types"
	"google.golang.org/protobuf/types/known/emptypb"
	"k8s.io/utils/pointer"

	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/config"
	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/handler"
	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/handler/fakes"
	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/microvm"
)

func TestNew_WithoutClientFuncShouldError(t *testing.T) {
	g := NewWithT(t)
	cfg := newTestConfig()
	p := handler.Params{
		Config: cfg,
	}
	_, err := handler.New(p)
	g.Expect(err).To(MatchError("func to generate FlintlockClient not provided"))
}

func TestNew_WithoutLoggerShouldError(t *testing.T) {
	g := NewWithT(t)
	cfg := newTestConfig()
	p := handler.Params{
		Config: cfg,
		Client: func(string) (client.FlintlockClient, error) { return &fakes.FakeFlintlockClient{}, nil },
	}
	_, err := handler.New(p)
	g.Expect(err).To(MatchError("logger not provided"))
}

func TestNew_WithoutPayloadShouldError(t *testing.T) {
	g := NewWithT(t)
	cfg := newTestConfig()
	p := handler.Params{
		Config: cfg,
		Client: func(string) (client.FlintlockClient, error) { return &fakes.FakeFlintlockClient{}, nil },
		L:      nullLogger(),
	}
	_, err := handler.New(p)
	g.Expect(err).To(MatchError("payload interface not fulfilled"))
}

func TestHandleWebhookPost(t *testing.T) {
	g := NewWithT(t)

	var (
		cfg = newTestConfig()

		queued          = "queued"
		completed       = "completed"
		nodeId          = "foo"
		runId     int64 = 1234
	)

	tt := []struct {
		name           string
		event          string
		clientFn       func(client.FlintlockClient) handler.ClientFunc
		fakesReturn    func(*fakes.FakePayload, *fakes.FakeFlintlockClient)
		expected       func(*fakes.FakePayload, *fakes.FakeFlintlockClient)
		expectedStatus int
	}{
		{
			name:     "payload service parse fails, processing any event fails",
			clientFn: newFakeClient,
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(nil, errors.New("fail"))
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))
				g.Expect(flClient.CreateCallCount()).To(Equal(0))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:     "payload service returns unknown payload, processing any event stops but does not fail",
			clientFn: newFakeClient,
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(fakeEvent("random", nodeId, runId), nil)
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))
				g.Expect(flClient.CreateCallCount()).To(Equal(0))
				g.Expect(flClient.ListCallCount()).To(Equal(0))
				g.Expect(flClient.DeleteCallCount()).To(Equal(0))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "payload service returns unknown event, processing any event stops but does not fail",
			clientFn: newFakeClient,
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(nil, nil)
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))
				g.Expect(flClient.CreateCallCount()).To(Equal(0))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:     "client func fails, processing queued event fails",
			event:    queued,
			clientFn: newBadClient,
			fakesReturn: func(payloadService *fakes.FakePayload, _ *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(fakeEvent(queued, nodeId, runId), nil)
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))
				g.Expect(flClient.ListCallCount()).To(Equal(0))
				g.Expect(flClient.DeleteCallCount()).To(Equal(0))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:     "client func fails, processing completed event fails",
			event:    completed,
			clientFn: newBadClient,
			fakesReturn: func(payloadService *fakes.FakePayload, _ *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(fakeEvent(completed, nodeId, runId), nil)
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))
				g.Expect(flClient.ListCallCount()).To(Equal(0))
				g.Expect(flClient.DeleteCallCount()).To(Equal(0))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var (
				payloadService = &fakes.FakePayload{}
				flClient       = fakes.FakeFlintlockClient{}
				flClientFn     = tc.clientFn(&flClient)
			)

			p := handler.Params{
				Config:  cfg,
				Client:  flClientFn,
				Payload: payloadService,
				L:       nullLogger(),
			}
			h, err := handler.New(p)
			g.Expect(err).NotTo(HaveOccurred())
			r := httptest.NewRecorder()

			tc.fakesReturn(payloadService, &flClient)

			h.HandleWebhookPost(r, &http.Request{})

			g.Expect(r.Result().StatusCode).To(Equal(tc.expectedStatus))
			tc.expected(payloadService, &flClient)
		})
	}
}

func TestHandleWebhookPost_Queued(t *testing.T) {
	g := NewWithT(t)

	var (
		cfg = newTestConfig()

		queued       = "queued"
		nodeId       = "foo"
		runId  int64 = 1234
		mvmUid       = "foobar"
	)

	tt := []struct {
		name           string
		fakesReturn    func(*fakes.FakePayload, *fakes.FakeFlintlockClient)
		expected       func(*fakes.FakePayload, *fakes.FakeFlintlockClient)
		expectedStatus int
	}{
		{
			name: "payload service parse fails, processing any event fails",
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(nil, errors.New("fail"))
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))
				g.Expect(flClient.CreateCallCount()).To(Equal(0))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "payload service returns unknown payload, processing any event stops but does not fail",
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(nil, nil)
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))
				g.Expect(flClient.CreateCallCount()).To(Equal(0))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "processing queued event succeeds",
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(fakeEvent(queued, nodeId, runId), nil)
				flClient.CreateReturns(fakeMicrovm(mvmUid), nil)
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))
				g.Expect(flClient.CreateCallCount()).To(Equal(1))
				g.Expect(flClient.CreateArgsForCall(0).Id).To(Equal(expectedName(nodeId, runId)))
				g.Expect(flClient.CreateArgsForCall(0).Namespace).To(Equal(microvm.Namespace))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "flintlock client create call fails, processing queued event fails",
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(fakeEvent(queued, nodeId, runId), nil)
				flClient.CreateReturns(nil, errors.New("fail"))
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))
				g.Expect(flClient.CreateCallCount()).To(Equal(1))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		// {
		// 	name:  "client func fails, processing completed event fails",
		// 	fakesReturn: func(event string, payloadService *fakes.FakePayload, _ *fakes.FakeFlintlockClient) {
		// 		payloadService.ParseReturns(fakeEvent(event, nodeId, runId), nil)
		// 	},
		// 	expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
		// 		g.Expect(payloadService.ParseCallCount()).To(Equal(1))
		// 		g.Expect(flClient.ListCallCount()).To(Equal(0))
		// 		g.Expect(flClient.DeleteCallCount()).To(Equal(0))
		// 	},
		// 	expectedStatus: http.StatusInternalServerError,
		// },
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var (
				payloadService = &fakes.FakePayload{}
				flClient       = fakes.FakeFlintlockClient{}
				flClientFn     = newFakeClient(&flClient)
			)

			p := handler.Params{
				Config:  cfg,
				Client:  flClientFn,
				Payload: payloadService,
				L:       nullLogger(),
			}
			h, err := handler.New(p)
			g.Expect(err).NotTo(HaveOccurred())
			r := httptest.NewRecorder()

			tc.fakesReturn(payloadService, &flClient)

			h.HandleWebhookPost(r, &http.Request{})

			g.Expect(r.Result().StatusCode).To(Equal(tc.expectedStatus))
			tc.expected(payloadService, &flClient)
		})
	}
}

func TestHandleWebhookPost_Completed(t *testing.T) {
	g := NewWithT(t)

	var (
		cfg = newTestConfig()

		completed       = "completed"
		nodeId          = "foo"
		runId     int64 = 1234
		mvmUid          = "foobar"
	)

	tt := []struct {
		name           string
		fakesReturn    func(*fakes.FakePayload, *fakes.FakeFlintlockClient)
		expected       func(*fakes.FakePayload, *fakes.FakeFlintlockClient)
		expectedStatus int
	}{
		{
			name: "processing completed event succeeds",
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(fakeEvent(completed, nodeId, runId), nil)
				flClient.ListReturns(fakeMicrovmList(mvmUid), nil)
				flClient.DeleteReturns(&emptypb.Empty{}, nil)
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))

				g.Expect(flClient.ListCallCount()).To(Equal(1))
				name, namespace := flClient.ListArgsForCall(0)
				g.Expect(name).To(Equal(expectedName(nodeId, runId)))
				g.Expect(namespace).To(Equal(microvm.Namespace))

				g.Expect(flClient.DeleteCallCount()).To(Equal(1))
				g.Expect(flClient.DeleteArgsForCall(0)).To(Equal(mvmUid))
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "flintlock client list call fails, processing completed event fails",
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(fakeEvent(completed, nodeId, runId), nil)
				flClient.ListReturns(nil, errors.New("fail"))
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))

				g.Expect(flClient.ListCallCount()).To(Equal(1))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "flintlock client delete call fails, processing completed event fails",
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(fakeEvent(completed, nodeId, runId), nil)
				flClient.ListReturns(fakeMicrovmList(mvmUid), nil)
				flClient.DeleteReturns(&emptypb.Empty{}, errors.New("fail"))
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))

				g.Expect(flClient.ListCallCount()).To(Equal(1))
				g.Expect(flClient.DeleteCallCount()).To(Equal(1))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "flintlock client list call returns no entries, processing completed event stops but does not fail",
			fakesReturn: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				payloadService.ParseReturns(fakeEvent(completed, nodeId, runId), nil)
				flClient.ListReturns(fakeMicrovmList(""), nil)
			},
			expected: func(payloadService *fakes.FakePayload, flClient *fakes.FakeFlintlockClient) {
				g.Expect(payloadService.ParseCallCount()).To(Equal(1))

				g.Expect(flClient.ListCallCount()).To(Equal(1))
				g.Expect(flClient.DeleteCallCount()).To(Equal(0))
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			var (
				payloadService = &fakes.FakePayload{}
				flClient       = fakes.FakeFlintlockClient{}
				flClientFn     = newFakeClient(&flClient)
			)

			p := handler.Params{
				Config:  cfg,
				Client:  flClientFn,
				Payload: payloadService,
				L:       nullLogger(),
			}
			h, err := handler.New(p)
			g.Expect(err).NotTo(HaveOccurred())
			r := httptest.NewRecorder()

			tc.fakesReturn(payloadService, &flClient)

			h.HandleWebhookPost(r, &http.Request{})

			g.Expect(r.Result().StatusCode).To(Equal(tc.expectedStatus))
			tc.expected(payloadService, &flClient)
		})
	}
}

func fakeEvent(action, nodeID string, id int64) *github.WorkflowJobPayload {
	job := github.WorkflowJobPayload{}
	job.Action = action
	job.WorkflowJob.ID = id
	job.WorkflowJob.RunID = id
	job.WorkflowJob.NodeID = nodeID

	return &job
}

func expectedName(strName string, intName int64) string {
	return fmt.Sprintf("%s-%d-%d", strName, intName, intName)
}

func fakeMicrovm(uid string) *v1alpha1.CreateMicroVMResponse {
	return &v1alpha1.CreateMicroVMResponse{
		Microvm: &types.MicroVM{
			Spec: &types.MicroVMSpec{
				Uid: pointer.String(uid),
			},
		},
	}
}

func fakeMicrovmList(uid string) *v1alpha1.ListMicroVMsResponse {
	list := &v1alpha1.ListMicroVMsResponse{}

	if uid != "" {
		list.Microvm = []*types.MicroVM{{
			Spec: &types.MicroVMSpec{
				Uid: pointer.String(uid),
			},
		}}
	}

	return list
}

func newTestConfig() *config.Config {
	return &config.Config{
		Host:          "host",
		APIToken:      "token",
		SSHPublicKey:  "key",
		WebhookSecret: "secret",
	}
}

func nullLogger() *logrus.Entry {
	l := logrus.New()
	l.SetOutput(ioutil.Discard)
	log := logrus.NewEntry(l)
	return log
}

func newFakeClient(c client.FlintlockClient) handler.ClientFunc {
	return func(string) (client.FlintlockClient, error) {
		return c, nil
	}
}

func newBadClient(c client.FlintlockClient) handler.ClientFunc {
	return func(string) (client.FlintlockClient, error) {
		return nil, errors.New("fail")
	}
}
