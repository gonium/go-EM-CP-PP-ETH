package EM_CP_PP_ETH

import (
	"encoding/binary"
	"fmt"
	"github.com/goburrow/modbus"
	"io"
)

const (
	// Expected length of status array message
	LEN_INPUT_REGISTER_STATUS_BYTES          = 84
	LEN_DISCRETE_INPUT_REGISTER_STATUS_BYTES = 1
)

const (
	DIGITAL_INPUT_EN  = 1 << 0
	DIGITAL_INPUT_XR  = 1 << 1
	DIGITAL_INPUT_LD  = 1 << 2
	DIGITAL_INPUT_ML  = 1 << 3
	DIGITAL_OUTPUT_CR = 1 << 4
	DIGITAL_OUTPUT_LR = 1 << 5
	DIGITAL_OUTPUT_VR = 1 << 6
	DIGITAL_OUTPUT_ER = 1 << 7
)

const (
	ERROR_CABLE_13A_20A     = 1 << 0
	ERROR_CABLE_13A         = 1 << 1
	ERROR_INVALID_PP        = 1 << 2
	ERROR_INVALID_CP        = 1 << 3
	ERROR_STATE_F           = 1 << 4
	ERROR_LOCKING           = 1 << 5
	ERROR_UNLOCKING         = 1 << 6
	ERROR_LD_FAILURE        = 1 << 7
	ERROR_OVERCURRENT       = 1 << 8
	ERROR_COM_MEASUREMENT   = 1 << 9
	ERROR_STATE_D_REJECTED  = 1 << 10
	ERROR_CONTACTOR_FAILURE = 1 << 11
	ERROR_CP_NO_DIODE       = 1 << 12
)

type Status struct {
	EVStatus              string
	ProximityCurrent      uint16
	ChargeTimeMinutes     uint16
	ChargeTimeHours       uint16
	DIPConfiguration      uint16
	FirmwareVersion       uint32
	Errorcode             Errorcode
	L1Voltage             float32
	L2Voltage             float32
	L3Voltage             float32
	L1Current             float32
	L2Current             float32
	L3Current             float32
	ActivePower           float32
	ReactivePower         float32
	ApparentPower         float32
	PowerFactor           float32
	Energy                float32
	MaxPower              float32
	CurrentChargePower    float32
	Frequency             float32
	L1MaxCurrent          float32
	L2MaxCurrent          float32
	L3MaxCurrent          float32
	OverCurrentProtection uint16
	DigitalInputStates    DigiInputs
	DigitalOutputStates   DigiOutputs
	ActualChargingCurrent uint16
}

type DigiInputs struct {
	EN bool
	XR bool
	LD bool
	ML bool
}

type DigiOutputs struct {
	CR bool
	LR bool
	VR bool
	ER bool
}

type Errorcode struct {
	OK                    bool
	Cable13A_20A          bool
	Cable13A              bool
	InvalidPP             bool
	InvalidCP             bool
	StateF                bool
	Locking               bool
	Unlocking             bool
	FailureLD             bool
	Overcurrent           bool
	ComMeasurementFailure bool
	RejectedStateD        bool
	ContactorFailure      bool
	CPNoDiode             bool
}

type StatusCache struct {
	modbusClient modbus.Client
	Status       Status
}

func NewStatusCache(client modbus.Client) *StatusCache {
	return &StatusCache{
		modbusClient: client,
		Status:       Status{},
	}
}

// Refreshes the cache with the current settings/values of the charge
// controller. This is a multi-stage process because of the way the
// modbus interface works. By calling Refresh() you trigger various
// steps to read the status and fill the cache.
func (sc *StatusCache) Refresh() (err error) {

	// 1. Parse Input Registers.
	results, err := sc.readInputRegisterStatus()
	if err != nil {
		return err
	}
	err = sc.parseInputRegisterStatus(results)
	if err != nil {
		return fmt.Errorf("Failed to parse input register status results: %s", err.Error())
	}

	// 2. Parse Discrete Registers.
	results, err = sc.readDiscreteInputStatus()
	if err != nil {
		return err
	}
	err = sc.parseDiscreteInputStatus(results)
	if err != nil {
		return fmt.Errorf("Failed to parse discrete register status results: %s", err.Error())
	}

	//	// 3. Add the actual charging current
	//	result, err := sc.ReadActualChargingCurrent()
	//	if err != nil {
	//		return err
	//	}
	//	sc.Status.ActualChargingCurrent = result

	return nil
}

func (sc *StatusCache) readDiscreteInputStatus() (results []byte, err error) {
	results, err = sc.modbusClient.ReadDiscreteInputs(200, 8)
	if err != nil {
		return results, fmt.Errorf("Modbus com error: %s", err.Error())
	}
	return results, nil
}

func checkByteMaskAndSet(state byte, mask byte) bool {
	if (state & mask) != 0 {
		return true
	} else {
		return false
	}
}

func checkUint16MaskAndSet(state uint16, mask uint16) bool {
	if (state & mask) != 0 {
		return true
	} else {
		return false
	}
}

func (sc *StatusCache) parseDiscreteInputStatus(input []byte) (err error) {

	if len(input) != LEN_DISCRETE_INPUT_REGISTER_STATUS_BYTES {
		return fmt.Errorf(
			"Invalid length of status byte array - expected %d, got %d",
			LEN_DISCRETE_INPUT_REGISTER_STATUS_BYTES, len(input),
		)
	}

	state := input[0]
	sc.Status.DigitalInputStates.EN = checkByteMaskAndSet(state, DIGITAL_INPUT_EN)
	sc.Status.DigitalInputStates.XR = checkByteMaskAndSet(state, DIGITAL_INPUT_XR)
	sc.Status.DigitalInputStates.LD = checkByteMaskAndSet(state, DIGITAL_INPUT_LD)
	sc.Status.DigitalInputStates.ML = checkByteMaskAndSet(state, DIGITAL_INPUT_ML)
	sc.Status.DigitalOutputStates.CR = checkByteMaskAndSet(state, DIGITAL_OUTPUT_CR)
	sc.Status.DigitalOutputStates.LR = checkByteMaskAndSet(state, DIGITAL_OUTPUT_LR)
	sc.Status.DigitalOutputStates.VR = checkByteMaskAndSet(state, DIGITAL_OUTPUT_VR)
	sc.Status.DigitalOutputStates.ER = checkByteMaskAndSet(state, DIGITAL_OUTPUT_ER)

	return nil
}

func (sc *StatusCache) readInputRegisterStatus() (results []byte, err error) {
	results, err = sc.modbusClient.ReadInputRegisters(100, 42)
	if err != nil {
		return results, fmt.Errorf("Modbus com error: %s", err.Error())
	}
	return results, nil
}

func (sc *StatusCache) parseInputRegisterStatus(input []byte) (err error) {
	if len(input) != LEN_INPUT_REGISTER_STATUS_BYTES {
		return fmt.Errorf(
			"Invalid length of status byte array - expected %d, got %d",
			LEN_INPUT_REGISTER_STATUS_BYTES, len(input),
		)
	}

	// Extract vehicle status
	vehiclestate := binary.BigEndian.Uint16(input[0:2])
	switch vehiclestate {
	case 65:
		sc.Status.EVStatus = "A"
	case 66:
		sc.Status.EVStatus = "B"
	case 67:
		sc.Status.EVStatus = "C"
	case 68:
		sc.Status.EVStatus = "D"
	case 69:
		sc.Status.EVStatus = "E"
	case 70:
		sc.Status.EVStatus = "F"
	default:
		return fmt.Errorf("Invalid vehicle state '%d'", vehiclestate)
	}

	sc.Status.ProximityCurrent = binary.BigEndian.Uint16(input[2:4])
	// This is strange. See the web interface for the "true" value.
	sc.Status.ChargeTimeMinutes = binary.BigEndian.Uint16(input[4:6]) / 60
	sc.Status.ChargeTimeHours = binary.BigEndian.Uint16(input[6:8])
	sc.Status.DIPConfiguration = binary.BigEndian.Uint16(input[8:10])
	sc.Status.FirmwareVersion = binary.BigEndian.Uint32(input[10:14])

	state := binary.BigEndian.Uint16(input[14:16])
	if state == 0 {
		sc.Status.Errorcode.OK = true
	} else {
		sc.Status.Errorcode.OK = false
	}
	sc.Status.Errorcode.Cable13A_20A = checkUint16MaskAndSet(state, ERROR_CABLE_13A_20A)
	sc.Status.Errorcode.Cable13A = checkUint16MaskAndSet(state, ERROR_CABLE_13A)
	sc.Status.Errorcode.InvalidPP = checkUint16MaskAndSet(state, ERROR_INVALID_PP)
	sc.Status.Errorcode.InvalidCP = checkUint16MaskAndSet(state, ERROR_INVALID_CP)
	sc.Status.Errorcode.StateF = checkUint16MaskAndSet(state, ERROR_STATE_F)
	sc.Status.Errorcode.Locking = checkUint16MaskAndSet(state, ERROR_LOCKING)
	sc.Status.Errorcode.Unlocking = checkUint16MaskAndSet(state, ERROR_UNLOCKING)
	sc.Status.Errorcode.FailureLD = checkUint16MaskAndSet(state, ERROR_LD_FAILURE)
	sc.Status.Errorcode.Overcurrent = checkUint16MaskAndSet(state, ERROR_OVERCURRENT)
	sc.Status.Errorcode.ComMeasurementFailure = checkUint16MaskAndSet(state, ERROR_COM_MEASUREMENT)
	sc.Status.Errorcode.RejectedStateD = checkUint16MaskAndSet(state, ERROR_STATE_D_REJECTED)
	sc.Status.Errorcode.ContactorFailure = checkUint16MaskAndSet(state, ERROR_CONTACTOR_FAILURE)
	sc.Status.Errorcode.CPNoDiode = checkUint16MaskAndSet(state,
		ERROR_CP_NO_DIODE)

	sc.Status.L1Voltage =
		float32(binary.BigEndian.Uint32(sc.swapWords(input[16:20]))) / 100
	sc.Status.L2Voltage = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[20:24]))) / 100
	sc.Status.L3Voltage = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[24:28]))) / 100
	sc.Status.L1Current = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[28:32]))) / 1000
	sc.Status.L2Current = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[32:36]))) / 1000
	sc.Status.L3Current = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[36:40]))) / 1000
	sc.Status.ActivePower = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[40:44]))) * 10
	sc.Status.ReactivePower = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[44:48])))
	sc.Status.ApparentPower = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[48:52]))) * 10
	sc.Status.PowerFactor = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[52:56]))) / 1000
	sc.Status.Energy =
		float32(binary.BigEndian.Uint32(sc.swapWords(input[56:60]))) / 100
	sc.Status.MaxPower = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[60:64]))) * 10
	sc.Status.CurrentChargePower = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[64:68])))
	sc.Status.Frequency = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[68:72]))) / 100
	sc.Status.L1MaxCurrent = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[72:76])))
	sc.Status.L2MaxCurrent = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[76:80])))
	sc.Status.L3MaxCurrent = float32(
		binary.BigEndian.Uint32(sc.swapWords(input[80:84])))
	sc.Status.OverCurrentProtection = binary.BigEndian.Uint16(input[84:86])

	return nil
}

func (sc StatusCache) swapWords(data []byte) []byte {
	// TODO: Proper error handling needed.
	if len(data) != 4 {
		return data
	}

	data_r := make([]byte, 0)
	data_r = append(data_r, data[2:4]...)
	data_r = append(data_r, data[0:2]...)

	return data_r
}

func (sc StatusCache) WriteFormattedStatus(out io.Writer) {
	fmt.Fprintf(out, "EV status: %s\n", sc.Status.EVStatus)
	fmt.Fprintf(out, "Proximity Current: %d A\n", sc.Status.ProximityCurrent)
	fmt.Fprintf(out, "Charge time: %d:%d\n", sc.Status.ChargeTimeHours,
		sc.Status.ChargeTimeMinutes)
	fmt.Fprintf(out, "DIP configuration: %d\n", sc.Status.DIPConfiguration)
	fmt.Fprintf(out, "Firmware Version: %d\n", sc.Status.FirmwareVersion)
	fmt.Fprintf(out, "Error state: %+v\n", sc.Status.Errorcode)
	fmt.Fprintf(out, "Voltage [V]: L1 %.2f L2 %.2f, L3 %.2f\n", sc.Status.L1Voltage,
		sc.Status.L2Voltage, sc.Status.L3Voltage)
	fmt.Fprintf(out, "Current [A]: L1 %.2f, L2 %.2f, L3 %.2f\n", sc.Status.L1Current,
		sc.Status.L2Current, sc.Status.L3Current)
	fmt.Fprintf(out, "Active power [W]: %.2f\n", sc.Status.ActivePower)
	fmt.Fprintf(out, "Reactive power [W]: %.2f\n", sc.Status.ReactivePower)
	fmt.Fprintf(out, "Apparent power [VA]: %.2f\n", sc.Status.ApparentPower)
	fmt.Fprintf(out, "Power factor: %.2f\n", sc.Status.PowerFactor)
	fmt.Fprintf(out, "Energy [kWh]: %.2f\n", sc.Status.Energy)
	fmt.Fprintf(out, "Max Power (charge sequence) [W]: %.2f\n", sc.Status.MaxPower)
	fmt.Fprintf(out, "Energy (charge sequence) [kWh]: %.2f\n", sc.Status.CurrentChargePower)
	fmt.Fprintf(out, "Frequency [Hz]: %.2f\n", sc.Status.Frequency)
	fmt.Fprintf(out, "Max Current [A]: L1 %.2f, L2 %.2f, L3 %.2f\n", sc.Status.L1MaxCurrent,
		sc.Status.L2MaxCurrent, sc.Status.L3MaxCurrent)
	fmt.Fprintf(out, "Overcurrent protection: %d\n", sc.Status.OverCurrentProtection)
	fmt.Fprintf(out, "Digital inputs: %+v\n",
		sc.Status.DigitalInputStates)
	fmt.Fprintf(out, "Digital outputs: %+v\n",
		sc.Status.DigitalOutputStates)
}
