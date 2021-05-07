package migrations_test

import (
	"io/ioutil"
	"os"
	"regexp"
	"testing"

	"github.com/OpenBazaar/openbazaar-go/repo/migrations"
	"github.com/OpenBazaar/openbazaar-go/schema"
)

const preMigration035Config = `{
	"OtherConfigProperty1": [1, 2, 3],
	"OtherConfigProperty2": "abc123",
	"Wallets": {
        "BCH": {
            "API": [
                "https://bch.trezor.io/api"
            ],
            "APITestnet": [
                "https://tbch.trezor.io/api"
            ]
        },
		"BTC": {
            "API": [
                "https://btc.trezor.io/api"
            ],
            "APITestnet": [
                "https://tbtc.trezor.io/api"
            ]
        },
        "LTC": {
            "API": [
                "https://ltc.trezor.io/api"
            ],
            "APITestnet": [
                "https://tltc.trezor.io/api"
            ]
        },
        "ZEC": {
            "API": [
                "https://zec.trezor.io/api"
            ],
            "APITestnet": [
                "https://tzec.trezor.io/api"
            ]
        }
    }
}`

const postMigration035Config = `{
	"OtherConfigProperty1": [1, 2, 3],
	"OtherConfigProperty2": "abc123",
	"Wallets": {
        "BCH": {
            "API": [
                "https://bch1.mobazha.com/api"
            ],
            "APITestnet": [
                "https://tbch1.trezor.io/api"
            ]
        },
        "BTC": {
            "API": [
                "https://btc1.mobazha.com/api"
            ],
            "APITestnet": [
                "https://tbtc1.trezor.io/api"
            ]
        },
        "LTC": {
            "API": [
                "https://ltc1.mobazha.com/api"
            ],
            "APITestnet": [
                "https://tltc1.trezor.io/api"
            ]
        },
        "ZEC": {
            "API": [
                "https://zec1.mobazha.com/api"
            ],
            "APITestnet": [
                "https://tzec1.trezor.io/api"
            ]
        }
    }
}`

func TestMigration035(t *testing.T) {
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
	if err = ioutil.WriteFile(configPath, []byte(preMigration035Config), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if err = ioutil.WriteFile(repoverPath, []byte("30"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	var m migrations.Migration035
	err = m.Up(testRepo.DataPath(), "", true)
	if err != nil {
		t.Fatal(err)
	}

	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	var re = regexp.MustCompile(`\s`)
	if re.ReplaceAllString(string(configBytes), "") != re.ReplaceAllString(string(postMigration035Config), "") {
		t.Logf("actual: %s", re.ReplaceAllString(string(configBytes), ""))
		t.Fatal("incorrect post-migration config")
	}

	assertCorrectRepoVer(t, repoverPath, "36")

	err = m.Down(testRepo.DataPath(), "", true)
	if err != nil {
		t.Fatal(err)
	}

	configBytes, err = ioutil.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}

	if re.ReplaceAllString(string(configBytes), "") != re.ReplaceAllString(string(preMigration035Config), "") {
		t.Logf("actual: %s", re.ReplaceAllString(string(configBytes), ""))
		t.Fatal("incorrect post-migration config")
	}

	assertCorrectRepoVer(t, repoverPath, "35")
}
