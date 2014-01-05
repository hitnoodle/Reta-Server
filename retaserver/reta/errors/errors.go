package errors

type errorString struct {
	s string
}

func (e *errorString) Error() string {
	return e.s
}

func New(test string) error {
	return &errorString{test}
}
