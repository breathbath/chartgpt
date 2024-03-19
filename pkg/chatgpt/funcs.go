package chatgpt

import "breathbathChatGPT/pkg/recommend"

var (
	// see https://json-schema.org/understanding-json-schema for details
	findWineFunction = map[string]interface{}{
		"name":        "find_wine",
		"description": "Find wine by provided parameters",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"цвет": map[string]interface{}{
					"type": "string",
					"enum": colors,
				},
				"год": map[string]interface{}{
					"type": "number",
				},
				"сахар": map[string]interface{}{
					"type": "string",
					"enum": sugars,
				},
				"крепость": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "number",
					},
				},
				"подходящие блюда": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{
							"аперитив", "баранина", "блюда", "вегетарианская", "говядина", "грибы", "десерт", "дичь", "закуски", "курица", "морепродукты", "мясные", "овощи", "оливки", "острые", "паста", "пернатая", "ракообразные", "рыба", "свинина", "суши", "сыр", "телятина", "фрукты", "фуа-гра", "ягнятина"},
					},
				},
				"тело": map[string]interface{}{
					"type": "string",
					"enum": bodies,
				},
				"название": map[string]interface{}{
					"description": "Название вина",
					"type":        "string",
				},
				"вкус и аромат": map[string]interface{}{
					"type": "string",
				},
				"страна": map[string]interface{}{
					"type": "string",
				},
				"регион": map[string]interface{}{
					"type": "string",
				},
				"виноград": map[string]interface{}{
					"description": "сорт винограда",
					"type":        "string",
				},
				"тип": map[string]interface{}{
					"type": "string",
					"enum": types,
				},
				"стиль": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "string",
						"enum": recommend.StylesEnaum,
					},
				},
				"цена": map[string]interface{}{
					"type": "array",
					"items": map[string]interface{}{
						"type": "number",
					},
				},
			},
		},
	}
	feedbackRecommendationFunction = map[string]interface{}{
		"name":        "feedback_wine_recommendation",
		"description": "Give a feedback to a wine recommendation",
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"is_positive_feedback": map[string]interface{}{
					"type": "boolean",
				},
				"feedback_text": map[string]interface{}{
					"type": "string",
				},
			},
		},
	}
)
