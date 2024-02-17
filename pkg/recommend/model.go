package recommend

import (
	"breathbathChatGPT/pkg/monitoring"
	"breathbathChatGPT/pkg/utils"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"strings"
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
	Type, Taste string
	Year int
	AlcoholPercentage,
	PriceRange *utils.RangeFloat
	MatchingDishes,
	Style []string
}

var StylesEnaum = []string{"минеральные", "травянистые", "пряные", "пикантные", "ароматные", "фруктовые", "освежающие", "десертные", "выдержанные", "бархатистые"}

func (wf WineFilter) GetEmptyPrimaryFilters() []string {
	filterNames := []string{}
	if wf.Color == "" {
		filterNames = append(filterNames, "цвет")
	}

	if wf.Country == "" {
		filterNames = append(filterNames, "страна")
	}

	if wf.Sugar == "" {
		filterNames = append(filterNames, "сахар")
	}

	return filterNames
}

func (wf WineFilter) GetRandomSecondaryFilters() []string {
	emptyFilters := wf.GetEmptySecondaryFilters()
	return utils.GetRandomItems(emptyFilters)
}

func (wf WineFilter) GetEmptySecondaryFilters() []string {
	filterNames := []string{}
	if wf.Grape == "" {
		filterNames = append(filterNames, "виноград")
	}

	if len(wf.Style) == 0 {
		filterNames = append(filterNames, "стиль: "+strings.Join(StylesEnaum, ","))
	}

	if wf.Body == "" {
		filterNames = append(filterNames, "тело")
	}

	if wf.Taste == "" {
		filterNames = append(filterNames, "тело")
	}

	if len(wf.MatchingDishes) == 0 {
		filterNames = append(filterNames, "подходящие блюда")
	}

	if wf.AlcoholPercentage.IsEmpty() {
		filterNames = append(filterNames, "крепость")
	}

	return filterNames
}

func (wf WineFilter) GetPrimaryFiltersCount() int {
	count := 0

	if wf.Color != "" {
		count++
	}

	if wf.Country != "" {
		count++
	}

	if len(wf.Sugar) > 0 {
		count++
	}

	return count
}

func (wf WineFilter) GetTotalPrimaryFiltersCount() int {
	return 3
}

func (wf WineFilter) HasSecondaryFilters() bool {
	return wf.Grape != "" || len(wf.Style) != 0 || wf.Taste != "" || wf.Body != "" || len(wf.MatchingDishes) > 0 || !wf.AlcoholPercentage.IsEmpty()
}

func (wf WineFilter) HasExpertFilters() bool {
	return wf.Year > 0 || wf.Region != ""
}

func (wf WineFilter) String() string {
	wineJson, err := json.Marshal(wf)
	if err != nil {
		return fmt.Sprint(wf)
	}

	return string(wineJson)
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
	UserLogin        string
	WineID           int
	Wine             Wine `gorm:"constraint:OnDelete:CASCADE;"`
	RecommendationID *uint
	Recommendation   *monitoring.Recommendation `gorm:"constraint:OnDelete:SET NULL;"`
}
