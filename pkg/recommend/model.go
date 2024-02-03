package recommend

import (
	"encoding/json"
	"gorm.io/gorm"
)

type Wine struct {
	gorm.Model
	Color            string `gorm:"size:255"`
	Sugar            string `gorm:"size:255"`
	Strength         string `gorm:"size:255"`
	Photo            string `gorm:"size:255"`
	Name             string `gorm:"size:255"`
	Article          string `gorm:"size:255;unique"`
	RealName         string `gorm:"size:255"`
	Year             string `gorm:"size:255"`
	Country          string `gorm:"size:255"`
	Region           string `gorm:"size:255"`
	Manufacturer     string `gorm:"size:255"`
	Grape            string `gorm:"size:255"`
	Price            float64
	Body             string `gorm:"size:255"`
	SmellDescription string `db:"smell_description"`
	TasteDescription string `db:"taste_description"`
	FoodDescription  string `db:"food_description"`
	Style            string
	Recommend        string `db:"recommend"`
	Type             string `gorm:"size:255"`
}

type WineFilter struct {
	Color, Sugar, Country string
}

func (w Wine) String() string {
	wineJson, err := json.Marshal(w)
	if err != nil {
		return w.Name
	}

	return string(wineJson)
}
