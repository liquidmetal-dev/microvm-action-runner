package payload_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-playground/webhooks/v6/github"
	. "github.com/onsi/gomega"

	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/payload"
)

func Test_ParsePayload(t *testing.T) {
	g := NewWithT(t)

	req, err := newRequest()
	g.Expect(err).NotTo(HaveOccurred())

	s := payload.New("")
	p, err := s.Parse(req)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(p).To(BeAssignableToTypeOf(&github.WorkflowJobPayload{}))
}

func Test_ParsePayload_WithSecret(t *testing.T) {
	g := NewWithT(t)

	req, err := newRequest()
	g.Expect(err).NotTo(HaveOccurred())
	req.Header.Set("X-Hub-Signature", "secret")

	s := payload.New("secret")
	_, err = s.Parse(req)
	// I don't want to test that Github's parser does what it is supposed to,
	// I just want to check that we are applying the secret Option. If we see this
	// error then it means the secret is being set. No point in wasting time to
	// make the HMAC verification pass.
	g.Expect(err).To(MatchError(github.ErrHMACVerificationFailed))
}

func newRequest() (*http.Request, error) {
	dat, err := json.Marshal(github.WorkflowJobPayload{})
	if err != nil {
		return nil, err
	}
	postData := bytes.NewBuffer(dat)
	req, err := http.NewRequest("POST", "foobar", postData)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-GitHub-Event", string(github.WorkflowJobEvent))

	return req, nil
}
