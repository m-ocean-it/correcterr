package pkg

import (
	"errors"
	"fmt"
)

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

func NilError() error {
	var someError error
	var anotherError error

	if someError == nil {
		return anotherError
	}

	return nil
}

func LengthOfSlice() error {
	var slice []int
	err := errors.New("empty")
	if len(slice) == 0 {
		return err
	}

	return nil
}

func NewErrorAfterCheck() error {
	var err error
	if err != nil {
		return errors.New("some new error")
	}

	return nil
}

func IfTrue() error {
	var err error
	if true {
		return err
	}

	return nil
}

func CompareNumbers() error {
	a := 2
	b := 3
	var err error

	if a != b {
		return err
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

func ErrorfWrap2() error {
	err1 := errors.New("1")
	err2 := errors.New("2")

	if err1 != nil {
		return fmt.Errorf("errors: %w, %w", err1, err2)
	}

	return nil
}

func Closure() {
	var err error
	func() error {
		if innerErr := errors.New("inner"); innerErr != nil {
			return err // want "returning not the error that was checked"
		}

		return nil
	}()
}

func DoubleWrap() error {
	var err error
	if err != nil {
		return fmt.Errorf("error: %w", fmt.Errorf("error: %w", err))
	}

	return nil
}

func TripleFooWrap() error {
	var err error
	if err != nil {
		return fooWrap(1, fooWrap(2, fooWrap(3, err, "c"), "b"), "a")
	}

	return nil
}

func ReturningWrappedMessage() error {
	err := errors.New("some error")
	if err != nil {
		return fooWrap(1, errors.New("new error"), err.Error())
	}

	return nil
}

func fooWrap(_ int, err error, _ string) error {
	return err
}
