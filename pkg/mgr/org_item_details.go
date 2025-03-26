package mgr

import (
	"time"

	"github.com/pkg/errors"
)

type orgItemDetail struct {
	Email          string    `json:"email"`
	OrgUUID        string    `json:"org_uuid"`
	OrgName        string    `json:"org_name"`
	CollectionUUID string    `json:"collection_uuid"`
	CollectionName string    `json:"collection_name"`
	ItemUUID       string    `json:"item_uuid"`
	ItemName       string    `json:"item_name"`
	AccountName    string    `json:"account_name"`
	Access         string    `json:"access"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (m *VMManager) listOrgItems() ([]orgItemDetail, error) {
	var details []orgItemDetail

	sql := `
	SELECT
		u.email,
		c.org_uuid,
		o.name as org_name,
		c.uuid as collection_uuid,
		c.name as collection_name,
		p.uuid as item_uuid,
		p.name as item_name,
		(p.data::json)->>'username' as account_name,
		CASE
			WHEN uc.manage = TRUE THEN 'manage'
			WHEN uc.read_only = FALSE THEN 'edit'
			ELSE 'view'
		END as access,
		p.created_at,
		p.updated_at
	FROM
		collections c
		INNER JOIN organizations o ON o.uuid = c.org_uuid
		INNER JOIN users_collections uc ON uc.collection_uuid = c.uuid
		INNER JOIN ciphers_collections cc ON cc.collection_uuid = c.uuid
		INNER JOIN users u ON u.uuid = uc.user_uuid
		INNER JOIN ciphers p ON cc.cipher_uuid = p.uuid
	WHERE
		c.org_uuid IS NOT NULL
		AND p.deleted_at IS NULL
	ORDER BY
		1, 2, 3, 4
	`

	if err := m.db.Raw(sql).Scan(&details).Error; err != nil {
		return nil, errors.Wrap(err, "fail to query item details")
	}
	return details, nil
}
