package wits0

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/TIBCOSoftware/flogo-lib/core/trigger"
)

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

func createJSON(packet string, settings *wits0Settings) string {
	lines := strings.Split(packet, settings.lineEnding)
	records := make([]wits0Record, len(lines)-3)
	parsingPackets := false
	index := 0
	for _, line := range lines {
		line = strings.Replace(line, settings.lineEnding, "", -1)
		if parsingPackets {
			if line == settings.packetFooter {
				parsingPackets = false
			} else {
				records[index].Record = line[0:2]
				records[index].Item = line[2:4]
				records[index].Data = line[4:len(line)]
				index = index + 1
			}
		} else if line == settings.packetHeader {
			parsingPackets = true
		}
	}
	jsonRecord, err := json.Marshal(wits0Records{records})
	if err != nil {
		log.Error("Error converting packet to JSON: ", err)
		return ""
	}
	return string(jsonRecord)

}
