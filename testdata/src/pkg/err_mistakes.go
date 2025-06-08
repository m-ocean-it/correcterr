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

func CheckingAndReturningDifferentErrorsNoLint() error {
	var err1 = errors.New("1")
	var err2 = errors.New("2")

	if err1 != nil {
		return err2 //nolint
	}

	return nil
}

func CheckingAndReturningDifferentErrorsNoLintAll() error {
	var err1 = errors.New("1")
	var err2 = errors.New("2")

	if err1 != nil {
		return err2 //nolint:all
	}

	return nil
}

func CheckingAndReturningDifferentErrorsNoLintAllWithSomethingElse() error {
	var err1 = errors.New("1")
	var err2 = errors.New("2")

	if err1 != nil {
		return err2 //nolint: foo,all ,bar
	}

	return nil
}

func CheckingAndReturningDifferentErrorsNoLintCorrecterr() error {
	var err1 = errors.New("1")
	var err2 = errors.New("2")

	if err1 != nil {
		return err2 //nolint:correcterr,foo
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

func ReturningMessage() (error, string) {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if err != nil {
		return anotherErr, err.Error()
	}

	return nil, "foo"
}

func ReturningWrappedMessage() error {
	err := errors.New("some error")
	if err != nil {
		return fooWrap(1, errors.New("new error"), err.Error())
	}

	return nil
}

func ReturningWrappedMessage2() error {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if err != nil {
		return fooWrap(1, errors.New("new error"), anotherErr.Error())
	}

	return nil
}

func ErrorsIsCorrect() error {
	err := errors.New("original")
	wrappedErr := fmt.Errorf("wrapped: %w", err)
	if errors.Is(wrappedErr, err) {
		return wrappedErr
	}

	return nil
}

func ErrorsIsWrong() error {
	err := errors.New("original")
	anotherErr := errors.New("another")

	wrappedErr := fmt.Errorf("wrapped: %w", err)
	if errors.Is(wrappedErr, err) {
		return anotherErr // want "returning not the error that was checked"
	}

	return nil
}

func FooCheckCorrect() error {
	err := errors.New("some error")

	if fooCheck(1, err, "a") {
		return err
	}

	return nil
}

func FooCheckWrappedCorrect() error {
	err := errors.New("some error")

	if fooCheck(1, err, "a") {
		return fmt.Errorf("error: %w", err)
	}

	return nil
}

func FooCheckReturnMessageCorrect() error {
	err := errors.New("some error")

	if fooCheck(1, err, "a") {
		return errors.New(err.Error())
	}

	return nil
}

func FooCheckReturnMessageCorrectWrapped() error {
	err := errors.New("some error")

	if fooCheck(1, err, "a") {
		return fmt.Errorf("error: %s", err.Error())
	}

	return nil
}

func FooCheckWrong() error {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if fooCheck(1, err, "a") {
		return anotherErr // want "returning not the error that was checked"
	}

	return nil
}

func FooCheckWrappedWrong() error {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if fooCheck(1, err, "a") {
		return fmt.Errorf("error: %w", anotherErr) // want "returning not the error that was checked"
	}

	return nil
}

func FooCheckReturnMessageWrong() error {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if fooCheck(1, err, "a") {
		return fmt.Errorf("error: %s", anotherErr.Error()) // want "returning not the error that was checked"
	}

	return nil
}

func FooCheckCorrect2() (error, error) {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if fooCheck(1, err, "a") {
		return err, anotherErr
	}

	return nil, nil
}

func CheckTwoErrorsCorrect() error {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if err != nil && anotherErr != nil {
		return err
	}

	return nil
}

func ReturnWrappedErrorMessage() error {
	err := errors.New("some error")

	if err != nil {
		return errors.New(fooWrap(1, err, "a").Error())
	}

	return nil
}

func fooWrap(_ int, err error, _ string) error {
	return err
}

func fooCheck(_ int, err error, _ string) bool {
	return err != nil
}

// TODO
// func CheckTwoErrorsWrong() error {
// 	err := errors.New("some error")
// 	anotherErr := errors.New("another error")
// 	thirdErr := errors.New("third error")

// 	if err != nil && anotherErr != nil {
// 		return thirdErr // should error
// 	}

// 	return nil
// }
