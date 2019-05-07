package helper

import (
	"errors"
	"strconv"

	"github.com/shopspring/decimal"
)

func Uint64FromString(str string) (uint64, error) {
	value, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, err
	}

	return value, nil
}

func Bcsub(a string, b string) (string, error) {
	oldSize, err := decimal.NewFromString(a)
	if err != nil {
		return "", err
	}
	subSize, err := decimal.NewFromString(b)
	if err != nil {
		return "", err
	}
	ret := oldSize.Sub(subSize)

	if ret.LessThan(decimal.Zero) {
		return "", errors.New("a: " + a + ", b: " + b + ", sub result is less than 0")
	}
	return ret.String(), nil
}

func FloatDiffFromString(a string, b string) error {
	if a == b {
		return nil
	}

	aF, err := decimal.NewFromString(a)
	if err != nil {
		return err
	}
	bF, err := decimal.NewFromString(b)
	if err != nil {
		return err
	}

	if !aF.Equal(bF) {
		return errors.New("FloatDiffFromString: " + a + " != " + b)
	}

	return nil
}
