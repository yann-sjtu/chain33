package store

type validator struct {
}

// Validate validates the given record, returning an error if it's invalid.
func (validator) Validate(_ string, _ []byte) error {
	return nil
}

// Select selects the best record from the set of records.
func (validator) Select(_ string, _ [][]byte) (int, error) {
	return 0, nil
}
