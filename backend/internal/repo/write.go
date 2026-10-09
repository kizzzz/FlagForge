package repo

import "gorm.io/gorm"

func writeResult(result *gorm.DB) error {
	if result.Error != nil {
		return MapError(result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
