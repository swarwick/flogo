package wits0

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"github.com/TIBCOSoftware/flogo-lib/core/action"
	"github.com/TIBCOSoftware/flogo-lib/core/trigger"
	"github.com/TIBCOSoftware/flogo-lib/logger"
	"github.com/tarm/serial"
)

// log is the default package logger
var log = logger.GetLogger("trigger-wits0")

// wits0TriggerFactory My wits0Trigger factory
type wits0TriggerFactory struct {
	metadata *trigger.Metadata
}

// NewFactory create a new wits0Trigger factory
func NewFactory(md *trigger.Metadata) trigger.Factory {
	return &wits0TriggerFactory{metadata: md}
}

// New Creates a new trigger instance for a given id
func (t *wits0TriggerFactory) New(config *trigger.Config) trigger.Trigger {
	return &wits0Trigger{metadata: t.metadata, config: config}
}

// wits0Trigger is the wits0Trigger implementation
type wits0Trigger struct {
	metadata  *trigger.Metadata
	runner    action.Runner
	config    *trigger.Config
	handlers  []*trigger.Handler
	stopCheck chan bool
}

func (t *wits0Trigger) Initialize(ctx trigger.InitContext) error {
	log.Debug("Initialize")
	t.handlers = ctx.GetHandlers()
	return nil
}

func (t *wits0Trigger) Metadata() *trigger.Metadata {
	return t.metadata
}

// GetSettingSafe get a setting and returns default if not found
func GetSettingSafe(endpoint *trigger.Handler, setting string, defaultValue string) string {
	var retString string
	defer func() {
		if r := recover(); r != nil {
			retString = defaultValue
		}
	}()
	retString = endpoint.GetStringSetting(setting)
	return retString
}

// GetSafeNumber gets the number from the config checking for empty and using default
func GetSafeNumber(endpoint *trigger.Handler, setting string, defaultValue int) int {
	if settingString := GetSettingSafe(endpoint, setting, ""); settingString != "" {
		value, _ := strconv.Atoi(settingString)
		return value
	}
	return defaultValue
}

// GetSafeBool gets the bool from the config checking for empty and using default
func GetSafeBool(endpoint *trigger.Handler, setting string, defaultValue bool) bool {
	if settingString := GetSettingSafe(endpoint, setting, ""); settingString != "" {
		value, _ := strconv.ParseBool(settingString)
		return value
	}
	return defaultValue
}

// wits0Packet the WITS0 packet structure
type wits0Packet struct {
	Records []wits0Record
}

// wits0Record the WITS0 record structure
type wits0Record struct {
	Record string
	Item   string
	Data   string
}

// Start implements trigger.wits0Trigger.Start
func (t *wits0Trigger) Start() error {
	log.Debug("Start")
	t.stopCheck = make(chan bool)
	handlers := t.handlers
	for _, handler := range handlers {
		t.connectToSerial(handler)
	}
	return nil
}

func (t *wits0Trigger) connectToSerial(endpoint *trigger.Handler) {

	config := &serial.Config{
		Name:        GetSettingSafe(endpoint, "SerialPort", ""),
		Baud:        GetSafeNumber(endpoint, "BaudRate", 9600),
		ReadTimeout: time.Duration(GetSafeNumber(endpoint, "ReadTimeoutSeconds", 1)),
		Size:        byte(GetSafeNumber(endpoint, "DataBits", 8)),
		Parity:      serial.Parity(GetSafeNumber(endpoint, "Parity", 0)),
		StopBits:    serial.StopBits(GetSafeNumber(endpoint, "StopBits", 1)),
	}

	packetHeader := GetSettingSafe(endpoint, "PacketHeader", "&&")
	packetFooter := GetSettingSafe(endpoint, "PacketFooter", "!!")
	lineEnding := GetSettingSafe(endpoint, "LineSeparator", "\r\n")
	heartBeatValue := GetSettingSafe(endpoint, "HeartBeatValue", "&&\r\n0111-9999\r\n!!\r\n")
	heartBeatInterval := GetSafeNumber(endpoint, "HeartBeatInterval", 30)
	packetFooterWithLineSeparator := packetFooter + lineEnding
	outputRaw := GetSafeBool(endpoint, "OutputRaw", false)

	log.Debug("Serial Config: ", config)
	log.Debug("packetHeader: ", packetHeader)
	log.Debug("packetFooter: ", packetFooter)
	log.Debug("lineEnding: ", lineEnding)
	log.Debug("heartBeatValue: ", heartBeatValue)
	log.Debug("heartBeatInterval: ", heartBeatInterval)
	log.Debug("outputRaw: ", outputRaw)

	log.Info("Connecting to serial port: " + t.config.GetSetting("SerialPort"))
	stream, err := serial.OpenPort(config)
	if err != nil {
		log.Error(err)
		return
	}
	log.Info("Connected to serial port")

	buf := make([]byte, 1024)
	buffer := bytes.NewBufferString("")
	t.heartBeat(heartBeatInterval, heartBeatValue, stream)
readLoop:
	for {
		buffer = readSerialData(endpoint, buffer, buf, stream, outputRaw, packetHeader, packetFooter, packetFooterWithLineSeparator, lineEnding)
		select {
		case <-t.stopCheck:
			break readLoop
		default:
		}
	}
}

func (t *wits0Trigger) heartBeat(heartBeatInterval int, heartBeatValue string, stream *serial.Port) {
	if heartBeatInterval > 0 {
		duration := time.Second * time.Duration(heartBeatInterval)

		go func() {
			start := time.Now()
		writeLoop:
			for {
				time.Sleep(time.Millisecond * 100)
				select {
				case <-t.stopCheck:
					break writeLoop
				default:
				}

				elapsed := time.Now().Sub(start)
				if elapsed > duration {
					log.Debug("Sending heartbeat")
					stream.Write([]byte(heartBeatValue))
					start = time.Now()
				}
			}
		}()
	}
}

func readSerialData(endpoint *trigger.Handler, buffer *bytes.Buffer, buf []byte, stream *serial.Port, outputRaw bool, packetHeader string, packetFooter string, packetFooterWithLineSeparator string, lineEnding string) *bytes.Buffer {
	n, err := stream.Read(buf)
	if err != nil {
		log.Error(err)
	}
	if n > 0 {
		buffer.Write(buf[:n])
	}
	if buffer.Len() > 0 {
		check := buffer.String()
		indexStart := strings.Index(check, packetHeader)
		indexEnd := strings.Index(check, packetFooterWithLineSeparator)
		if indexEnd >= 0 && indexStart >= 0 && indexEnd < indexStart {
			log.Info("Dropping initial bad packet")
			return bytes.NewBufferString(check[indexStart:len(check)])
		}
		if indexStart >= 0 {
			startRemoved := string(check[indexStart:len(check)])
			indexStartSecond := indexStart + strings.Index(startRemoved, packetHeader)
			if indexStartSecond > 0 && indexStartSecond > indexStart && indexStartSecond < indexEnd {
				log.Info("Dropping bad packet")
				return bytes.NewBufferString(check[indexStartSecond+len(packetHeader) : len(check)])
			}
		}
		if indexStart >= 0 && indexEnd > indexStart {
			indexEndIncludeStopPacket := indexEnd + len(packetFooterWithLineSeparator)
			packet := check[indexStart:indexEndIncludeStopPacket]
			outputData := packet
			if !outputRaw {

				lines := strings.Split(packet, lineEnding)
				records := make([]wits0Record, len(lines)-3)
				parsingPackets := false
				index := 0
				for _, line := range lines {
					line = strings.Replace(line, lineEnding, "", -1)
					if parsingPackets {
						if line == packetFooter {
							parsingPackets = false
						} else {
							records[index].Record = line[0:2]
							records[index].Item = line[2:4]
							records[index].Data = line[4:len(line)]
							index = index + 1
						}
					} else if line == packetHeader {
						parsingPackets = true
					}
				}
				jsonRecord, err := json.Marshal(wits0Packet{records})
				if err != nil {
					log.Error("Error converting packet to JSON: ", err)
					outputData = ""
				} else {
					outputData = string(jsonRecord)
				}
			}
			if len(outputData) > 0 {
				trgData := make(map[string]interface{})
				trgData["data"] = outputData
				_, err := endpoint.Handle(context.Background(), trgData)
				if err != nil {
					log.Error("Error starting action: ", err.Error())
				}
			}
			return bytes.NewBufferString(check[indexEndIncludeStopPacket:len(check)])
		}
	}
	return buffer
}

// Stop implements trigger.wits0Trigger.Start
func (t *wits0Trigger) Stop() error {
	// stop the trigger
	close(t.stopCheck)
	return nil
}
