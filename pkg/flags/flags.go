package flags

import (
	"github.com/urfave/cli/v2"
	"github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/config"
)

// WithFlagsFunc can be used with CLIFlags to build a list of flags for a
// command.
type WithFlagsFunc func() []cli.Flag

// CLIFlags takes a list of WithFlagsFunc options and returns a list of flags
// for a command.
func CLIFlags(options ...WithFlagsFunc) []cli.Flag {
	flags := []cli.Flag{}

	for _, group := range options {
		flags = append(flags, group()...)
	}

	return flags
}

const (
	userFlag   = "user"
	repoFlag   = "repo"
	hostsFlag  = "hosts"
	tokenFlag  = "token"
	secretFlag = "secret"
	keyFlag    = "key"
)

// WithRepoFlags adds the github user and repo flags to the command.
func WithRepoFlags() WithFlagsFunc {
	return func() []cli.Flag {
		return []cli.Flag{
			&cli.StringFlag{
				Name:     userFlag,
				Aliases:  []string{"u"},
				Usage:    "the github username or org for the repo",
				Required: true,
			},
			&cli.StringFlag{
				Name:     repoFlag,
				Aliases:  []string{"r"},
				Usage:    "the github repo name",
				Required: true,
			},
		}
	}
}

// WithHostFlag adds the flintlock GRPC address flag to the command.
func WithHostsFlag() WithFlagsFunc {
	return func() []cli.Flag {
		return []cli.Flag{
			&cli.StringSliceFlag{
				Name:     hostsFlag,
				Aliases:  []string{"host"},
				Usage:    "a list of flintlock server addresses (eg. 1.2.3.4:9090)",
				Required: true,
			},
		}
	}
}

// WithAPITokenFlag adds the github APIToken flag to the command.
func WithAPITokenFlag() WithFlagsFunc {
	return func() []cli.Flag {
		return []cli.Flag{
			&cli.StringFlag{
				Name:     tokenFlag,
				Aliases:  []string{"t"},
				Usage:    "github API token with repo scope",
				Required: true,
			},
		}
	}
}

// WithWebhookSecretFlag adds the webhook secrect flag to the command.
func WithWebhookSecretFlag() WithFlagsFunc {
	return func() []cli.Flag {
		return []cli.Flag{
			&cli.StringFlag{
				Name:     secretFlag,
				Aliases:  []string{"s"},
				Usage:    "the plaintext secret set for the webhook",
				Required: false,
			},
		}
	}
}

// WithSSHPublicKeyFlag adds the SSH key flag to the command.
func WithSSHPublicKeyFlag() WithFlagsFunc {
	return func() []cli.Flag {
		return []cli.Flag{
			&cli.StringFlag{
				Name:     keyFlag,
				Aliases:  []string{"k"},
				Usage:    "public ssh key for microvm access",
				Required: false,
			},
		}
	}
}

// ParseFlags processes all flags on the CLI context and builds a config object
// which will be used in the command's action.
func ParseFlags(cfg *config.Config) cli.BeforeFunc {
	return func(ctx *cli.Context) error {
		cfg.Repository = ctx.String(repoFlag)
		cfg.Username = ctx.String(userFlag)
		cfg.Hosts = ctx.StringSlice(hostsFlag)
		cfg.APIToken = ctx.String(tokenFlag)
		cfg.WebhookSecret = ctx.String(secretFlag)
		cfg.SSHPublicKey = ctx.String(keyFlag)

		return nil
	}
}
