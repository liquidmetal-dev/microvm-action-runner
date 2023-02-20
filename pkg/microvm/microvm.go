package microvm

import (
	"embed"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/warehouse-13/hammertime/pkg/defaults"
	"github.com/weaveworks-liquidmetal/flintlock/api/types"
	"github.com/weaveworks-liquidmetal/flintlock/client/cloudinit/instance"
	"github.com/weaveworks-liquidmetal/flintlock/client/cloudinit/userdata"
	"gopkg.in/yaml.v2"
)

const (
	Namespace      = "self-hosted"
	userdataScript = "userdata.sh"
)

func New(ghToken, publicKey, user, repo, id string) (*types.MicroVMSpec, error) {
	mvm := defaults.BaseMicroVM()
	mvm.Id = id
	mvm.Namespace = Namespace

	metadata, err := createMetadata(id, Namespace)
	if err != nil {
		return nil, err
	}

	userdata, err := createUserData(id, user, repo, ghToken, publicKey)
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

func createUserData(id, ghToken, user, repo, publicKey string) (string, error) {
	dat, err := embeddedScript.ReadFile(userdataScript)
	if err != nil {
		return "", err
	}

	script := string(dat)

	script = strings.Replace(script, "REPLACE_PAT", ghToken, 1)
	script = strings.Replace(script, "REPLACE_ID", id, 1)
	script = strings.Replace(script, "REPLACE_ORG_USER", user, 1)
	script = strings.Replace(script, "REPLACE_REPO", repo, 1)

	userData := &userdata.UserData{
		HostName: id,
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

	if publicKey != "" {
		userData.Users[0].SSHAuthorizedKeys = []string{publicKey}
	}

	data, err := yaml.Marshal(userData)
	if err != nil {
		return "", fmt.Errorf("marshalling bootstrap data: %w", err)
	}

	dataWithHeader := append([]byte("#cloud-config\n"), data...)

	return base64.StdEncoding.EncodeToString(dataWithHeader), nil
}
