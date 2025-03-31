package mgr

import (
	"github.com/google/uuid"
	"github.com/imtaco/vwmgr/pkg/model"
	"github.com/imtaco/vwmgr/pkg/pkcs"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

const (
	roleOwner  = 0
	roleAdmin  = 1
	roleUser   = 2
	roleCustom = 3
)

func (m *VMManager) createUser(
	email string,
	name string,
	masterPassword string,
	org2role map[string]int32,
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

	// check orgSymKey first
	for orgUUID := range org2role {
		if _, ok := m.orgSymKeys[orgUUID]; !ok {
			return errors.Errorf("fail to found orr symmetric key of %s", orgUUID)
		}
	}

	return m.db.Transaction(func(tx *gorm.DB) error {
		user := model.User{
			UUID:               uid,
			Name:               name,
			Email:              email,
			PasswordHash:       hashPwdHash,
			PasswordIterations: pkcs.ITERATIONS,
			Salt:               salt,
			Akey:               userAkey,
			PublicKey:          pkcs.Base64Encode(publicKey),
			PrivateKey:         pkcs.BWSymEncrypt(symKey, privateKey),
			EquivalentDomains:  "[]",
			ExcludedGlobals:    "[]",
			SecurityStamp:      uuid.NewString(),
			ClientKdfIter:      pkcs.ITERATIONS,
		}
		if err := tx.Create(&user).Error; err != nil {
			return err
		}

		for orgUUID, role := range org2role {
			userOrg := model.UsersOrganization{
				UUID:      uuid.NewString(),
				UserUUID:  uid,
				OrgUUID:   orgUUID,
				Akey:      pkcs.BWPKEncrypt(m.orgSymKeys[orgUUID], pubInf),
				AccessAll: false,
				Status:    2,
				Atype:     role,
			}
			if err := tx.Create(&userOrg).Error; err != nil {
				return err
			}
		}

		return nil
	})
}
