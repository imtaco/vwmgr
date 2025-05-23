package mgr

import (
	"github.com/imtaco/vwmgr/pkg/model"
	"github.com/pkg/errors"
)

type leaveUserItem struct {
	OrgUUID        string `json:"org_uuid"`
	OrgName        string `json:"org_name"`
	CollectionUUID string `json:"collection_uuid"`
	CollectionName string `json:"collection_name"`
	Email          string `json:"email"`
}

func (m *VMManager) userDepartReport(email string) ([]leaveUserItem, error) {
	var items []leaveUserItem

	// check user first
	user := model.User{}
	if err := m.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}

	sql := `
	WITH depart_user_collections AS (
		SELECT
			collection_uuid
		FROM
			users_collections_expands
		WHERE
			user_uuid IN (SELECT uuid FROM users WHERE email = ?)
	),
	users_collections_detail AS (
		SELECT
			uce.collection_uuid,
			u.email
		FROM
			users_collections_expands uce
			INNER JOIN users u ON u.uuid = uce.user_uuid
		WHERE
			(uce.manage = TRUE OR uce.read_only = FALSE)
			AND u.email != ?
			AND uce.user_org_status = 2
	)
	SELECT
		c.org_uuid AS org_uuid,
		o.name as org_name,
		c.uuid AS collection_uuid,
		c.name AS collection_name,
		ucd.email
	FROM
		depart_user_collections ubc
		INNER JOIN collections c ON c.uuid = ubc.collection_uuid
		INNER JOIN organizations o ON o.uuid = c.org_uuid
		LEFT JOIN users_collections_detail ucd ON ucd.collection_uuid = c.uuid
	ORDER BY
		1
	`

	if err := m.db.Raw(sql, email, email).Scan(&items).Error; err != nil {
		return nil, errors.Wrap(err, "fail to query item details")
	}
	return items, nil
}
