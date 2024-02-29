package recommend

import (
	"breathbathChatGPT/pkg/monitoring"
	"context"
	"fmt"
	"gorm.io/gorm"
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
	limit int,
	excludeWineIds []uint,
) (wines []Wine, err error) {
	where := []string{
		"`deleted_at` IS NULL",
	}
	whereParams := []interface{}{}
	orderParams := []interface{}{}
	order := []string{}
	q := "SELECT * FROM wines WHERE $where$ ORDER BY $order$ LIMIT %d"
	q = fmt.Sprintf(q, limit)

	if f.Color != "" {
		where = append(where, "AND color = ?")
		whereParams = append(whereParams, f.Color)
	}

	if f.Country != "" {
		where = append(where, "AND country = ?")
		whereParams = append(whereParams, f.Country)
	}

	if f.Sugar != "" {
		where = append(where, "AND sugar = ?")
		whereParams = append(whereParams, f.Sugar)
	}

	if f.Body != "" {
		if f.Body != "полнотелое" {
			where = append(where, "AND body != ?")
			whereParams = append(whereParams, "полнотелое")
		} else {
			where = append(where, "AND body = ?")
			whereParams = append(whereParams, f.Body)
		}
	}

	if f.Type != "" {
		where = append(where, "AND type LIKE ?")
		whereParams = append(whereParams, "%"+f.Type+"%")
	} else {
		where = append(where, "AND type = ?")
		whereParams = append(whereParams, DefaultWineType)
	}

	if f.Region != "" {
		where = append(where, "AND region LIKE ?")
		whereParams = append(whereParams, "%"+f.Region+"%")
	}

	if f.Grape != "" {
		where = append(where, "AND grape LIKE ?")
		whereParams = append(whereParams, "%"+f.Grape+"%")
	}

	if f.Year > 0 {
		where = append(where, "AND year = ?")
		whereParams = append(whereParams, f.Year)
	}

	if f.PriceRange.IsEmpty() {
		order = append(order, "CASE WHEN price BETWEEN ? AND ? THEN 0 ELSE 1 END")
		orderParams = append(orderParams, DefaultPriceRangeFrom)
		orderParams = append(orderParams, DefaultPriceRangeTo)
	} else {
		if f.PriceRange.From > 0 {
			where = append(where, "AND price >= ?")
			whereParams = append(whereParams, f.PriceRange.From)
		}
		if f.PriceRange.To > 0 {
			where = append(where, "AND price <= ?")
			whereParams = append(whereParams, f.PriceRange.To)
		}
	}

	if !f.AlcoholPercentage.IsEmpty() {
		if f.AlcoholPercentage.From > 0 {
			where = append(where, "AND alcohol_percentage >= ?")
			whereParams = append(whereParams, f.AlcoholPercentage.From)
		}
		if f.AlcoholPercentage.To > 0 {
			where = append(where, "AND alcohol_percentage <= ?")
			whereParams = append(whereParams, f.AlcoholPercentage.To)
		}
	}

	if len(f.Style) > 0 {
		for i := range f.Style {
			where = append(where, "AND style LIKE ?")
			whereParams = append(whereParams, "%"+f.Style[i]+"%")
		}
	}

	if f.Name != "" {
		where = append(where, "AND MATCH(name, real_name) AGAINST (?)")
		whereParams = append(whereParams, f.Name)
		order = append(order, "MATCH(name, real_name) AGAINST(?) DESC")
		orderParams = append(orderParams, f.Name)
	}

	if f.Taste != "" {
		where = append(where, "AND MATCH(smell_description, taste_description) AGAINST (?)")
		whereParams = append(whereParams, f.Taste)
		order = append(order, "MATCH(smell_description, taste_description) AGAINST(?) DESC")
		orderParams = append(orderParams, f.Taste)
	}
	order = append(order, "RAND()", "photo DESC")

	if len(f.MatchingDishes) > 0 {
		dishes := strings.Join(f.MatchingDishes, " ")
		where = append(where, "AND MATCH(recommend) AGAINST (?)")
		whereParams = append(whereParams, dishes)

		order = append(order, `MATCH(recommend) AGAINST(?) DESC, RAND()`)
		orderParams = append(orderParams, dishes)
	}

	if len(excludeWineIds) > 0 {
		where = append(where, "AND id not IN ?")
		whereParams = append(whereParams, excludeWineIds)
	}

	q = strings.ReplaceAll(q, "$where$", strings.Join(where, " "))
	q = strings.ReplaceAll(q, "$order$", strings.Join(order, ", "))

	allParams := []interface{}{}
	allParams = append(allParams, whereParams...)
	allParams = append(allParams, orderParams...)

	res := wp.conn.Raw(q, allParams...).Find(&wines)

	if recommendStats != nil {
		recommendStats.DBQuery = wp.conn.Dialector.Explain(q, allParams...)
	}

	if res.Error != nil {
		return nil, err
	}

	return wines, nil
}
