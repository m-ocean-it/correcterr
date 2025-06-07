# correcterr

## About

It's a linter that checks whether the returned error is the same one that was checked in the condition.

### Examples

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
if errors.Is(err1, err2) {
    return anotherErr // will be reported
}
```

*Note: returning the message of the checked error with the `Error`-method is considered "returning" a checked error. For example, this will not be reported:*

```go
if err != nil {
    return fmt.Errorf("error message: %s", err.Error())
}
```

## Installation
```sh
go install github.com/m-ocean-it/correcterr@latest
```

The same command will update the package on your machine.

## Usage
```sh
correcterr ./...  # or specify a package
```

### The `nolint`-directive is supported

All examples below are sufficient to disable a diagnostic on a specific line:

```go
//nolint
//nolint:correcterr
//nolint:foo,correcterr,bar
//nolint:all
```

*Make sure to not add a space in front of `nolint`.*

## Roadmap

- [ ] Maybe, ignore when `errors.New` is used after checking some error 
- [ ] Pull request to `golangci-lint`
