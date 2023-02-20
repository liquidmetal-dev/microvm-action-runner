#!/bin/bash

USER=ubuntu
SCRIPT="/home/$USER/register.sh"
WORK_DIR="/home/$USER/actions-runner"
RUNNER_VERSION=2.294.0
TAR_NAME="actions-runner-linux-x64-$RUNNER_VERSION.tar.gz"
ORG="REPLACE_ORG_USER"
REPO="REPLACE_REPO"
REPO_URL="https://github.com/$ORG/$REPO"
RUNNER_NAME="REPLACE_ID"

get_token() {
	curl \
		-X POST \
		-H "Accept: application/vnd.github+json" \
		-H "Authorization: token REPLACE_PAT" \
		"https://api.github.com/repos/$ORG/$REPO/actions/runners/registration-token" | \
		jq -r .token
}
TOKEN=$(get_token)

# create ubuntu user, no password
adduser --disabled-password --gecos "" "$USER"
usermod -aG sudo "$USER"
passwd -d "$USER"

# write a script that a non-root user can call
cat <<EOF > "$SCRIPT"
#!/bin/bash

# get registration token

sudo apt update
sudo apt install -y jq

# create work dir
sudo mkdir -p "$WORK_DIR"
cd "$WORK_DIR" || true
sudo chown "$USER:$USER" "$WORK_DIR"

# download runner
curl -o "$TAR_NAME" -L "https://github.com/actions/runner/releases/download/v$RUNNER_VERSION/$TAR_NAME"
tar xzf "$TAR_NAME"

# register with github
./config.sh --name "$RUNNER_NAME" --url "$REPO_URL" --token "$TOKEN" --unattended --ephemeral

# start service
sudo ./svc.sh install
sudo ./svc.sh start

# victory dance
echo "MicroVM has been registered as self hosted runner"
sudo touch "$HOME/registration_complete"
EOF

chmod +x "$SCRIPT"
chown "$USER:$USER" "$SCRIPT"

# run the script as user
su "$USER" "$SCRIPT"
