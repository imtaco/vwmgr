package mgr

import "gorm.io/gorm"

const (
	ITERATIONS = 600_000
)

func New(
	orgUUID string,
	orgSymKey []byte,
	db *gorm.DB,
) *VMManager {
	return &VMManager{
		orgUUID:   orgUUID,
		orgSymKey: orgSymKey,
		db:        db,
	}
}

type VMManager struct {
	orgUUID   string
	orgSymKey []byte
	db        *gorm.DB
}
