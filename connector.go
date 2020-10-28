// Copyright (c) 2020 Snowflake Computing Inc. All right reserved.

package gosnowflake

import (
	"context"
	"database/sql/driver"
)

// Driver is the interface for a Snowflake driver
type Driver interface {
	Open(dsn string) (driver.Conn, error)
	OpenWithConfig(ctx context.Context, config Config) (driver.Conn, error)
}

// Connector creates Driver with the specified Config
type Connector struct {
	driver Driver
	cfg    Config
}

// NewConnector creates a new connector with the given SnowflakeDriver and Config.
func NewConnector(driver Driver, config Config) Connector {
	return Connector{driver, config}
}

// Connect creates a new connection.
func (t Connector) Connect(ctx context.Context) (driver.Conn, error) {
	cfg := t.cfg
	err := fillMissingConfigParameters(&cfg)
	if err != nil {
		return nil, err
	}
	return t.driver.OpenWithConfig(ctx, cfg)
}

// Driver creates a new driver.
func (t Connector) Driver() driver.Driver {
	return t.driver
}
