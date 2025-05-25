# correcterr

## About

It's a linter that checks whether the returned error is the same one that was checked in the condition.

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


## Roadmap

### Account for `errors.Is` checks
```go
if errors.Is(err1, err2) {
    return err2 // want "returning not the error that was checked"
}
```

### Check for `nil`-error returns
```go
if err == nil {
    return err // want "returning err when nil"
}
```
