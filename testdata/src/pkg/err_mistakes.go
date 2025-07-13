package pkg

import (
	"errors"
)

// ----------------------------------------------------
// Triggers

func CheckingAndReturningDifferentErrors() error {
	var err1 = errors.New("1")
	var err2 = errors.New("2")

	if err1 != nil {
		return err2 // want "returning not the error that was checked"
	}

	return nil
}

// ----------------------------------------------------
// Non-triggers

func Correct() error {
	var err1 = errors.New("1")
	var err2 = errors.New("2")

	if err1 != nil {
		return err1
	}

	if err2 != nil {
		return err2
	}

	return nil
}
