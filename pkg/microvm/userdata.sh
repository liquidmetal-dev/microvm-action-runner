#!/bin/bash

USER=ubuntu
SCRIPT="/home/$USER/register.sh"
WORK_DIR="/home/$USER/actions-runner"
ORG="REPLACE_ORG_USER"
REPO="REPLACE_REPO"
REPO_URL="https://github.com/$ORG/$REPO"
RUNNER_NAME="REPLACE_ID"
LABELS="REPLACE_LABELS"

get_token() {
	curl \
		-X POST \
		-H "Accept: application/vnd.github+json" \
		-H "Authorization: token REPLACE_PAT" \
		"https://api.github.com/repos/$ORG/$REPO/actions/runners/registration-token" | \
		jq -r .token
}
TOKEN=$(get_token)

# write a script that the ubuntu user can call
cat <<EOF > "$SCRIPT"
#!/bin/bash

cd "$WORK_DIR" || true

# register with github
./config.sh \
	--name "$RUNNER_NAME" \
	--url "$REPO_URL" \
	--token "$TOKEN" \
	--labels "$LABELS" \
	--unattended \
	--ephemeral \
	--disableupdate

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
