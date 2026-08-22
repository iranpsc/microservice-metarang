package handler

import (
	"strconv"
	"strings"
)

func parseWalletLoginQuery(raw string) bool {
	value, err := strconv.ParseBool(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return value
}
