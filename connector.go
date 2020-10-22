// Copyright (c) 2020 Snowflake Computing Inc. All right reserved.

package gosnowflake

import (
	"context"
	"database/sql/driver"
)

// SnowflakeConnector creates SnowflakeDriver with the specified Config
type SnowflakeConnector struct {
	driver SnowflakeDriverInterface
	cfg    Config
}

// NewSnowflakeConnector creates a new connector with the given SnowflakeDriver and Config.
func NewSnowflakeConnector(driver *SnowflakeDriverInterface, config *Config) SnowflakeConnector {
	return SnowflakeConnector{*driver, *config}
}

// Connect creates a new connection.
func (t SnowflakeConnector) Connect(ctx context.Context) (driver.Conn, error) {
	cfg := t.cfg
	err := fillMissingConfigParameters(&cfg)
	if err != nil {
		return nil, err
	}
	return t.driver.OpenWithConfig(ctx, cfg)
}

// Driver creates a new driver.
func (t SnowflakeConnector) Driver() driver.Driver {
	return t.driver
}
