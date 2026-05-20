# tiling

Awesome-wm style tiling layout manager for bubbletea TUIs.

## Testing

### Run all tests

```sh
go test ./...
```

### Run a specific test

```sh
go test -run TestGoldenRender/vertical_2_children ./...
```

### Verbose output

```sh
go test -v ./...
```

## Golden/Snapshot Tests

Tests use [charmbracelet/x/exp/golden](https://github.com/charmbracelet/x/tree/main/exp/golden) to compare rendered output against stored snapshots in `testdata/`. Each `.golden` file contains the expected output for a test, including ANSI escape sequences for styled terminal rendering.

### Updating golden files

When you intentionally change the rendered output, update the snapshots with:

```sh
go test -update ./...
```

This overwrites all `.golden` files under `testdata/` with the current output. Review the diffs with `git diff testdata/` before committing.

### Golden file layout

```
testdata/
├── TestSingleChild.golden
├── TestTwoChildrenVertical.golden
├── TestGoldenRender/
│   ├── single_child.golden
│   ├── vertical_2_children.golden
│   └── ...
└── TestGoldenModelView/
    ├── single_block.golden
    └── ...
```

Top-level test names produce flat files (`TestName.golden`). Subtests produce nested directories (`TestParent/subtest.golden`).

### Adding a new golden test

Add a subtest to `TestGoldenRender` or `TestGoldenModelView`:

```go
{
    name:   "my_new_case",
    layout: New().Size(80, 24).Children("A", "B"),
},
```

Then generate its golden file:

```sh
go test -update -run TestGoldenRender/my_new_case ./...
```
