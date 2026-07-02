package apperror

import "fmt"

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
// presence distinguishes a remote-response failure (RemoteError with
// errResp) from a transport failure (RemoteError with cause — no
// response received). Both paths produce a RemoteError, not an AppError.
// See the RemoteError and RemoteErrorResp docs.
type Response struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte // raw bytes; parsed views live on RemoteErrorResp fields
}

// String returns a debug representation. Body is output as a string.
func (r *Response) String() string {
	return fmt.Sprintf("Response(status=%d, body=%s)", r.StatusCode, string(r.Body))
}
