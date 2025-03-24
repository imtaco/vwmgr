package mgr

import (
	"github.com/google/uuid"
	"github.com/imtaco/vwmgr/model"
	"github.com/imtaco/vwmgr/pkcs"
	"gorm.io/gorm"
)

func (m *VMManager) Register(
	email string,
	name string,
	masterPassword string,
) error {
	userMasterKey := pkcs.DeriveMasterKey(email, masterPassword)
	passwordHash := pkcs.DerivePasswordHash(userMasterKey, masterPassword)

	salt := pkcs.RandBytes(64)
	hashPwdHash := pkcs.HashPasswordHash(passwordHash, salt)

	symKey := pkcs.RandBytes(64)
	userAkey := pkcs.BWSymEncrypt(userMasterKey, symKey)

	uid := uuid.NewString()
	publicKey, privateKey := pkcs.GenRSAKeyPair()

	pubInf, err := pkcs.PublicKeyInfo(publicKey)
	if err != nil {
		return err
	}

	return m.db.Transaction(func(tx *gorm.DB) error {
		user := model.User{
			UUID:               uid,
			Name:               name,
			Email:              email,
			PasswordHash:       hashPwdHash,
			PasswordIterations: ITERATIONS,
			Salt:               salt,
			Akey:               userAkey,
			PublicKey:          pkcs.Base64Encode(publicKey),
			PrivateKey:         pkcs.BWSymEncrypt(symKey, privateKey),
			EquivalentDomains:  "[]",
			ExcludedGlobals:    "[]",
			SecurityStamp:      uuid.NewString(),
			ClientKdfIter:      ITERATIONS,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		userOrg := model.UsersOrganization{
			UUID:      uuid.NewString(),
			UserUUID:  uid,
			OrgUUID:   m.orgUUID,
			Akey:      pkcs.BWPKEncrypt(m.orgSymKey, pubInf),
			AccessAll: false,
			Status:    2,
			Atype:     3,
		}
		if err := tx.Create(&userOrg).Error; err != nil {
			return err
		}

		return nil
	})
}
