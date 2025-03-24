package mgr

import (
	"crypto/rsa"
	"crypto/x509"
	"errors"

	"github.com/imtaco/vwmgr/model"
	"github.com/imtaco/vwmgr/pkcs"
	"gorm.io/gorm"
)

func (m *VMManager) ResetPassword(
	email string,
	name string,
	newMasterPassword string,
) error {
	// simple validation
	if email == "" {
		return errors.New("email is required")
	}
	if name == "" {
		return errors.New("name is required")
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

	return m.db.Transaction(func(tx *gorm.DB) error {
		user := model.User{
			Email: email,
		}
		err := tx.Model(&user).Updates(model.User{
			PasswordHash: hashPwdHash,
			Salt:         salt,
			Akey:         userAkey,
			PublicKey:    pkcs.Base64Encode(publicKey),
			PrivateKey:   pkcs.BWSymEncrypt(symKey, privateKey),
		}).Error
		if err != nil {
			return err
		}

		// TODO: need to change security_stamp ?
		userOrg := model.UsersOrganization{
			UserUUID: user.UUID,
		}
		err = tx.Model(&userOrg).Updates(model.UsersOrganization{
			Akey: pkcs.BWPKEncrypt(m.orgSymKey, pubInf.(*rsa.PublicKey)),
		}).Error
		if err != nil {
			return err
		}

		return nil
	})
}
