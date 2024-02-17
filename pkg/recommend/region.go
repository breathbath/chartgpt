package recommend

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

func FindCountryByRegion(db *gorm.DB, region string) (country string, err error) {
	var w Wine
	res := db.Where("region LIKE ? AND country != ''", "%"+region+"%").First(&w)
	if res.Error == nil {
		return w.Country, nil
	}

	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return "", nil
	}

	return "", res.Error
}
