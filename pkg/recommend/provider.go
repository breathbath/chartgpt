package recommend

import (
	"breathbathChatGPT/pkg/monitoring"
	"context"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

const (
	DefaultPriceRangeFrom = 1000
	DefaultPriceRangeTo   = 1500
	DefaultWineType       = "Вино"
)

type WineProvider struct {
	conn *gorm.DB
}

func NewWineProvider(conn *gorm.DB) *WineProvider {
	return &WineProvider{
		conn: conn,
	}
}

func (wp *WineProvider) FindByCriteria(
	ctx context.Context,
	f *WineFilter,
	recommendStats *monitoring.Recommendation,
) (found bool, w Wine, err error) {
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
		if f.Body != "полнотелое" {
			query.Where("body != ?", "полнотелое")
		} else {
			query.Where("body = ?", f.Body)
		}
	}

	if f.Type != "" {
		query.Where("type = ?", f.Type)
	} else {
		query.Where("type = ?", DefaultWineType)
	}

	if f.Region != "" {
		query.Where("region LIKE ?", "%"+f.Region+"%")
	}

	if f.Grape != "" {
		query.Where("grape LIKE ?", "%"+f.Grape+"%")
	}

	if f.Year > 0 {
		query.Where("year = ?", f.Year)
	}

	if f.PriceRange.IsEmpty() {
		query.Where("price >= ?", DefaultPriceRangeFrom)
		query.Where("price <= ?", DefaultPriceRangeTo)
	} else {
		if f.PriceRange.From > 0 {
			query.Where("price >= ?", f.PriceRange.From)
		}
		if f.PriceRange.To > 0 {
			query.Where("price <= ?", f.PriceRange.To)
		}
	}

	if !f.AlcoholPercentage.IsEmpty() {
		if f.AlcoholPercentage.From > 0 {
			query.Where("alcohol_percentage >= ?", f.AlcoholPercentage.From)
		}
		if f.AlcoholPercentage.To > 0 {
			query.Where("alcohol_percentage <= ?", f.AlcoholPercentage.To)
		}
	}

	if len(f.Style) > 0 {
		for i := range f.Style {
			query.Where("style LIKE ?", "%"+f.Style[i]+"%")
		}
	}

	if f.Name != "" {
		query.Where("MATCH(name, real_name) AGAINST (?)", f.Name)
		query.Clauses(clause.OrderBy{
			Expression: clause.Expr{SQL: `MATCH(name, real_name) AGAINST(?) DESC, RAND()`, Vars: []interface{}{f.Name}},
		})
	}

	if f.Name == "" {
		query.Order("RAND()").Order("photo DESC")
	}

	if len(f.MatchingDishes) > 0 {
		dishes := strings.Join(f.MatchingDishes, " ")
		query.Where("MATCH(recommend) AGAINST (?)", dishes)
		query.Clauses(clause.OrderBy{
			Expression: clause.Expr{SQL: `MATCH(recommend) AGAINST(?) DESC, RAND()`, Vars: []interface{}{dishes}},
		})
	}

	var wine Wine

	sql := query.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Take(&wine)
	})
	recommendStats.DBQuery = sql

	res := query.Take(&wine)

	if errors.Is(res.Error, gorm.ErrRecordNotFound) {
		return false, w, nil
	}
	if res.Error != nil {
		return false, w, err
	}

	return true, wine, nil
}
