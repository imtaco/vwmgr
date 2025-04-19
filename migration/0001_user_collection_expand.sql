-- +goose Up
CREATE VIEW users_collections_expands AS
    WITH user_collections_details AS (
        SELECT
            uc.collection_uuid,
            uc.user_uuid,
            uc.manage,
            uc.read_only,
            uo.status AS user_org_status
        FROM
            users_collections uc
            INNER JOIN collections c ON uc.collection_uuid = c.uuid
            INNER JOIN users_organizations uo ON uo.org_uuid = c.org_uuid AND uo.user_uuid = uc.user_uuid
    ),
    group_collections_details AS (
        SELECT
            cg.collections_uuid,
            uo.user_uuid,
            cg.manage,
            cg.read_only,
            uo.status AS user_org_status
        FROM
            collections_groups cg
            INNER JOIN groups_users gu ON cg.groups_uuid = gu.groups_uuid
            INNER JOIN users_organizations uo ON uo.uuid = gu.users_organizations_uuid
    )
    SELECT
        collection_uuid,
        user_uuid,
        max(user_org_status) AS user_org_status,
        bool_or(manage) manage,
        bool_and(read_only) read_only

    FROM (
        (SELECT * FROM user_collections_details)
        UNION ALL
        (SELECT * FROM group_collections_details)
    )
    GROUP BY
        1, 2;

-- +goose Down
DROP VIEW users_collections_expands;
