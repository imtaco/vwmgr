package mgr

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/imtaco/vwmgr/pkcs"
	"gorm.io/gorm"
)

func New(
	orgUUID string,
	orgSymKey []byte,
	db *gorm.DB,
) *VMManager {
	return &VMManager{
		orgUUID:   orgUUID,
		orgSymKey: orgSymKey,
		db:        db,
	}
}

type VMManager struct {
	orgUUID   string
	orgSymKey []byte
	db        *gorm.DB
}

type userEmail struct {
	Email string `uri:"email" binding:"required,email,max=64"`
}

func (m *VMManager) Bind(g *gin.Engine) {
	g.POST("/api/users/register", func(c *gin.Context) {
		type userInfo struct {
			Email    string `binding:"required,email,max=64"`
			Name     string `binding:"required,min=2,max=32"`
			Password string `binding:"required,min=12,max=128"`
		}
		u := userInfo{}

		if err := c.ShouldBindJSON(&u); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("try to register %+v", u)

		if err := m.createUser(u.Email, u.Name, u.Password); err != nil {
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
			NewPassword string `binding:"required,min=12,max=128"`
		}
		nu := userInfo{}

		// Bind JSON from request body into `user`
		if err := c.ShouldBindJSON(&u); err != nil {
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

	g.GET("/api/org/items", func(c *gin.Context) {
		log.Println("dump org items")

		items, err := m.listOrgItems()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		results := make([]orgItemDetail, 0, len(items))
		for _, d := range items {
			p, err := pkcs.BWSymDecryptMany(m.orgSymKey, d.CollectionName, d.CipherName)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			d.CollectionName, d.CipherName = string(p[0]), string(p[1])

			if d.AccountName != "" {
				accountNameDec, err := pkcs.BWSymDecrypt(m.orgSymKey, d.AccountName)
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

	g.GET("/api/users/:email/leave_report", func(c *gin.Context) {
		u := userEmail{}
		if err := c.ShouldBindUri(&u); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		log.Printf("get leave user report of %s", u.Email)

		items, err := m.userDepartReport(u.Email)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		results := []leaveUserItem{}
		for _, d := range items {
			colName, err := pkcs.BWSymDecrypt(m.orgSymKey, d.CollectionName)
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
