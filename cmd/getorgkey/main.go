package main

import (
	"fmt"
	"log"

	"github.com/imtaco/vwmgr/pkg/pkcs"
	"github.com/jessevdk/go-flags"
)

type appArgs struct {
	UserEmail      string `long:"user_email" env:"USER_EMAIL"`
	UserMasterPwd  string `long:"user_master_pwd" env:"USER_MASTER_PWD"`
	UserAKey       string `long:"user_akey" env:"USER_AKEY"`
	UserPrivateKey string `long:"user_private_key" venv:"USER_PRIVATE_KEY"`
	UserOrgAkey    string `long:"user_org_akey" venv:"USER_ORG_AKEY"`
}

func main() {
	args := appArgs{}
	if _, err := flags.Parse(&args); err != nil {
		log.Fatal("err:", err)
	}

	// TODO: retrieve from DB ?

	masterKey := pkcs.DeriveMasterKey(args.UserEmail, args.UserMasterPwd)
	symKey, err := pkcs.BWSymDecrypt(masterKey, args.UserAKey)
	if err != nil {
		log.Fatalf("fail to decrypt akey %v", err)
	}

	privateKey, err := pkcs.BWSymDecrypt(symKey, args.UserPrivateKey)
	if err != nil {
		log.Fatalf("fail to decrypt private key %v", err)
	}
	priInf, err := pkcs.PrivateKeyInfo(privateKey)
	if err != nil {
		log.Fatalf("fail to decrypt private key %v", err)
	}

	orgSymKey, err := pkcs.BWPKDecrypt(args.UserOrgAkey, priInf)
	if err != nil {
		log.Fatalf("fail to decrypt org akey %v", err)
	}

	fmt.Printf("org symmetric key: %x\n", orgSymKey)
}
