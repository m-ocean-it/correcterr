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

func CheckingAndReturningDifferentErrorsNoLintNoCorrecterr() error {
	var err1 = errors.New("1")
	var err2 = errors.New("2")

	if err1 != nil {
		return err2 //nolint:foo,bar // want "returning not the error that was checked"
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

func Switch() error {
	var err error

	switch {
	case false:
	case true:
		if innerErr := errors.New("inner"); innerErr != nil {
			return err // want "returning not the error that was checked"
		}
	}

	return nil
}

func RangeStmt() error {
	var err error

	for range 5 {
		if innerErr := errors.New("inner"); innerErr != nil {
			return err // want "returning not the error that was checked"
		}
	}

	return nil
}

func ForStmt() error {
	err := errors.New("error")

	for i := 0; i < 5; i++ {
		_ = i

		if innerErr := errors.New("inner"); innerErr != nil {
			return err // want "returning not the error that was checked"
		}
	}

	return nil
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

func NoInitialLocalErrNames() {
	closureWrapper(func() error {
		innerErr := errors.New("inner")
		anotherInnerErr := errors.New("another")

		if innerErr != nil {
			return anotherInnerErr // want "returning not the error that was checked"
		}

		return nil
	})
}

func Foobar2() error {
	_, err := doSmth()
	_, err2 := doSmth()

	if err != nil {
		_, errB := doSmth()
		_ = errB

		return err2 // want "returning not the error that was checked"
	}

	return nil
}

// ----------------------------------------------------
// Non-triggers

func ErrorfWrap2() error {
	err1 := errors.New("1")
	err2 := errors.New("2")

	if err1 != nil {
		return fmt.Errorf("errors: %w, %w", err1, err2)
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

func Foobar() error {
	_, err := doSmth()
	if err != nil {
		_, errB := doSmth()

		return errB
	}

	return nil
}

func Foobar3() error {
	_, err := doSmth()
	if err != nil {
		_, errB := doSmth()
		if errB != nil {
			_, errC := doSmth()

			return errC
		}
	}

	return nil
}

// ----------------------------------------------------
// Helpers

func closureWrapper(fn func() error) error {
	return fn()
}

func fooWrap(_ int, err error, _ string) error {
	return err
}

func doSmth() (int, error) {
	return 0, errors.New("doSmth failed")
}
