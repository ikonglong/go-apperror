package apperror

// Request captures an outbound HTTP/RPC request associated with an error.
// Capture is optional; clients often omit it (or omit just the Body) to
// avoid leaking sensitive request payloads through logs.
type Request struct {
	URL     string
	Method  string
	Headers map[string][]string
	Body    []byte // raw bytes
}

// Response captures an HTTP/RPC response associated with an error. Its
// presence is what distinguishes a remote-response failure (modeled as
// RemoteErrorResp + RemoteError) from a transport failure (modeled as a
// plain AppError). See the RemoteError and RemoteErrorResp docs.
type Response struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte // raw bytes; parsed views live on RemoteErrorResp fields
}
