# VaultWarden Bulk User Creator

A simple CLI tool for batch-creating users in [VaultWarden](https://github.com/dani-garcia/vaultwarden), the lightweight Bitwarden-compatible password manager. This tool assumes that there is only one organization in a VaultWarden site.”

## 🚀 Features

- Create VaultWarden users from RESTful API
- Supports email and password setup
- Interacts directly with the VaultWarden admin API or database
- Ideal for onboarding teams or initializing accounts in self-hosted environments

## 🛠️ Requirements

- A running VaultWarden instance with admin access
- Golang
- Admin token (if using the `/admin` API endpoints)

## Mgr API

### Create User

Create a user with email, name and master password. The created users will be in a confirmed status and assigned a custom role.

Request
```http
POST /api/users/register HTTP/1.1
Content-Type: application/json

{
    "name": "test01",
    "email": "test01@foobar.com",
    "password": "foobarfoobar"
}
```

Response
```json
{
    "status": "ok"
}
```

### Reset User Master Password

Reset the master password of a user by their email. Items in their personal vault are no longer available

Request
```http
POST /api/users/test01@foobar.com/reset HTTP/1.1
Content-Type: application/json

{
    "password": "barfoobarfoo"
}
```

Response
```json
{
    "status": "ok"
}
```

### Org Item List

List all items in the orginzation.

Request
```http
GET /api/org/items HTTP/1.1
```

Response
```
[
    {
        "email": "user01@foobar.com",
        "collection_id": "c79d5f48-1f9c-4be4-8a60-2c0e7d123f33",
        "collection_name": "SaaS Services",
        "item_uuid": "a3f1d2b0-89a1-4c9f-9152-d58c5c8b9bfa",
        "item_name": "FB Account",
        "account_name": "login_fb@foobar.com",
        "access": "manage",
        "created_at": "2025-03-26T03:42:01.315141Z",
        "updated_at": "2025-03-26T08:42:01.38078Z"
    },
    {
        "email": "user02@foobar.com",
        "collection_id": "0f2b91e4-87cf-4424-8320-81c957b71d91",
        "collection_name": "DB Accounts",
        "item_uuid": "d14f32a9-b7e8-4cf2-b82a-182e94a2b62a",
        "item_name": "mysql account",
        "account_name": "db_user_003",
        "access": "edit",
        "created_at": "2025-03-25T09:08:46.029018Z",
        "updated_at": "2025-03-25T09:10:46.099916Z"
    },
    {
        "email": "user32@foobar.com",
        "collection_id": "0f2b91e4-87cf-4424-8320-81c957b71d91",
        "collection_name": "Payment",
        "item_uuid": "d14f32a9-b7e8-4cf2-b82a-182e94a2b62a",
        "item_name": "bank_01",
        "account_name": "AABBCCDD",
        "access": "view",
        "created_at": "2025-03-26T03:42:01.315141Z",
        "updated_at": "2025-03-26T05:12:03.18078Z"
    },
]
```

### User Depart Report

List all collections the departing user belongs to, along with other users who have permission to modify their contents

Request
```http
GET /api/users/test01@foobar.com/depart_report HTTP/1.1
```

Response
```json
[
    {
        "collection_uuid": "ffffffff-3333-4444-aaaa-bbbbbbbbbbbb",
        "collection_name": "foolbar",
        "email": "user01@foobar.com"
    },
    {
        "collection_uuid": "eeeeeeee-2222-3333-bbbb-cccccccccccc",
        "collection_name": "barfoo",
        "collection_name": "foolbar",
    }
]

```
