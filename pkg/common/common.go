package common

import (
	"github.com/imtaco/vwmgr/pkg/model"
	"github.com/imtaco/vwmgr/pkg/pkcs"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func GetOrgSymKeys(
	db *gorm.DB,
	userEmail string,
	userMasterPwd string,
) (map[string][]byte, error) {

	// uuid -> orgSymKey
	result := map[string][]byte{}

	user := model.User{}
	if err := db.Where("email = ?", userEmail).First(&user).Error; err != nil {
		// not found or real error
		return nil, err
	}

	userOrgs := []model.UsersOrganization{}
	if err := db.Where("user_uuid = ?", user.UUID).Find(&userOrgs).Error; err != nil {
		// not found or real error
		return nil, err
	}

	masterKey := pkcs.DeriveMasterKey(userEmail, userMasterPwd)
	symKey, err := pkcs.BWSymDecrypt(masterKey, user.Akey)
	if err != nil {
		return nil, errors.Wrap(err, "fail to decrypt user akey")
	}

	privateKey, err := pkcs.BWSymDecrypt(symKey, user.PrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "fail to decrypt private key")
	}
	priInf, err := pkcs.PrivateKeyInfo(privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "fail to parse private key")
	}

	for _, uo := range userOrgs {
		orgSymKey, err := pkcs.BWPKDecrypt(uo.Akey, priInf)
		if err != nil {
			return nil, errors.Wrap(err, "fail to decrypt org akey")
		}
		result[uo.OrgUUID] = orgSymKey
	}
	return result, nil
}
