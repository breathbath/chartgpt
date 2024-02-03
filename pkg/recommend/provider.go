package recommend

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type WineProvider struct {
	conn *gorm.DB
}

func NewWineProvider(conn *gorm.DB) *WineProvider {
	return &WineProvider{
		conn: conn,
	}
}

func (wp *WineProvider) FindByCriteria(f WineFilter) (found bool, w Wine, err error) {
	filters := make(map[string]interface{})
	if f.Color != "" {
		filters["color"] = f.Color
	}

	if f.Country != "" {
		filters["country"] = f.Country
	}

	if f.Sugar != "" {
		filters["sugar"] = f.Sugar
	}

	var wine Wine
	err = wp.conn.
		Where(filters).
		Order("RAND()").
		Order("photo DESC").
		First(&wine).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, w, nil
	}
	if err != nil {
		return false, w, err
	}

	return true, wine, nil
}
