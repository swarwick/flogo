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
			"id": "NextAction"
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
	return nil, nil
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
	newTrg, _ := trg.(trigger.Initializable)

	initCtx := &initContext{handlers: make([]*trigger.Handler, 0, len(config.Handlers))}
	runner := &TestRunner{}
	//create handlers for that trigger and init

	for _, hConfig := range config.Handlers {

		/*
			//create the action
			actionFactory := action.GetFactory(hConfig.Action.Ref)
			if actionFactory == nil {
				log.Errorf("Action Factory '%s' not registered", hConfig.Action.Ref)
			}

			triggerConfg := hConfig.Action
			act, err := actionFactory.New(triggerConfg.Config)
			if err != nil {
				log.Errorf("Error creating actionFactory: %v", err)
			}
		*/

		log.Infof("hConfig: %v", hConfig)
		log.Infof("trg.Metadata().Output: %v", trg.Metadata().Output)
		log.Infof("trg.Metadata().Reply: %v", trg.Metadata().Reply)

		//handler := &trigger.Handler{config: &hconfig, act: nil, outputMd: nil, replyMd: nil, runner: runner}
		handler := trigger.NewHandler(hConfig, nil, trg.Metadata().Output, trg.Metadata().Reply, runner)
		//handler := trigger.NewHandler(hConfig, act, trg.Metadata().Output, trg.Metadata().Reply, runner)
		initCtx.handlers = append(initCtx.handlers, handler)

	}

	newTrg.Initialize(initCtx)

	go func() {
		time.Sleep(time.Second * 5)
		trg.Stop()
	}()

	trg.Start()
}
