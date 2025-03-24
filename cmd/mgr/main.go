package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net/http"

	"github.com/caarlos0/env/v10"
	"github.com/gin-gonic/gin"
	"github.com/imtaco/vwmgr/mgr"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type appArgs struct {
	DbHost     string `env:"DB_HOST"`
	DbPort     int    `env:"DB_PORT"`
	DbUser     string `env:"DB_USER"`
	DbPassword string `env:"DB_PASSWORD"`
	DbDatabase string `env:"DB_DATABASE"`
	BindAddr   string `env:"BIND_ADDR" envDefault:":9090"`

	OrgUUID      string `env:"ORG_UUID"`
	OrgSymKeyHex string `env:"ORG_SYM_KEY_HEX"`
}
type userInfo struct {
	Email    string
	Name     string
	Password string
}

func main() {
	args := appArgs{}
	if err := env.Parse(&args); err != nil {
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

	g.POST("/api/register", func(c *gin.Context) {
		u := userInfo{}

		// Bind JSON from request body into `user`
		if err := c.ShouldBindJSON(&u); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// TODO: payload validation
		// email in email regepx & domain
		// len(password) >= 12
		// name is not empty
		log.Printf("try to register %+v", u)
		err = mgr.Register(
			u.Email,
			u.Name,
			u.Password,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}

		c.JSON(http.StatusCreated, gin.H{"status": "ok"})
	})

	g.Run(args.BindAddr)
}
