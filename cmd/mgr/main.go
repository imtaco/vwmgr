package main

import (
	"encoding/hex"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/imtaco/vwmgr/pkg/mgr"
	"github.com/imtaco/vwmgr/pkg/utils"
	"github.com/jessevdk/go-flags"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type appArgs struct {
	DatabaseURL  string `long:"database_url" env:"DATABASE_URL"`
	BindAddr     string `long:"bind_addr" env:"BIND_ADDR" default:":9090"`
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

	dsn, err := utils.PGURLtoGormDSN(args.DatabaseURL)
	if err != nil {
		log.Fatalf("fail to convert pg URL to dsn %v", err)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("fail to open DB", err)
	}

	mgr := mgr.New(args.OrgUUID, orgSymKey, db)

	// TODO: switch to prod
	g := gin.Default()
	mgr.Bind(g)

	g.Run(args.BindAddr)
}
