package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/imtaco/vwmgr/pkg/common"
	"github.com/imtaco/vwmgr/pkg/mgr"
	"github.com/imtaco/vwmgr/pkg/utils"
	"github.com/jessevdk/go-flags"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type appArgs struct {
	DatabaseURL       string `long:"database_url" env:"DATABASE_URL"`
	BindAddr          string `long:"bind_addr" env:"BIND_ADDR" default:":9090"`
	APIKey            string `long:"api_key" env:"API_KEY"`
	SaUserEmail       string `long:"sa_user_email" env:"SA_USER_EMAIL"`
	SaPassword        string `long:"sa_user_password" env:"SA_USER_PASSWORD"`
	MigrateScriptPath string `long:"migrate_script_path" env:"MIGRATE_SCRIPT_PATH" default:"./migration"`
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatalf("fail to open DB: %v", err)
	}

	// migration
	if err := goose.SetDialect(string(goose.DialectPostgres)); err != nil {
		log.Fatalf("failed to set dialect: %v", err)
	}
	sqlDb, err := db.DB()
	if err != nil {
		log.Fatalf("an error occurred receiving the db instance: %v", err)
	}
	if err := goose.Up(sqlDb, args.MigrateScriptPath); err != nil {
		log.Fatalf("an error occurred during migration: %v", err)
	}

	orgSymKeys, err := common.GetOrgSymKeys(db, args.SaUserEmail, args.SaPassword)
	if err != nil {
		log.Fatalf("fail to get orgSymKey %v", err)
	}

	mgr := mgr.New(orgSymKeys, args.APIKey, db)

	// TODO: switch to prod
	g := gin.Default()
	mgr.Bind(g)

	g.Run(args.BindAddr)
}
