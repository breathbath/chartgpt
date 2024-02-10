package recommend

import (
	"breathbathChatGPT/pkg/utils"
	"encoding/json"
	"gorm.io/gorm"
)

type Wine struct {
	gorm.Model
	Color             string `gorm:"size:255"`
	Sugar             string `gorm:"size:255"`
	Photo             string `gorm:"size:255"`
	Name              string `gorm:"size:255"`
	Article           string `gorm:"size:255;unique"`
	RealName          string `gorm:"size:255"`
	Year              string `gorm:"size:255"`
	Country           string `gorm:"size:255"`
	Region            string `gorm:"size:255"`
	Manufacturer      string `gorm:"size:255"`
	Grape             string `gorm:"size:255"`
	Price             float64
	AlcoholPercentage float64
	Body              string `gorm:"size:255"`
	SmellDescription  string
	TasteDescription  string
	FoodDescription   string
	Style             string
	Recommend         string `db:"recommend"`
	Type              string `gorm:"size:255"`
}

type WineSummary struct {
	Color             string
	Sugar             string
	Name              string
	Article           string
	RealName          string
	Year              string
	Country           string
	Region            string
	Manufacturer      string
	Grape             string
	Price             float64
	AlcoholPercentage float64
	Body              string
	Style             string
	Recommend         string
	Type              string
}

type WineTextualSummary struct {
	Color             string  `json:"Цвет"`
	Sugar             string  `json:"Сахар"`
	Name              string  `json:"Название лат"`
	RealName          string  `json:"Название рус"`
	Year              string  `json:"Год"`
	Country           string  `json:"Страна"`
	Region            string  `json:"Регион"`
	Manufacturer      string  `json:"Производитель"`
	Grape             string  `json:"Сорт винограда"`
	AlcoholPercentage float64 `json:"Крепость"`
	Body              string  `json:"Тело"`
	SmellDescription  string  `json:"Аромат"`
	TasteDescription  string  `json:"Вкус"`
	FoodDescription   string  `json:"Гастрономия"`
	Style             string  `json:"Стиль"`
	Recommend         string  `json:"Рекомендованные блюда"`
	Type              string  `json:"Тип напитка"`
}

type WineFilter struct {
	Color,
	Sugar,
	Country,
	Body,
	Name,
	Region,
	Grape,
	Type string
	Year int
	AlcoholPercentage,
	PriceRange *utils.RangeFloat
	MatchingDishes,
	Style []string
}

func (w Wine) String() string {
	wineJson, err := json.Marshal(w)
	if err != nil {
		return w.Name
	}

	return string(wineJson)
}

func (w Wine) Summary() WineSummary {
	return WineSummary{
		Color:             w.Color,
		Sugar:             w.Sugar,
		Name:              w.Name,
		Article:           w.Article,
		RealName:          w.RealName,
		Year:              w.Year,
		Country:           w.Country,
		Region:            w.Region,
		Manufacturer:      w.Manufacturer,
		Grape:             w.Grape,
		Price:             w.Price,
		AlcoholPercentage: w.AlcoholPercentage,
		Body:              w.Body,
		Style:             w.Style,
		Recommend:         w.Recommend,
		Type:              w.Type,
	}
}

func (w Wine) WineTextualSummary() WineTextualSummary {
	return WineTextualSummary{
		Color:             w.Color,
		Sugar:             w.Sugar,
		Name:              w.Name,
		RealName:          w.RealName,
		Year:              w.Year,
		Country:           w.Country,
		Region:            w.Region,
		Manufacturer:      w.Manufacturer,
		Grape:             w.Grape,
		AlcoholPercentage: w.AlcoholPercentage,
		Body:              w.Body,
		SmellDescription:  w.SmellDescription,
		TasteDescription:  w.TasteDescription,
		FoodDescription:   w.FoodDescription,
		Style:             w.Style,
		Recommend:         w.Recommend,
		Type:              w.Type,
	}
}

func (w Wine) WineTextualSummaryStr() string {
	wineJson, err := json.Marshal(w.WineTextualSummary())
	if err != nil {
		return w.Name
	}

	return string(wineJson)
}

func (w Wine) SummaryStr() string {
	wineJson, err := json.Marshal(w.Summary())
	if err != nil {
		return w.Name
	}

	return string(wineJson)
}

type WineFavorite struct {
	gorm.Model
	UserLogin string
	WineID    int
	Wine      Wine `gorm:"constraint:OnDelete:CASCADE;"`
}
