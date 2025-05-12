package common

import (
	"reflect"
)

func POINTER[T any](val T) *T {
	return &val
}

func NEWSTRINGPOINTER[T *string](val T) T {
	if val == nil {
		return nil
	}
	tmp := *val
	return &tmp
}

func MATCHSLICES[T any](slice1 []T, slice2 []T) []T {
	return_slice := []T{}
	for _, value1 := range slice1 {
		for _, value2 := range slice2 {
			if reflect.DeepEqual(value1, value2) {
				return_slice = append(return_slice, value1)
			}
		}
	}
	return return_slice
}

// CONTAINS checks if list contains e
func CONTAINS[C comparable](list []C, e C) bool {
	for _, element := range list {
		if element == e {
			return true
		}
	}
	return false
}
