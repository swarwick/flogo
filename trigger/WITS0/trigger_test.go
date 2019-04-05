package wits0

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/TIBCOSoftware/flogo-lib/logger"

	"github.com/TIBCOSoftware/flogo-lib/core/trigger"
)

func getJSONMetadata() string {
	jsonMetadataBytes, err := ioutil.ReadFile("trigger.json")
	if err != nil {
		panic("No Json Metadata found for trigger.json path")
	}
	return string(jsonMetadataBytes)
}

const testConfig string = `{
	"id": "wits0",
	"settings": {
		"SerialPort": "/dev/ttyUSB0",
		"HeartBeatValue": "&&\n0111-9999\n!!",
		"PacketHeader": "&&",
		"PacketFooter": "!!",
		"LineSeparator":"\r\n"
	},
	"handlers": [{
		"action": {
			"id": "local://testFlow",
			"settings": {}
		}
	}]
}`

type initContext struct {
	handlers []*trigger.Handler
}

func (ctx initContext) GetHandlers() []*trigger.Handler {
	return ctx.handlers
}
func TestCreate(t *testing.T) {
	log.SetLogLevel(logger.DebugLevel)
	// New factory
	md := trigger.NewMetadata(getJSONMetadata())
	f := NewFactory(md)

	if f == nil {
		t.Fail()
		return
	}

	// New Trigger
	config := trigger.Config{}
	jsonErr := json.Unmarshal([]byte(testConfig), &config)
	if jsonErr != nil {
		log.Error(jsonErr)
		t.Fail()
		return
	}
	trg := f.New(&config)

	if trg == nil {
		t.Fail()
	}

	initCtx := &initContext{
		handlers: make([]*trigger.Handler, 0, len(config.Handlers)),
	}

	newTrg, _ := trg.(trigger.Initializable)
	newTrg.Initialize(initCtx)

	go func() {
		time.Sleep(time.Second * 5)
		trg.Stop()
	}()

	trg.Start()
}
