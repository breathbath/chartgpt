package recommend

import (
	"breathbathChatGPT/pkg/monitoring"
	"context"
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

func (wp *WineProvider) FindByCriteria(ctx context.Context, f *WineFilter) (found bool, w Wine, err error) {
	query := wp.conn.Model(&Wine{})

	if f.Color != "" {
		query.Where("color = ?", f.Color)
	}

	if f.Country != "" {
		query.Where("country = ?", f.Country)
	}

	if f.Sugar != "" {
		query.Where("sugar = ?", f.Sugar)
	}

	if f.Body != "" {
		query.Where("body = ?", f.Body)
	}

	if f.Year > 0 {
		query.Where("year = ?", f.Year)
	}

	if f.PriceRange != nil {
		if f.PriceRange.From > 0 {
			query.Where("price >= ?", f.PriceRange.From)
		}
		if f.PriceRange.To > 0 {
			query.Where("price <= ?", f.PriceRange.To)
		}
	}

	query.Order("RAND()").Order("photo DESC")

	var wine Wine

	sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.First(&wine)
	})
	monitoring.TrackRecommend(ctx).SetDBQuery(sql)

	res := query.First(&wine)

	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, w, nil
	}
	if res.Error != nil {
		return false, w, err
	}

	monitoring.TrackRecommend(ctx).SetRecommendedWineID(wine.Article)
	monitoring.TrackRecommend(ctx).SetRecommendedWineSummary(wine.SummaryStr())

	return true, wine, nil
}
