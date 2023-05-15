package utils

import "reflect"

func GetType(target interface{}) string {
	t := reflect.TypeOf(target)

	if t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	}

	return t.Name()
}
