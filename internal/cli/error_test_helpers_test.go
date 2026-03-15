package cli

import "errors"

func assertError(msg string) error {
	return errors.New(msg)
}
