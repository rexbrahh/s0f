package core

import "errors"

var (
    // ErrInvalidParent indicates a parent that does not exist or is not a folder.
    ErrInvalidParent = errors.New("invalid parent")
    // ErrCycleDetected indicates a move that would introduce a cycle.
    ErrCycleDetected = errors.New("cycle detected")
)

// ValidateOps performs basic syntactic validation of a batch before hitting storage.
func ValidateOps(tree Tree, ops []Op) error {
    // TODO: Implement full validation once storage wiring is available.
    _ = tree
    _ = ops
    return nil
}
