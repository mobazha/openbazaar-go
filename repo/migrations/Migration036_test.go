package migrations_test

import (
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/OpenBazaar/openbazaar-go/repo/migrations"
	"github.com/OpenBazaar/openbazaar-go/schema"
)

const preMigration036Config = `{
	"OtherConfigProperty1": [1, 2, 3],
	"OtherConfigProperty2": "abc123",
	"Wallets": {
    }
}`

const postMigration036Config = `{
	"OtherConfigProperty1": [1, 2, 3],
	"OtherConfigProperty2": "abc123",
	"Wallets": {
        "CFX": {
            "API": [
                "https://main.confluxrpc.com"
            ],
            "APITestnet": [
                "https://test.confluxrpc.com"
            ],
            "FeeAPI": "",
            "HighFeeDefault": 30,
            "LowFeeDefault": 7,
            "MaxFee": 200,
            "MediumFeeDefault": 15,
            "SuperLowFeeDefault": 0,
            "TrustedPeer": "",
            "Type": "API",
            "WalletOptions": {
                "RegistryAddress": "0x5c69ccf91eab4ef80d9929b3c1b4d5bc03eb0981",
                "TestnetRegistryAddress": "cfxtest:aca77f0cck29xd6ur0z4tsvnxe4v4w1fhyxd99p1p3"
            }
        }
    }
}`

func TestMigration036(t *testing.T) {
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
	if err = ioutil.WriteFile(configPath, []byte(preMigration036Config), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if err = ioutil.WriteFile(repoverPath, []byte("30"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	var m migrations.Migration036
	err = m.Up(testRepo.DataPath(), "", true)
	if err != nil {
		t.Fatal(err)
	}

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var re = regexp.MustCompile(`\s`)
	if re.ReplaceAllString(string(configBytes), "") != re.ReplaceAllString(string(postMigration036Config), "") {
		t.Logf("actual: %s", re.ReplaceAllString(string(configBytes), ""))
		t.Fatal("incorrect post-migration config")
	}

	assertCorrectRepoVer(t, repoverPath, "37")

	err = m.Down(testRepo.DataPath(), "", true)
	if err != nil {
		t.Fatal(err)
	}

	configBytes, err = ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if re.ReplaceAllString(string(configBytes), "") != re.ReplaceAllString(string(preMigration036Config), "") {
		t.Logf("actual: %s", re.ReplaceAllString(string(configBytes), ""))
		t.Fatal("incorrect post-migration config")
	}

	assertCorrectRepoVer(t, repoverPath, "36")
}
