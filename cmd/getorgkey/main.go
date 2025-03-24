package main

import (
	"fmt"
	"log"

	"github.com/caarlos0/env/v10"
	"github.com/imtaco/vwmgr/pkcs"
)

type appArgs struct {
	UserEmail      string `env:"USER_EMAIL"`
	UserMasterPwd  string `env:"USER_MASTER_PWD"`
	UserAKey       string `env:"USER_AKEY"`
	UserPrivateKey string `env:"USER_PRIVATE_KEY"`
	UserOrgAkey    string `env:"USER_ORG_AKEY"`
}

func main() {
	args := appArgs{}
	if err := env.Parse(&args); err != nil {
		log.Fatal(err)
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
