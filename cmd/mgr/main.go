package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/imtaco/vwmgr/pkg/common"
	"github.com/imtaco/vwmgr/pkg/mgr"
	"github.com/imtaco/vwmgr/pkg/utils"
	"github.com/jessevdk/go-flags"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type appArgs struct {
	DatabaseURL string `long:"database_url" env:"DATABASE_URL"`
	BindAddr    string `long:"bind_addr" env:"BIND_ADDR" default:":9090"`
	SaUserEmail string `long:"sa_user_email" env:"SA_USER_EMAIL"`
	SaPassword  string `long:"sa_user_password" env:"SA_USER_PASSWORD"`
}

func main() {
	args := appArgs{}
	if _, err := flags.Parse(&args); err != nil {
		log.Fatal("err:", err)
	}

	// TODO: args validation

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

	mgr := mgr.New(orgSymKeys, db)

	// TODO: switch to prod
	g := gin.Default()
	mgr.Bind(g)

	g.Run(args.BindAddr)
}
