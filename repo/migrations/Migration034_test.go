package migrations_test

import (
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/OpenBazaar/openbazaar-go/repo/migrations"
	"github.com/OpenBazaar/openbazaar-go/schema"
)

const preMigration034Config = `{
	"Bootstrap": [
        "/ip4/107.170.133.32/tcp/4001/ipfs/QmUZRGLhcKXF1JyuaHgKm23LvqcoMYwtb9jmh8CkP4og3K",
        "/ip4/139.59.174.197/tcp/4001/ipfs/QmZfTbnpvPwxCjpCG3CXJ7pfexgkBZ2kgChAiRJrTK1HsM",
        "/ip4/139.59.6.222/tcp/4001/ipfs/QmRDcEDK9gSViAevCHiE6ghkaBCU7rTuQj4BDpmCzRvRYg"
    ],
	"DataSharing": {
		"AcceptStoreRequests": false,
		"PushTo": [
			"QmbwN82MVyBukT7WTdaQDppaACo62oUfma8dUa5R9nBFHm",
			"QmPPg2qeF3n2KvTRXRZLaTwHCw8JxzF4uZK93RfMoDvf2o",
			"QmY8puEnVx66uEet64gAf4VZRo7oUyMCwG6KdB9KM92EGQ"
		]
	},
	"OtherConfigProperty1": [1, 2, 3],
	"OtherConfigProperty2": "abc123"
}`

const postMigration034Config = `{
	"Bootstrap": [
        "/ip4/107.170.133.32/tcp/4001/ipfs/QmUZRGLhcKXF1JyuaHgKm23LvqcoMYwtb9jmh8CkP4og3K",
        "/ip4/139.59.174.197/tcp/4001/ipfs/QmZfTbnpvPwxCjpCG3CXJ7pfexgkBZ2kgChAiRJrTK1HsM",
        "/ip4/139.59.6.222/tcp/4001/ipfs/QmRDcEDK9gSViAevCHiE6ghkaBCU7rTuQj4BDpmCzRvRYg",
		"/ip4/45.76.183.141/tcp/4001/ipfs/QmV2B7fcVR6o8ZKs7D8vexhhQjjKZtofJzoFsx44X2ioEE",
        "/ip4/137.220.50.87/tcp/4001/ipfs/QmSqRoRDqGWd9VLQVAWHqLmBH6RW93CPY7vdqXCZELCt52"
    ],
	"DataSharing": {
		"AcceptStoreRequests": false,
		"PushTo": [
			"QmbwN82MVyBukT7WTdaQDppaACo62oUfma8dUa5R9nBFHm",
			"QmPPg2qeF3n2KvTRXRZLaTwHCw8JxzF4uZK93RfMoDvf2o",
			"QmY8puEnVx66uEet64gAf4VZRo7oUyMCwG6KdB9KM92EGQ",
			"QmV2B7fcVR6o8ZKs7D8vexhhQjjKZtofJzoFsx44X2ioEE",
			"QmSqRoRDqGWd9VLQVAWHqLmBH6RW93CPY7vdqXCZELCt52"
		]
	},
	"OtherConfigProperty1": [1, 2, 3],
	"OtherConfigProperty2": "abc123"
}`

func TestMigration034(t *testing.T) {
	var testRepo, err = schema.NewCustomSchemaManager(schema.SchemaContext{
		DataPath:        schema.GenerateTempPath(),
		TestModeEnabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err = testRepo.BuildSchemaDirectories(); err != nil {
		t.Fatal(err)
	}
	defer testRepo.DestroySchemaDirectories()

	var (
		configPath  = testRepo.DataPathJoin("config")
		repoverPath = testRepo.DataPathJoin("repover")
	)
	if err = ioutil.WriteFile(configPath, []byte(preMigration034Config), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if err = ioutil.WriteFile(repoverPath, []byte("30"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	var m migrations.Migration034
	err = m.Up(testRepo.DataPath(), "", true)
	if err != nil {
		t.Fatal(err)
	}

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var re = regexp.MustCompile(`\s`)
	if re.ReplaceAllString(string(configBytes), "") != re.ReplaceAllString(string(postMigration034Config), "") {
		t.Logf("actual: %s", re.ReplaceAllString(string(configBytes), ""))
		t.Fatal("incorrect post-migration config")
	}

	assertCorrectRepoVer(t, repoverPath, "35")

	err = m.Down(testRepo.DataPath(), "", true)
	if err != nil {
		t.Fatal(err)
	}

	configBytes, err = ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if re.ReplaceAllString(string(configBytes), "") != re.ReplaceAllString(string(preMigration034Config), "") {
		t.Logf("actual: %s", re.ReplaceAllString(string(configBytes), ""))
		t.Fatal("incorrect post-migration config")
	}

	assertCorrectRepoVer(t, repoverPath, "34")
}
