package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/juju/errors"
	"github.com/juju/loggo"
	"github.com/juju/utils/clock"
	"github.com/juju/version"
	"gopkg.in/juju/names.v2"

	"github.com/juju/juju/agent"
	"github.com/juju/juju/mongo"
	"github.com/juju/juju/state"
)

var logger = loggo.GetLogger("forceagentversion")

func checkErr(label string, err error) {
	if err != nil {
		logger.Errorf("%s: %s", label, err)
		os.Exit(1)
	}
}

const dataDir = "/var/lib/juju"

func getState() (*state.State, error) {
	tag, err := getCurrentMachineTag(dataDir)
	if err != nil {
		return nil, errors.Annotate(err, "finding machine tag")
	}

	logger.Infof("current machine tag: %s", tag)

	config, err := getConfig(tag)
	if err != nil {
		return nil, errors.Annotate(err, "loading agent config")
	}

	mongoInfo, available := config.MongoInfo()
	if !available {
		return nil, errors.New("mongo info not available from agent config")
	}
	st, err := state.Open(state.OpenParams{
		Clock:              clock.WallClock,
		ControllerTag:      config.Controller(),
		ControllerModelTag: config.Model(),
		MongoInfo:          mongoInfo,
		MongoDialOpts:      mongo.DefaultDialOpts(),
	})
	if err != nil {
		return nil, errors.Annotate(err, "opening state connection")
	}
	return st, nil
}

func getCurrentMachineTag(datadir string) (names.MachineTag, error) {
	var empty names.MachineTag
	values, err := filepath.Glob(filepath.Join(datadir, "agents", "machine-*"))
	if err != nil {
		return empty, errors.Annotate(err, "problem globbing")
	}
	switch len(values) {
	case 0:
		return empty, errors.Errorf("no machines found")
	case 1:
		return names.ParseMachineTag(filepath.Base(values[0]))
	default:
		return empty, errors.Errorf("too many options: %v", values)
	}
}

func getConfig(tag names.MachineTag) (agent.ConfigSetterWriter, error) {
	path := agent.ConfigPath("/var/lib/juju", tag)
	return agent.ReadConfig(path)
}

func main() {
	loggo.GetLogger("").SetLogLevel(loggo.TRACE)

	st, err := getState()
	checkErr("getting state connection", err)
	defer st.Close()

	args := os.Args
	if len(args) < 3 {
		fmt.Printf("Useage: %s <model-uuid> <version>", args[0])
		os.Exit(1)
	}

	modelUUID := args[1]
	agentVersion := version.MustParse(args[2])

	modelSt, err := st.ForModel(names.NewModelTag(modelUUID))
	checkErr("open model", err)
	checkErr("set model agent version", modelSt.SetModelAgentVersion(agentVersion))
}
