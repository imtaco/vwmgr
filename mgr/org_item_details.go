package mgr

import (
	"time"

	"github.com/pkg/errors"
)

type orgItemDetail struct {
	Email          string
	CollectionUUID string
	CollectionName string
	CipherName     string
	AccountName    string
	ReadOnly       bool
	Manage         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (m *VMManager) listOrgItems() ([]orgItemDetail, error) {
	var details []orgItemDetail

	sql := `
	SELECT
		u.email,
		c.uuid as collection_uuid,
		c.name as collection_name,
		p.name as cipher_name,
		(p.data::json)->>'username' as account_name,
		uc.read_only,
		uc.manage,
		p.created_at,
		p.updated_at
	FROM
		collections c
		INNER JOIN users_collections uc ON uc.collection_uuid = c.uuid
		INNER JOIN ciphers_collections cc ON cc.collection_uuid = c.uuid
		INNER JOIN users u ON u.uuid = uc.user_uuid
		INNER JOIN ciphers p ON cc.cipher_uuid = p.uuid
	WHERE
		c.org_uuid = ?
		AND p.deleted_at IS NULL
	ORDER BY
		1, 2, 3, 4
	`

	if err := m.db.Raw(sql, m.orgUUID).Scan(&details).Error; err != nil {
		return nil, errors.Wrap(err, "fail to query item details")
	}
	return details, nil
}
