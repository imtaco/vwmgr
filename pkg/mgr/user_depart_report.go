package mgr

import (
	"github.com/pkg/errors"
)

type leaveUserItem struct {
	CollectionUUID string `json:"collection_uuid"`
	CollectionName string `json:"collection_name"`
	Email          string `json:"email"`
}

func (m *VMManager) userDepartReport(email string) ([]leaveUserItem, error) {
	var items []leaveUserItem

	sql := `
	WITH user_belong_collections AS (
		SELECT
			DISTINCT collection_uuid
		FROM
			users_collections
		WHERE
			user_uuid = (SELECT uuid FROM users WHERE email = ?)
	),
	users_collections_detail AS (
		SELECT
			uc.collection_uuid,
			u.email
		FROM
			users_collections uc
			INNER JOIN users u ON u.uuid = uc.user_uuid
		WHERE
			(uc.manage = TRUE OR uc.read_only = FALSE)
			AND u.email != ?
	)
	SELECT
		c.uuid AS collection_uuid,
		c.name AS collection_name,
		ucd.email
	FROM
		user_belong_collections ubc
		INNER JOIN collections c ON c.uuid = ubc.collection_uuid
		LEFT JOIN users_collections_detail ucd ON ucd.collection_uuid = c.uuid
	WHERE
		c.org_uuid = ?
	ORDER BY
		1
	`

	if err := m.db.Raw(sql, email, email, m.orgUUID).Scan(&items).Error; err != nil {
		return nil, errors.Wrap(err, "fail to query item details")
	}
	return items, nil
}
