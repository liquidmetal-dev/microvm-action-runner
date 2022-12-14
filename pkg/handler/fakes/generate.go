package fakes

// Run go generate to regenerate these mocks.
//go:generate ../../../bin/counterfeiter -o fake_client.go github.com/warehouse-13/hammertime/pkg/client.FlintlockClient
//go:generate ../../../bin/counterfeiter -o fake_payload.go github.com/weaveworks-liquidmetal/microvm-action-runner/pkg/payload.Payload
