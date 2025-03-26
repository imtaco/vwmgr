package mgr

import (
	"crypto/rsa"
	"crypto/x509"

	"github.com/imtaco/vwmgr/pkg/model"
	"github.com/imtaco/vwmgr/pkg/pkcs"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func (m *VMManager) resetUserPassword(
	email string,
	newMasterPassword string,
) error {
	// simple validation
	if email == "" {
		return errors.New("email is required")
	}

	userMasterKey := pkcs.DeriveMasterKey(email, newMasterPassword)
	passwordHash := pkcs.DerivePasswordHash(userMasterKey, newMasterPassword)

	salt := pkcs.RandBytes(64)
	hashPwdHash := pkcs.HashPasswordHash(passwordHash, salt)

	symKey := pkcs.RandBytes(64)
	userAkey := pkcs.BWSymEncrypt(userMasterKey, symKey)

	publicKey, privateKey := pkcs.GenRSAKeyPair()

	pubInf, err := x509.ParsePKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}

	user := model.User{}
	if err := m.db.Where("email = ?", email).First(&user).Error; err != nil {
		return err
	}

	userOrgs := []model.UsersOrganization{}
	if err := m.db.Where("user_uuid = ?", user.UUID).Find(&userOrgs).Error; err != nil {
		// not found or real error
		return err
	}

	// check orgSymKey first
	for _, uo := range userOrgs {
		if _, ok := m.orgSymKeys[uo.OrgUUID]; !ok {
			return errors.Errorf("fail to found orr symmetric key of %s", uo.OrgUUID)
		}
	}

	return m.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&model.User{}).Where("uuid = ?", user.UUID).
			Updates(map[string]interface{}{
				"password_hash": hashPwdHash,
				"salt":          salt,
				"akey":          userAkey,
				"public_key":    pkcs.Base64Encode(publicKey),
				"private_key":   pkcs.BWSymEncrypt(symKey, privateKey),
			}).Error
		if err != nil {
			return err
		}

		for _, uo := range userOrgs {
			err = tx.Model(&model.UsersOrganization{}).
				Where("uuid = ?", uo.UUID).
				Update("akey", pkcs.BWPKEncrypt(m.orgSymKeys[uo.OrgUUID], pubInf.(*rsa.PublicKey))).
				Error
			if err != nil {
				return err
			}
		}

		// TODO: need to change security_stamp ?

		return nil
	})
}
