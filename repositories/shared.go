package repositories

type NotFoundError struct {
	Err error
}

func (e NotFoundError) Error() string {
	return "not found"
}

func (e NotFoundError) Unwrap() error {
	return e.Err
}

type PermissionDeniedOrNotFoundError struct {
	Err error
}

func (e PermissionDeniedOrNotFoundError) Error() string {
	return "Invalid space. Ensure that the space exists and you have access to it."
}

func (e PermissionDeniedOrNotFoundError) Unwrap() error {
	return e.Err
}

type ResourceNotFoundError struct {
	Err error
}

func (e ResourceNotFoundError) Error() string {
	return "Resource not found."
}

func (e ResourceNotFoundError) Unwrap() error {
	return e.Err
}