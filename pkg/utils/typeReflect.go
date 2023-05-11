package utils

import "reflect"

func GetType(target interface{}) string {
	if t := reflect.TypeOf(target); t.Kind() == reflect.Ptr {
		return "*" + t.Elem().Name()
	} else {
		return t.Name()
	}
}
