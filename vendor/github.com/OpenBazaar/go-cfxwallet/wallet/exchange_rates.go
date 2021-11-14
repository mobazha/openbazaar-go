package wallet

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	exchange "github.com/OpenBazaar/spvwallet/exchangerates"
	"golang.org/x/net/proxy"
)

// ExchangeRateProvider - used for looking up exchange rates for CFX
type ExchangeRateProvider struct {
	fetchURL        string
	cache           map[string]float64
	client          *http.Client
	decoder         ExchangeRateDecoder
	bitcoinProvider *exchange.BitcoinPriceFetcher
}

// ExchangeRateDecoder - used for serializing/deserializing provider struct
type ExchangeRateDecoder interface {
	decode(dat interface{}, cache map[string]float64, bp *exchange.BitcoinPriceFetcher) (err error)
}

// MobazhaDecoder - decoder to be used by OB
type MobazhaDecoder struct{}

// ConfluxPriceFetcher - get CFX prices from the providers (exchanges)
type ConfluxPriceFetcher struct {
	sync.Mutex
	cache     map[string]float64
	providers []*ExchangeRateProvider
}

// NewConfluxPriceFetcher - instantiate a cfx price fetcher
func NewConfluxPriceFetcher(dialer proxy.Dialer) *ConfluxPriceFetcher {
	bp := exchange.NewBitcoinPriceFetcher(dialer)
	z := ConfluxPriceFetcher{
		cache: make(map[string]float64),
	}
	dial := net.Dial
	if dialer != nil {
		dial = dialer.Dial
	}
	tbTransport := &http.Transport{Dial: dial}
	client := &http.Client{Transport: tbTransport, Timeout: time.Minute}

	z.providers = []*ExchangeRateProvider{
		{"https://mobazha.info/api/ticker", z.cache, client, MobazhaDecoder{}, bp},
	}
	go z.run()
	return &z
}

// GetExchangeRate - fetch the exchange rate for the specified currency
func (z *ConfluxPriceFetcher) GetExchangeRate(currencyCode string) (float64, error) {
	currencyCode = NormalizeCurrencyCode(currencyCode)

	z.Lock()
	defer z.Unlock()
	price, ok := z.cache[currencyCode]
	if !ok {
		return 0, errors.New("currency not tracked")
	}
	return price, nil
}

// GetLatestRate - refresh the cache and return the latest exchange rate for the specified currency
func (z *ConfluxPriceFetcher) GetLatestRate(currencyCode string) (float64, error) {
	currencyCode = NormalizeCurrencyCode(currencyCode)

	z.fetchCurrentRates()
	z.Lock()
	defer z.Unlock()
	price, ok := z.cache[currencyCode]
	if !ok {
		return 0, errors.New("currency not tracked")
	}
	return price, nil
}

// GetAllRates - refresh the cache
func (z *ConfluxPriceFetcher) GetAllRates(cacheOK bool) (map[string]float64, error) {
	if !cacheOK {
		err := z.fetchCurrentRates()
		if err != nil {
			return nil, err
		}
	}
	z.Lock()
	defer z.Unlock()
	copy := make(map[string]float64, len(z.cache))
	for k, v := range z.cache {
		copy[k] = v
	}
	return copy, nil
}

// UnitsPerCoin - return Drip in 1 CFX
func (z *ConfluxPriceFetcher) UnitsPerCoin() int64 {
	return 1000000000000000000
}

func (z *ConfluxPriceFetcher) fetchCurrentRates() error {
	z.Lock()
	defer z.Unlock()
	for _, provider := range z.providers {
		err := provider.fetch()
		if err == nil {
			return nil
		}
	}
	return errors.New("all exchange rate API queries failed")
}

func (z *ConfluxPriceFetcher) run() {
	z.fetchCurrentRates()
	ticker := time.NewTicker(time.Minute * 15)
	defer ticker.Stop()
	for range ticker.C {
		z.fetchCurrentRates()
	}
}

func (provider *ExchangeRateProvider) fetch() (err error) {
	if len(provider.fetchURL) == 0 {
		err = errors.New("provider has no fetchUrl")
		return err
	}
	resp, err := provider.client.Get(provider.fetchURL)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(resp.Body)
	var dataMap interface{}
	err = decoder.Decode(&dataMap)
	if err != nil {
		return err
	}
	return provider.decoder.decode(dataMap, provider.cache, provider.bitcoinProvider)
}

func (b MobazhaDecoder) decode(dat interface{}, cache map[string]float64, bp *exchange.BitcoinPriceFetcher) (err error) {
	//data := dat.(map[string]interface{})
	data, ok := dat.(map[string]interface{})
	if !ok {
		return errors.New(reflect.TypeOf(b).Name() + ".decode: type assertion failed invalid json")
	}

	cfx, ok := data["CFX"]
	if !ok {
		return errors.New(reflect.TypeOf(b).Name() + ".decode: type assertion failed, missing 'CFX' field")
	}
	val, ok := cfx.(map[string]interface{})
	if !ok {
		return errors.New(reflect.TypeOf(b).Name() + ".decode: type assertion failed")
	}
	cfxRate, ok := val["last"].(float64)
	if !ok {
		return errors.New(reflect.TypeOf(b).Name() + ".decode: type assertion failed, missing 'last' (float) field")
	}
	for k, v := range data {
		if k != "timestamp" {
			val, ok := v.(map[string]interface{})
			if !ok {
				return errors.New(reflect.TypeOf(b).Name() + ".decode: type assertion failed")
			}
			price, ok := val["last"].(float64)
			if !ok {
				return errors.New(reflect.TypeOf(b).Name() + ".decode: type assertion failed, missing 'last' (float) field")
			}
			cache[k] = price * (1 / cfxRate)
		}
	}
	return nil
}

// NormalizeCurrencyCode standardizes the format for the given currency code
func NormalizeCurrencyCode(currencyCode string) string {
	return strings.ToUpper(currencyCode)
}
