package microvm

import (
	"embed"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/weaveworks-liquidmetal/flintlock/api/types"
	"github.com/weaveworks-liquidmetal/flintlock/client/cloudinit/instance"
	"github.com/weaveworks-liquidmetal/flintlock/client/cloudinit/userdata"
	"gopkg.in/yaml.v2"
)

const (
	Namespace      = "self-hosted"
	userdataScript = "userdata.sh"
)

type UserdataCfg struct {
	GithubToken string
	PublicKey   string
	User        string
	Repo        string
	Id          string
	Labels      []string
}

func New(cfg UserdataCfg) (*types.MicroVMSpec, error) {
	mvm := defaultMicroVM()
	mvm.Id = cfg.Id
	mvm.Namespace = Namespace

	metadata, err := createMetadata(cfg.Id, Namespace)
	if err != nil {
		return nil, err
	}

	userdata, err := createUserData(cfg)
	if err != nil {
		return nil, err
	}

	mvm.Metadata = map[string]string{
		"meta-data": metadata,
		"user-data": userdata,
	}

	return mvm, nil
}

func createMetadata(name, ns string) (string, error) {
	metadata := instance.New(
		instance.WithInstanceID(fmt.Sprintf("%s/%s", ns, name)),
		instance.WithLocalHostname(name),
		instance.WithPlatform("liquid_metal"),
	)

	userMeta, err := yaml.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("unable to marshal metadata: %w", err)
	}

	return base64.StdEncoding.EncodeToString(userMeta), nil
}

//go:embed userdata.sh
var embeddedScript embed.FS

func createUserData(cfg UserdataCfg) (string, error) {
	dat, err := embeddedScript.ReadFile(userdataScript)
	if err != nil {
		return "", err
	}

	script := string(dat)

	script = strings.Replace(script, "REPLACE_PAT", cfg.GithubToken, 1)
	script = strings.Replace(script, "REPLACE_ID", cfg.Id, 1)
	script = strings.Replace(script, "REPLACE_ORG_USER", cfg.User, 1)
	script = strings.Replace(script, "REPLACE_REPO", cfg.Repo, 1)
	script = strings.Replace(script, "REPLACE_LABELS", formatLabels(cfg.Labels), 1)

	userData := &userdata.UserData{
		HostName: cfg.Id,
		Users: []userdata.User{{
			Name: "root",
		}},
		FinalMessage: "The Liquid Metal booted system is good to go after $UPTIME seconds",
		BootCommands: []string{
			"ln -sf /run/systemd/resolve/stub-resolv.conf /etc/resolv.conf",
		},
		RunCommands: []string{
			script,
		},
	}

	if cfg.PublicKey != "" {
		userData.Users[0].SSHAuthorizedKeys = []string{cfg.PublicKey}
	}

	data, err := yaml.Marshal(userData)
	if err != nil {
		return "", fmt.Errorf("marshalling bootstrap data: %w", err)
	}

	dataWithHeader := append([]byte("#cloud-config\n"), data...)

	return base64.StdEncoding.EncodeToString(dataWithHeader), nil
}

func formatLabels(labels []string) string {
	return strings.Join(labels[:], ",")
}
