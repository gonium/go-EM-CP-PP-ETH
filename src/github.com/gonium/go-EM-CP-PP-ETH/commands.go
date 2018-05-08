package EM_CP_PP_ETH

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/goburrow/modbus"
	"net/http"
	"strings"
	"time"
)

type Commander struct {
	modbusClient modbus.Client
}

func NewCommander(client modbus.Client) *Commander {
	return &Commander{
		modbusClient: client,
	}
}

func (c *Commander) HTTPHardReset(host string) error {
	url := fmt.Sprintf("http://%s/config.html?reset=1", host)
	tr := &http.Transport{
		MaxIdleConns:          1,
		IdleConnTimeout:       3 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,

		DisableCompression: true,
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   3 * time.Second,
	}
	req, err := http.NewRequest("GET", url, nil)
	ctx, cancel := context.WithTimeout(context.Background(),
		1*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		// The EM-CP-PP-ETH does not return any response to our call.
		// The client will be canceled. We need to check for this error
		// and suppress it.
		if strings.Contains(err.Error(), "context deadline exceeded") {
			return nil
		} else {
			// This is unexpected, forward the error.
			return err
		}
	}
	defer resp.Body.Close()
	return nil
}

func (c *Commander) ReadChargingEnabled() (result bool, err error) {
	results, err := c.modbusClient.ReadCoils(402, 1)
	if err != nil {
		return false, err
	}
	if results[0] == 0 {
		return false, nil
	}
	return true, nil
}

func (c *Commander) WriteChargingEnabled(newstate bool) (err error) {
	var update uint16
	update = 0x0000
	if newstate {
		update = 0xFF00
	}
	_, err = c.modbusClient.WriteSingleCoil(402, update)
	if err != nil {
		return err
	}
	return nil
}

func (c *Commander) ReadDigimodeEnabled() (result bool, err error) {
	results, err := c.modbusClient.ReadCoils(401, 1)
	if err != nil {
		return false, err
	}
	if results[0] == 0 {
		return false, nil
	}
	return true, nil
}

func (c *Commander) WriteDigimodeEnabled(newstate bool) (err error) {
	var update uint16
	update = 0x0000
	if newstate {
		update = 0xFF00
	}
	_, err = c.modbusClient.WriteSingleCoil(401, update)
	if err != nil {
		return err
	}
	return nil
}

func (c *Commander) ReadActualChargingCurrent() (result uint16, err error) {
	results, err := c.modbusClient.ReadHoldingRegisters(300, 1)
	if err != nil {
		return 0, err
	}
	result = binary.BigEndian.Uint16(results)
	return result, nil
}

func (c *Commander) WriteActualChargingCurrent(current uint16) (result uint16, err error) {
	results, err := c.modbusClient.WriteSingleRegister(300, current)
	if err != nil {
		return 0, err
	}
	result = binary.BigEndian.Uint16(results)
	return result, nil
}
