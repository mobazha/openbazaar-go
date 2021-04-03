package migrations

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

var (
	Migration034BootstrapBefore = []string{
		"/ip4/107.170.133.32/tcp/4001/ipfs/QmUZRGLhcKXF1JyuaHgKm23LvqcoMYwtb9jmh8CkP4og3K",
		"/ip4/139.59.174.197/tcp/4001/ipfs/QmZfTbnpvPwxCjpCG3CXJ7pfexgkBZ2kgChAiRJrTK1HsM",
		"/ip4/139.59.6.222/tcp/4001/ipfs/QmRDcEDK9gSViAevCHiE6ghkaBCU7rTuQj4BDpmCzRvRYg",
	}

	Migration034BootstrapAfter = []string{
		"/ip4/107.170.133.32/tcp/4001/ipfs/QmUZRGLhcKXF1JyuaHgKm23LvqcoMYwtb9jmh8CkP4og3K",
		"/ip4/139.59.174.197/tcp/4001/ipfs/QmZfTbnpvPwxCjpCG3CXJ7pfexgkBZ2kgChAiRJrTK1HsM",
		"/ip4/139.59.6.222/tcp/4001/ipfs/QmRDcEDK9gSViAevCHiE6ghkaBCU7rTuQj4BDpmCzRvRYg",
		"/ip4/45.76.183.141/tcp/4001/ipfs/QmV2B7fcVR6o8ZKs7D8vexhhQjjKZtofJzoFsx44X2ioEE",
		"/ip4/137.220.50.87/tcp/4001/ipfs/QmSqRoRDqGWd9VLQVAWHqLmBH6RW93CPY7vdqXCZELCt52",
	}

	Migration034PushToBefore = []string{
		"QmbwN82MVyBukT7WTdaQDppaACo62oUfma8dUa5R9nBFHm",
		"QmPPg2qeF3n2KvTRXRZLaTwHCw8JxzF4uZK93RfMoDvf2o",
		"QmY8puEnVx66uEet64gAf4VZRo7oUyMCwG6KdB9KM92EGQ",
	}

	Migration034PushToAfter = []string{
		"QmbwN82MVyBukT7WTdaQDppaACo62oUfma8dUa5R9nBFHm",
		"QmPPg2qeF3n2KvTRXRZLaTwHCw8JxzF4uZK93RfMoDvf2o",
		"QmY8puEnVx66uEet64gAf4VZRo7oUyMCwG6KdB9KM92EGQ",
		"QmV2B7fcVR6o8ZKs7D8vexhhQjjKZtofJzoFsx44X2ioEE",
		"QmSqRoRDqGWd9VLQVAWHqLmBH6RW93CPY7vdqXCZELCt52",
	}
)

type migration034Bootstrap struct {
	PushTo []string
}

type migration034DataSharing struct {
	AcceptStoreRequests bool
	PushTo              []string
}

type Migration034 struct{}

func (Migration034) Up(repoPath, dbPassword string, testnet bool) error {
	var (
		configMap        = map[string]interface{}{}
		configBytes, err = ioutil.ReadFile(path.Join(repoPath, "config"))
	)
	if err != nil {
		return fmt.Errorf("reading config: %s", err.Error())
	}

	if err = json.Unmarshal(configBytes, &configMap); err != nil {
		return fmt.Errorf("unmarshal config: %s", err.Error())
	}

	configMap["DataSharing"] = migration034DataSharing{PushTo: Migration034PushToAfter}
	configMap["Bootstrap"] = Migration034BootstrapAfter

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	if err := writeRepoVer(repoPath, 35); err != nil {
		return fmt.Errorf("bumping repover to 35: %s", err.Error())
	}
	return nil
}

func (Migration034) Down(repoPath, dbPassword string, testnet bool) error {
	var (
		configMap        = map[string]interface{}{}
		configBytes, err = ioutil.ReadFile(path.Join(repoPath, "config"))
	)
	if err != nil {
		return fmt.Errorf("reading config: %s", err.Error())
	}

	if err = json.Unmarshal(configBytes, &configMap); err != nil {
		return fmt.Errorf("unmarshal config: %s", err.Error())
	}

	configMap["DataSharing"] = migration034DataSharing{PushTo: Migration034PushToBefore}
	configMap["Bootstrap"] = Migration034BootstrapBefore

	newConfigBytes, err := json.MarshalIndent(configMap, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal migrated config: %s", err.Error())
	}

	if err := ioutil.WriteFile(path.Join(repoPath, "config"), newConfigBytes, os.ModePerm); err != nil {
		return fmt.Errorf("writing migrated config: %s", err.Error())
	}

	if err := writeRepoVer(repoPath, 34); err != nil {
		return fmt.Errorf("dropping repover to 34: %s", err.Error())
	}
	return nil
}
