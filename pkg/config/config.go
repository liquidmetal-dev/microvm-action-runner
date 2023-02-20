package config

// Config stores the parsed flag opt values for use by the commands
type Config struct {
	// Username is the user or org which owns the repo
	Username string
	// Repository is the name of the repo
	Repository string
	// Hosts is a slice of addresses + ports of any number of flintlock servers
	Hosts []string
	// APIToken is the Github PAT with repo scope
	APIToken string
	// SSHPublicKey is the pub key to add to MicroVMs
	SSHPublicKey string
	// WebhookSecret is a plaintext string for extra auth to the github runner webhook
	WebhookSecret string
}
