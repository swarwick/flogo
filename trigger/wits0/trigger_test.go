package wits0

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/TIBCOSoftware/flogo-lib/core/action"
	"github.com/TIBCOSoftware/flogo-lib/core/data"
	"github.com/TIBCOSoftware/flogo-lib/core/trigger"
	"github.com/TIBCOSoftware/flogo-lib/logger"
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

	},
	"handlers": [{
		"action": {
            "ref": "github.com/TIBCOSoftware/flogo-contrib/action/flow",
            "data": {
              "flowURI": "res://flow:query"
            },
            "mappings": {
              "input": [
                {
                  "mapTo": "request",
                  "type": "assign",
                  "value": "$.args"
                }
              ],
              "output": [
                {
                  "mapTo": "data",
                  "type": "assign",
                  "value": "$.resp"
                }
              ]
            }
          },
		"settings": {
			"SerialPort": "/dev/ttyUSB0",
			"HeartBeatValue": "&&\n0111-9999\n!!",
			"PacketHeader": "&&",
			"PacketFooter": "!!",
			"LineSeparator":"\r\n"
		}
	}]
}`

type initContext struct {
	handlers []*trigger.Handler
}

func (ctx initContext) GetHandlers() []*trigger.Handler {
	return ctx.handlers
}

type TestRunner struct {
}

var Test action.Runner

// Run implements action.Runner.Run
func (tr *TestRunner) Run(context context.Context, action action.Action, uri string, options interface{}) (code int, data interface{}, err error) {
	log.Infof("Ran Action (Run): %v", uri)
	return 0, nil, nil
}

func (tr *TestRunner) RunAction(ctx context.Context, act action.Action, options map[string]interface{}) (results map[string]*data.Attribute, err error) {
	log.Infof("Ran Action (RunAction): %v", act)
	return nil, nil
}

func (tr *TestRunner) Execute(ctx context.Context, act action.Action, inputs map[string]*data.Attribute) (results map[string]*data.Attribute, err error) {
	log.Infof("Ran Action (Execute): %v", act)
	value := inputs["data"].Value().(string)
	log.Info(value)
	return nil, nil
}

type TestAction struct {
}

func (tr *TestAction) Metadata() *action.Metadata {
	log.Infof("Metadata")
	return nil
}

func (tr *TestAction) IOMetadata() *data.IOMetadata {
	log.Infof("IOMetadata")
	return nil
}

func TestConnect(t *testing.T) {
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
	newTrg, _ := trg.(trigger.Initializable)

	initCtx := &initContext{handlers: make([]*trigger.Handler, 0, len(config.Handlers))}
	runner := &TestRunner{}
	action := &TestAction{}
	//create handlers for that trigger and init
	for _, hConfig := range config.Handlers {
		log.Infof("hConfig: %v", hConfig)
		log.Infof("trg.Metadata().Output: %v", trg.Metadata().Output)
		log.Infof("trg.Metadata().Reply: %v", trg.Metadata().Reply)

		handler := trigger.NewHandler(hConfig, action, trg.Metadata().Output, trg.Metadata().Reply, runner)
		initCtx.handlers = append(initCtx.handlers, handler)
	}

	newTrg.Initialize(initCtx)

	go func() {
		time.Sleep(time.Second * 5)
		trg.Stop()
	}()

	trg.Start()
}
