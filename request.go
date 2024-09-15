package peanut

// Request defines the interface for a request.
type Request interface {
	// GetHeader returns the header with the given key.
	GetHeader(key string) string
	// GetHeaders returns all headers.
	GetHeaders() map[string]string
	// UnmarshalBody unmarshals the body into the given struct.
	UnmarshalBody(v interface{}) error
}
