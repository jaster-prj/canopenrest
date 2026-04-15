package common

import (
	"reflect"
)

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
