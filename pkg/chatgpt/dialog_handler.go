package chatgpt

var requiredFilters = []string{
	"цена", "цвет", "страна", "стиль",
}

var usualNotRequiredFilters = []string{
	"виноград", "сахар", "тело", "подходящие блюда", "крепость",
}

var specialNotRequiredFilters = []string{
	"год", "регион",
}

var synonyms = map[string]string{
	"виноград":         "цвет",
	"подходящие блюда": "цвет",
}

var occasionTypes = map[string]string{
	"smallGroup": "Для романтического свидания",
	"bigGroup":   "Для вечеринки",
}
