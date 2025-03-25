package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/go-resty/resty/v2"
	"github.com/imtaco/vwmgr/pkcs"
	"github.com/jessevdk/go-flags"
)

type appArgs struct {
	BaseURL      string `long:"base_url" env:"BASE_URL"`
	ClientID     string `long:"client_id" env:"CLIENT_ID"`
	ClientSecret string `long:"client_secret" env:"CLIENT_SECRET"`
	OrgUUID      string `long:"org_uuid" env:"ORG_UUID"`
	DeviceID     string `long:"device_id" env:"DEVICE_ID"`
	OrgSymKeyHex string `long:"org_sym_key_hex" env:"ORG_SYM_KEY_HEX"`
	OutputFile   string `long:"output_file" env:"OUTPUT_FILE"`
}

type modifyFunc func(value interface{}) interface{}

func main() {
	args := appArgs{}
	if _, err := flags.Parse(&args); err != nil {
		log.Fatal(err)
	}

	orgSymKey, err := hex.DecodeString(args.OrgSymKeyHex)
	if err != nil {
		log.Fatalf("fail to parse org sym key %v", err)
	}

	restyClient := resty.New()

	resp, err := restyClient.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"grant_type":        "client_credentials",
			"scope":             "api",
			"client_id":         args.ClientID,
			"client_secret":     args.ClientSecret,
			"device_type":       "9",
			"device_identifier": args.DeviceID,
			"device_name":       "chrome",
		}).
		Post(fmt.Sprintf("%s/identity/connect/token", args.BaseURL))
	if err != nil {
		log.Fatalf("fail to getting token: %v", err)
	}
	if !resp.IsSuccess() {
		log.Fatalf("fail to get token, status: %d, msg: %s", resp.StatusCode(), string(resp.Body()))
	}

	type tokenResponse struct {
		AccessToken string `json:"access_token"`
	}

	var token tokenResponse
	if err := json.Unmarshal(resp.Body(), &token); err != nil {
		log.Fatalf("failed to parse token response: %v", err)
	}

	log.Println("✅ access Token received")

	// Step 2: Use Access Token to get JSON data
	apiResp, err := restyClient.R().
		SetAuthToken(token.AccessToken).
		SetHeader("Accept", "application/json").
		Get(fmt.Sprintf("%s//api/organizations/%s/export", args.BaseURL, args.OrgUUID))

	if err != nil {
		log.Fatalf("fail to fetch data: %v", err)
	}
	if !resp.IsSuccess() {
		log.Fatalf("fail to fetch data, status: %d, msg: %s", apiResp.StatusCode(), string(apiResp.Body()))
	}
	log.Println("✅ data received:")

	var results interface{}
	if err := json.Unmarshal(apiResp.Body(), &results); err != nil {
		log.Fatal(err)
	}

	mod := func(value interface{}) interface{} {
		strValue, ok := value.(string)
		if !ok || !pkcs.IsBWSymFormat(strValue) {
			return value
		}

		bs, err := pkcs.BWSymDecrypt(orgSymKey, strValue)
		if err != nil {
			return value
		}

		return string(bs)
	}

	results = traverseAndModify(results, mod)

	bs, err := json.Marshal(results)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(args.OutputFile, bs, 0644); err != nil {
		log.Fatalf("fail to write file %s, err: %v", args.OutputFile, err)
	}
}

func traverseAndModify(data interface{}, modify modifyFunc) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		for key, val := range v {
			v[key] = traverseAndModify(val, modify)
		}
		return v
	case []interface{}:
		for i, item := range v {
			v[i] = traverseAndModify(item, modify)
		}
		return v
	default:
		return modify(v)
	}
}
