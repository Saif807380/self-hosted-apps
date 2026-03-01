package controller

import (
	"fmt"

	"github.com/google/uuid"
)

func parseUUID(s string) (uuid.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("invalid uuid %q: %w", s, err)
	}
	return u, nil
}

func parseUUIDs(ss []string) ([]uuid.UUID, error) {
	result := make([]uuid.UUID, 0, len(ss))
	for _, s := range ss {
		u, err := parseUUID(s)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, nil
}

func int16PtrFromInt32(v *int32) *int16 {
	if v == nil {
		return nil
	}
	r := int16(*v)
	return &r
}

func int32SliceFromInt16(s []int16) []int32 {
	r := make([]int32, len(s))
	for i, v := range s {
		r[i] = int32(v)
	}
	return r
}

func int16SliceFromInt32(s []int32) []int16 {
	r := make([]int16, len(s))
	for i, v := range s {
		r[i] = int16(v)
	}
	return r
}
