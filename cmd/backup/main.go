package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/go-resty/resty/v2"
	"github.com/imtaco/vwmgr/pkg/common"
	"github.com/imtaco/vwmgr/pkg/pkcs"
	"github.com/imtaco/vwmgr/pkg/utils"
	"github.com/jessevdk/go-flags"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type appArgs struct {
	DatabaseURL  string `long:"database_url" env:"DATABASE_URL"`
	BaseURL      string `long:"base_url" env:"BASE_URL"`
	SaUserEmail  string `long:"sa_user_email" env:"SA_USER_EMAIL"`
	SaPassword   string `long:"sa_user_password" env:"SA_USER_PASSWORD"`
	DeviceID     string `long:"device_id" env:"DEVICE_ID"`
	OutputFolder string `long:"output_folder" env:"OUTPUT_FOLDER"`
}

type modifyFunc func(value interface{}) interface{}

func main() {
	args := appArgs{}
	if _, err := flags.Parse(&args); err != nil {
		log.Fatal(err)
	}

	restyClient := resty.New()

	userMasterKey := pkcs.DeriveMasterKey(args.SaUserEmail, args.SaPassword)
	passwordHash := pkcs.DerivePasswordHash(userMasterKey, args.SaPassword)

	resp, err := restyClient.R().
		SetHeader("Content-Type", "application/x-www-form-urlencoded").
		SetFormData(map[string]string{
			"grant_type":        "password",
			"scope":             "api offline_access",
			"client_id":         "web",
			"username":          args.SaUserEmail,
			"password":          passwordHash,
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

	dsn, err := utils.PGURLtoGormDSN(args.DatabaseURL)
	if err != nil {
		log.Fatalf("fail to convert pg URL to dsn %v", err)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("fail to open DB", err)
	}

	orgSymKeys, err := common.GetOrgSymKeys(db, args.SaUserEmail, args.SaPassword)
	if err != nil {
		log.Fatalf("fail to get orgSymKey %v", err)
	}

	for orgUUID, orgSymKey := range orgSymKeys {
		outputFile := filepath.Join(args.OutputFolder, fmt.Sprintf("%s.json", orgUUID))

		// Use Access Token to get JSON data
		apiResp, err := restyClient.R().
			SetAuthToken(token.AccessToken).
			SetHeader("Accept", "application/json").
			Get(fmt.Sprintf("%s//api/organizations/%s/export", args.BaseURL, orgUUID))

		if err != nil {
			log.Fatalf("fail to fetch data: %v", err)
		}
		if !resp.IsSuccess() {
			log.Fatalf("fail to fetch data, status: %d, msg: %s", apiResp.StatusCode(), string(apiResp.Body()))
		}
		log.Printf("✅ data received: %s", orgUUID)

		var results interface{}
		if err := json.Unmarshal(apiResp.Body(), &results); err != nil {
			log.Fatal(err)
		}

		mod := func(value interface{}) interface{} {
			// attempt to decrypt fields using the Bitwarden format
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
		if err := os.WriteFile(outputFile, bs, 0644); err != nil {
			log.Fatalf("fail to write file %s, err: %v", outputFile, err)
		}
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
