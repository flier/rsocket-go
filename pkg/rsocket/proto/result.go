package proto

type Result struct {
	Payload *Payload

	Err error
}

func Ok(payload *Payload) *Result {
	return &Result{payload, nil}
}

func Err(err error) *Result {
	return &Result{nil, err}
}
