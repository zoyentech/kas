package kas

import (
	"context"
	"errors"
	"io"
)

// errNotImplemented 是 S1 占位实现的哨兵；各 Stage 落地后本文件整体删除。
var errNotImplemented = errors.New("kas: not implemented in current stage")

func newClient(cfg Config, deps Dependencies, opts ...Option) (*Client, error) {
	return nil, errNotImplemented // TODO(S3): Story 3.3.2.1
}

func (c *Client) answer(ctx context.Context, req Request) (Response, error) {
	return Response{}, errNotImplemented // TODO(S3): Story 3.3.1.1
}

func (c *Client) health(ctx context.Context) HealthReport {
	return HealthReport{} // TODO(S6): Story 6.1.1.1
}

func (c *Client) reload(ctx context.Context, r io.Reader) (string, error) {
	return "", errNotImplemented // TODO(S2): Story 2.2.2.2
}

func (c *Client) close() error {
	return nil // TODO(S3): Story 3.3.2.1
}

func loadConfig(r io.Reader) (Config, error)     { return Config{}, errNotImplemented } // TODO(S2): Story 2.1.1.2
func loadConfigFile(path string) (Config, error) { return Config{}, errNotImplemented } // TODO(S2): Story 2.1.1.2
func validateConfig(cfg Config) error            { return errNotImplemented }           // TODO(S2): Feature 2.1.2
