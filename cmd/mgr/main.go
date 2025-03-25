package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/imtaco/vwmgr/mgr"
	"github.com/jessevdk/go-flags"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type appArgs struct {
	DbHost     string `long:"db_host" env:"DB_HOST"`
	DbPort     int    `long:"db_port" env:"DB_PORT" default:"5432"`
	DbUser     string `long:"db_user" env:"DB_USER"`
	DbPassword string `long:"db_password" env:"DB_PASSWORD"`
	DbDatabase string `long:"db_database" env:"DB_DATABASE"`
	BindAddr   string `long:"bind_addr" env:"BIND_ADDR" default:":9090"`

	OrgUUID      string `long:"org_uuid" env:"ORG_UUID"`
	OrgSymKeyHex string `long:"org_sym_key_hex" env:"ORG_SYM_KEY_HEX"`
}

func main() {
	args := appArgs{}
	if _, err := flags.Parse(&args); err != nil {
		log.Fatal("err:", err)
	}

	// TODO: args validation

	orgSymKey, err := hex.DecodeString(args.OrgSymKeyHex)
	if err != nil {
		log.Fatalf("fail to parse org sym key %v", err)
	}

	db, err := gorm.Open(
		postgres.Open(
			fmt.Sprintf(
				"host=%s port=%d user=%s dbname=%s password=%s sslmode=disable",
				args.DbHost,
				args.DbPort,
				args.DbUser,
				args.DbDatabase,
				args.DbPassword,
			)),
		&gorm.Config{},
	)
	if err != nil {
		log.Fatal("fail to open DB", err)
	}

	mgr := mgr.New(args.OrgUUID, orgSymKey, db)

	// TODO: switch to prod
	g := gin.Default()
	mgr.Bind(g)

	g.Run(args.BindAddr)
}
