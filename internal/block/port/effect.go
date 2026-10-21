package port

import "errors"

// ErrNoEffect marks a Tool error raised before the Tool changed anything:
// the edit text was not found, the path was refused, the command was
// blocked. The Runtime may hand such an error back to the model to try
// again, even for a Tool that changes files or runs commands, because
// nothing needs undoing.
var ErrNoEffect = errors.New("tool made no change")

// NoEffect wraps err as an ErrNoEffect failure; its message is unchanged.
func NoEffect(err error) error {
	if err == nil {
		return nil
	}
	return noEffectError{err}
}

type noEffectError struct{ err error }

func (e noEffectError) Error() string        { return e.err.Error() }
func (e noEffectError) Unwrap() error        { return e.err }
func (e noEffectError) Is(target error) bool { return target == ErrNoEffect }
