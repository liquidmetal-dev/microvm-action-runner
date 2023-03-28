package microvm_test

import (
	"encoding/base64"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/weaveworks-liquidmetal/flintlock/api/types"
	"github.com/weaveworks-liquidmetal/flintlock/client/cloudinit/userdata"
	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/microvm"
	"gopkg.in/yaml.v2"
)

func Test_MicrovmNew(t *testing.T) {
	g := NewWithT(t)

	var (
		id       = "foo"
		repoName = "action-runner"
		userName = "liquid-metal"
		token    = "token"
		labels   = []string{"foo", "bar"}
	)

	cfg := microvm.UserdataCfg{
		GithubToken: token,
		PublicKey:   "",
		User:        userName,
		Repo:        repoName,
		Id:          id,
		Labels:      labels,
	}

	spec, err := microvm.New(cfg)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(spec.Namespace).To(Equal(microvm.Namespace))
	g.Expect(spec.Id).To(Equal(id))

	userData := decodeData(g, spec)

	g.Expect(userData.Users[0].Name).To(Equal("root"))
	g.Expect(userData.Users[0].SSHAuthorizedKeys).To(BeNil())
	g.Expect(userData.RunCommands[0]).To(ContainSubstring(token))
	g.Expect(userData.RunCommands[0]).To(ContainSubstring(userName))
	g.Expect(userData.RunCommands[0]).To(ContainSubstring(repoName))
	g.Expect(userData.RunCommands[0]).To(ContainSubstring(strings.Join(labels[:], ",")))
}

func Test_MicrovmNew_WithSSHKey(t *testing.T) {
	g := NewWithT(t)

	var (
		name  = "foo"
		token = "token"
		key   = "key"
	)

	cfg := microvm.UserdataCfg{
		GithubToken: token,
		PublicKey:   key,
		User:        "",
		Repo:        "",
		Id:          name,
	}

	spec, err := microvm.New(cfg)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(spec.Namespace).To(Equal(microvm.Namespace))
	g.Expect(spec.Id).To(Equal(name))

	userData := decodeData(g, spec)

	g.Expect(userData.Users[0].SSHAuthorizedKeys).To(ContainElement(key))
}

func decodeData(g *WithT, spec *types.MicroVMSpec) userdata.UserData {
	dat, err := base64.StdEncoding.DecodeString(spec.Metadata["user-data"])
	g.Expect(err).NotTo(HaveOccurred())

	var userData userdata.UserData
	g.Expect(yaml.Unmarshal(dat, &userData)).To(Succeed())

	return userData
}
