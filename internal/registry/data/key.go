package data

import (
	mathrand "math/rand"

	"gorm.io/gorm"

	"github.com/infrahq/infra/internal/registry/models"
)

func CreateKey(db *gorm.DB, key *models.Key) (*models.Key, error) {
	if key.KeyID == 0 {
		// not a security issue; just an identifier
		key.KeyID = mathrand.Int31() // nolint:gosec
	}

	if err := add(db, &models.Key{}, key, nil); err != nil {
		return nil, err
	}

	return key, nil
}

func GetKey(db *gorm.DB, selector SelectorFunc) (result *models.Key, err error) {
	result = &models.Key{}

	if err := get(db, &models.Key{}, result, selector(db)); err != nil {
		return nil, err
	}

	return result, nil
}

func ByKeyID(keyID int32) SelectorFunc {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("key_id = ?", keyID)
	}
}