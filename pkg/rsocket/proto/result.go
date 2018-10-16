package proto

// Result of Payload or error
type Result struct {
	Payload *Payload

	Err error
}

// Ok returns a Result with Payload
func Ok(payload *Payload) *Result {
	return &Result{payload, nil}
}

// Err returns a Result with error
func Err(err error) *Result {
	return &Result{nil, err}
}
