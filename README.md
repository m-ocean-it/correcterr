# correcterr

## About

It's a linter that checks whether the returned error is the same one that was checked in the condition.

Due to the fuzzy nature of error handling in Golang, the linter has to employ certain heuristics to reduce the amount of false-positives. If you encounter obvious false-positives (or false-negatives), feel free to [open an issue](https://github.com/m-ocean-it/correcterr/issues/new).

The linter, as it turned out, is quite similar to [`nilnesserr`](https://github.com/alingse/nilnesserr) and [seems](https://github.com/m-ocean-it/correcterr/issues/2#issuecomment-2972835988) to find a lot of the same problems, however, it [detects](https://github.com/m-ocean-it/correcterr/issues/2#issuecomment-2972844048) some additional mistakes which `nilnesserr` fails to identify (as of June 14, 2025).

### Examples

For more examples, see the `err_mistakes.go` file which is used in automated testing.

#### Will trigger

```go
if txErr != nil {
    return err // will be reported
}
```

```go
if txErr := doSmth(); txErr != nil {
    return err // will be reported
}
```

```go
if err != nil {
    return fmt.Errorf("error: %w", anotherErr) // will be reported
}
```

```go
if err != nil {
    return someCustomWrapper(someCustomError(anotherErr)) // will be reported
}
```

#### Will NOT trigger

```go
if err != nil {
    return errors.New("another") // creating an error on the spot is fine
}
```

```go
import "custom_errors"

if err != nil {
    return custom_errors.SomeError // returning an error defined outside of the function's scope is fine
}
```


## Installation
```sh
go install github.com/m-ocean-it/correcterr/cmd/correcterr@latest
```

The same command will update the package on your machine.

## Usage
```sh
correcterr [-flag] [package]
```

To run on the whole project:

```sh
correcterr ./...
```

### The `nolint`-directive is supported

All examples below are sufficient to disable a diagnostic on a specific line:

```go
//nolint
//nolint:correcterr
//nolint:foo,correcterr,bar
//nolint:all
```

*Make sure not to add a space in front of `nolint`.*

## [`golangci-lint`](https://github.com/golangci/golangci-lint) integration

`correcterr` is [not likely](https://github.com/golangci/golangci-lint/pull/5875) to become a part of linters included in `golangci-lint`, however, I will, probably, implement a [plugin](https://golangci-lint.run/plugins/module-plugins/) to allow easy integration with `golangci-lint` "by hand".
