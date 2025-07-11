package pkg

import (
	"errors"
	"fmt"
)

var ExternalError = errors.New("external error")

var ExternalErrors = struct {
	ExtError error
}{
	ExtError: errors.New("ExtError"),
}

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

func ReturningMessage() (error, string) {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if err != nil {
		return anotherErr, err.Error() // want "returning not the error that was checked"
	}

	return nil, "foo"
}

func ClosureErrors() error {
	err := closureWrapper(func() error {
		_, err := doSmth()
		if err != nil {
			return fooWrap(1, err, "a")
		}

		if _, innerErr := doSmth(); innerErr != nil {
			return fooWrap(1, err, "a") // want "returning not the error that was checked"
		}

		return nil
	})

	return err
}

func ClosureReturnsErrorAssignedOutside() error {
	funcErr := errors.New("func error")
	anotherFuncErr := errors.New("another func error")

	err := closureWrapper(func() error {
		if funcErr != nil {
			return anotherFuncErr // want "returning not the error that was checked"
		}

		return nil
	})

	return err
}

func ClosureReturnsErrorAssignedInside() error {
	err := closureWrapper(func() error {
		innerErr := errors.New("inner")
		anotherInnerErr := errors.New("another inner")

		if innerErr != nil {
			return anotherInnerErr // want "returning not the error that was checked"
		}

		return nil
	})

	return err
}

func ClosureReturnsErrorDeclaredOutside() error {
	var (
		innerErr        = errors.New("inner")
		anotherInnerErr = errors.New("another inner")
	)

	err := closureWrapper(func() error {
		if innerErr != nil {
			return anotherInnerErr // want "returning not the error that was checked"
		}

		return nil
	})

	return err
}

func ClosureReturnsErrorDeclaredInside() error {
	err := closureWrapper(func() error {
		var (
			innerErr        = errors.New("inner")
			anotherInnerErr = errors.New("another inner")
		)

		if innerErr != nil {
			return anotherInnerErr // want "returning not the error that was checked"
		}

		return nil
	})

	return err
}

func ClosureInDeclaration() error {
	var err = closureWrapper(func() error {
		innerErr := errors.New("inner")
		anotherInnerErr := errors.New("another")

		if innerErr != nil {
			return anotherInnerErr // want "returning not the error that was checked"
		}

		return nil
	})

	return err
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

// ----------------------------------------------------
// Suppressed triggers

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

type someStruct struct {
	val int
	err error
}

var structs = []someStruct{}

func ReturnFieldOfStruct() (int, error) {
	for _, s := range structs {
		if s.val > 1 {
			return s.val, s.err
		}
	}

	return 0, nil
}

func ReturnFieldOfStruct2() error {
	cond1 := true
	cond2 := true

	if cond1 && cond2 {
		return fmt.Errorf("%w, upload_id = %s", ExternalErrors.ExtError, 12)
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
		return anotherErr
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
		return anotherErr
	}

	return nil
}

func FooCheckWrappedWrong() error {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if fooCheck(1, err, "a") {
		return fmt.Errorf("error: %w", anotherErr)
	}

	return nil
}

func FooCheckReturnMessageWrong() error {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if fooCheck(1, err, "a") {
		return fmt.Errorf("error: %s", anotherErr.Error())
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

func ReturnWrappedErrorMessage2() error {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if err != nil {
		return errors.New(fooWrap(1, anotherErr, "a").Error())
	}

	return nil
}

func ErrorCheckedInOuterIfStatement() error {
	err := errors.New("some error")
	anotherErr := errors.New("another error")

	if err != nil {
		if anotherErr != nil {
			return err
		}
	}

	return nil
}

func WrappingWithAssignmentBeforeReturning() error {
	err := errors.New("error")
	if err != nil {
		wrappedErr := fmt.Errorf("wrapped: %w", err)

		return wrappedErr
	}

	return nil
}

func WrappingWithDeclarationBeforeReturning() error {
	err := errors.New("error")
	if err != nil {
		var wrappedErr error = fmt.Errorf("wrapped: %w", err)

		return wrappedErr
	}

	return nil
}

func AssignCheckedErr() error {
	err := errors.New("foo")
	if err != nil {
		tmpErr := err
		err := errors.New("bar")
		if err != nil {
			return fmt.Errorf("errors: %s, %s", tmpErr, err)
		}
	}

	return nil
}

func AssignCheckedErrViaDeclaration() error {
	err := errors.New("foo")
	if err != nil {
		var tmpErr error = err
		err := errors.New("bar")
		if err != nil {
			return fmt.Errorf("errors: %s, %s", tmpErr, err)
		}
	}

	return nil
}

func PkgError() error {
	err := errors.New("error")
	if err != nil {
		return ExternalError
	}

	return nil
}

func WrapCycle() error {
	err := errors.New("error")
	if err != nil {
		var wrappedB error
		wrappedA := fmt.Errorf("wrapped: %w", wrappedB)
		wrappedB = fmt.Errorf("wrapped: %w", wrappedA)
		wrappedB = fmt.Errorf("wrapped: %w", wrappedA)
		wrappedA = fmt.Errorf("wrapped: %w", wrappedB)

		return wrappedB
	}

	return nil
}

func NestedErrorCheckReturnsOuterError() error {
	err := errors.New("some error")

	if err != nil {
		if innerErr := errors.New("inner"); innerErr != nil {
			return err
		}
	}

	return nil
}

func NestedErrorCheckReturnsOuterErrorMessage() error {
	err := errors.New("some error")

	if err != nil {
		if innerErr := errors.New("inner"); innerErr != nil {
			return fmt.Errorf("%w: %s", err, innerErr.Error())
		}
	}

	return nil
}

func NestedErrorCheckReturnsOuterErrorMessage2() error {
	err := errors.New("some error")

	if err != nil {
		if innerErr := errors.New("inner"); innerErr != nil {
			return fmt.Errorf("wrapped: %s", err.Error())
		}
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

func More() error {
	_, err := doSmth()
	if err != nil {
		if _, errB := doSmth(); errB != nil {
			return fmt.Errorf("wrapped: %w", err)
		}

		return nil
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

func fooCheck(_ int, err error, _ string) bool {
	return err != nil
}

func doSmth() (int, error) {
	return 0, errors.New("doSmth failed")
}
