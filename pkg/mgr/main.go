package mgr

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/imtaco/vwmgr/pkg/pkcs"
	"gorm.io/gorm"
)

func New(
	orgSymKeys map[string][]byte,
	apiKey string,
	db *gorm.DB,
) *VMManager {
	return &VMManager{
		orgSymKeys: orgSymKeys,
		apiKey:     apiKey,
		db:         db,
	}
}

type VMManager struct {
	orgSymKeys map[string][]byte
	apiKey     string
	db         *gorm.DB
}

type userEmail struct {
	Email string `uri:"email" binding:"required,email,max=64"`
}

func (m *VMManager) Bind(g *gin.Engine) {
	g.Use(m.validateAPIKey)

	g.POST("/api/users", func(c *gin.Context) {
		type userInfo struct {
			Email    string   `json:"email" binding:"required,email,max=64"`
			Name     string   `json:"name" binding:"required,min=2,max=32"`
			Password string   `json:"password" binding:"required,min=12,max=128"`
			OrgUUID  []string `json:"org_uuid" binding:"required,dive,uuid"`
		}
		u := userInfo{}

		if err := c.ShouldBindJSON(&u); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("try to register %+v", u)

		if err := m.createUser(u.Email, u.Name, u.Password, u.OrgUUID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"status": "ok"})
	})

	g.POST("/api/users/:email/reset", func(c *gin.Context) {
		u := userEmail{}
		if err := c.ShouldBindUri(&u); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		type userInfo struct {
			NewPassword string `json:"new_password" binding:"required,min=12,max=128"`
		}
		nu := userInfo{}
		// Bind JSON from request body into `user`
		if err := c.ShouldBindJSON(&nu); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("try to reset %s", u.Email)

		if err := m.resetUserPassword(u.Email, nu.NewPassword); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	g.GET("/api/orgs/items", func(c *gin.Context) {
		log.Println("dump org items")

		items, err := m.listOrgItems()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		results := make([]orgItemDetail, 0, len(items))
		for _, d := range items {
			orgSymKey, ok := m.orgSymKeys[d.OrgUUID]
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "fail to find some org sym key"})
				return
			}

			p, err := pkcs.BWSymDecryptMany(orgSymKey, d.CollectionName, d.ItemName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			d.CollectionName, d.ItemName = string(p[0]), string(p[1])

			if d.AccountName != "" {
				accountNameDec, err := pkcs.BWSymDecrypt(orgSymKey, d.AccountName)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
					return
				}
				d.AccountName = string(accountNameDec)
			}

			results = append(results, d)
		}

		c.JSON(http.StatusOK, results)
	})

	g.GET("/api/users/:email/depart_report", func(c *gin.Context) {
		u := userEmail{}
		if err := c.ShouldBindUri(&u); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("get depart user report of %s", u.Email)

		items, err := m.userDepartReport(u.Email)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			}
			return
		}

		results := []leaveUserItem{}
		for _, d := range items {
			orgSymKey, ok := m.orgSymKeys[d.OrgUUID]
			if !ok {
				c.JSON(http.StatusNotFound, gin.H{"error": "fail to find some org sym key"})
				return
			}

			colName, err := pkcs.BWSymDecrypt(orgSymKey, d.CollectionName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			d.CollectionName = string(colName)
			results = append(results, d)
		}

		c.JSON(http.StatusOK, results)
	})
}

func (m *VMManager) validateAPIKey(c *gin.Context) {
	apiKey := c.Request.Header.Get("X-API-Key")
	if apiKey != m.apiKey {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Authentication failed"})
		return
	}
}
