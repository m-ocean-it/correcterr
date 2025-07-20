package pkg

import (
	"errors"
	"fmt"
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

func CheckingAndReturningDifferentErrors2() error {
	var err1 = errors.New("1")
	if err2 := errors.New("2"); err2 != nil {
		return err1 // want "returning not the error that was checked"
	}

	return nil
}

func ErrorfWrap() error {
	err1 := errors.New("1")
	err2 := errors.New("2")

	if err1 != nil {
		return fmt.Errorf("error: %w", err2) // want "returning not the error that was checked"
	}

	return nil
}

func FuncLit() {
	var err error

	func() error {
		if innerErr := errors.New("inner"); innerErr != nil {
			return err // want "returning not the error that was checked"
		}

		return nil
	}()
}

func AssignFuncLit() error {
	var err error

	funcLitErr := func() error {
		if innerErr := errors.New("inner"); innerErr != nil {
			return err // want "returning not the error that was checked"
		}

		return nil
	}()

	return funcLitErr
}

func NestedIfStatements() error {
	err := errors.New("error")
	anotherErr := errors.New("another")

	if true {
		if err != nil {
			return anotherErr // want "returning not the error that was checked"
		}
	}

	return nil
}

func TripleFooWrapOfWrongError() error {
	err := errors.New("error")
	anotherError := errors.New("another")

	if err != nil {
		return fooWrap(1, fooWrap(2, fooWrap(3, anotherError, "c"), "b"), "a") // want "returning not the error that was checked"
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

// ----------------------------------------------------
// Helpers

func fooWrap(_ int, err error, _ string) error {
	return err
}
