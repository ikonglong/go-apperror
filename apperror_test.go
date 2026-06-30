package apperror

import (
	"errors"
	"strings"
	"testing"
)

// errorCase mirrors the Python ErrorCase used in tests: a Case identified by
// a string, with an associated op-status code that is not asserted on.
type errorCase struct {
	id     string
	opCode Code
}

func (e errorCase) Identifier() string { return e.id }

// connection raises a low-level RuntimeError-like error.
type connection struct{}

func (connection) ExecSQL(sql string) error {
	return errors.New("Network error")
}

type dbClient struct{ conn connection }

func (d dbClient) Insert(data string) error {
	if err := d.conn.ExecSQL("insert into ..."); err != nil {
		return NewInternal("db.insert", WithMessage("DB insert failed"), WithCause(err))
	}
	return nil
}

type service struct{ db dbClient }

func (s service) Save(data string) error { return s.db.Insert(data) }

type api struct{ s service }

func (a api) Create(data string) error { return a.s.Save(data) }

func TestCheckAppErrorCause(t *testing.T) {
	a := api{s: service{db: dbClient{conn: connection{}}}}
	err := a.Create("test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *AppError, got %T", err)
	}
	cause := appErr.Cause()
	if cause == nil {
		t.Fatal("expected cause to be set")
	}
	if cause.Error() != "Network error" {
		t.Errorf("cause.Error() = %q, want %q", cause.Error(), "Network error")
	}

	// Unwrap path also works for errors.Is.
	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) should be true")
	}
}

func TestCreateAppErrorWithCause(t *testing.T) {
	low := errors.New("low-level error in func_a")
	appErr := NewInternal("test.cause", WithMessage("higher-level error msg"), WithCause(low))
	appErr.AddNote("additional err context...")
	if !errors.Is(appErr, low) {
		t.Errorf("errors.Is(appErr, low) = false, want true")
	}
}

func TestAddNote(t *testing.T) {
	e := NewInternal("test.ctx", WithMessage("initial error message"))
	e.AddNote("first context")
	e.AddNote("second context")
	want := "second context -> first context -> initial error message"
	if e.Message() != want {
		t.Errorf("Message() = %q, want %q", e.Message(), want)
	}
	if !strings.Contains(e.Error(), want) {
		t.Errorf("Error() should contain %q, got %q", want, e.Error())
	}
}

func TestAppErrorString(t *testing.T) {
	cases := []struct {
		name string
		err  *AppError
		want string
	}{
		{
			name: "event only, message falls back to code description",
			err:  NewInternal("user.signup"),
			want: "AppError(code=INTERNAL(13), event=user.signup, case=None, message='internal', details=None)",
		},
		{
			name: "with explicit message",
			err:  NewInternal("user.signup", WithMessage("boom")),
			want: "AppError(code=INTERNAL(13), event=user.signup, case=None, message='boom', details=None)",
		},
		{
			name: "with case",
			err:  NewInternal("user.signup", WithMessage("boom"), WithCase(errorCase{id: "1001", opCode: CodeInternal})),
			want: "AppError(code=INTERNAL(13), event=user.signup, case=1001, message='boom', details=None)",
		},
		{
			name: "with details",
			err:  NewInternal("user.signup", WithMessage("boom"), WithDetails(map[string]string{"key": "value"})),
			want: "AppError(code=INTERNAL(13), event=user.signup, case=None, message='boom', details=map[key:value])",
		},
		{
			name: "full",
			err: NewInternal("user.signup",
				WithMessage("boom"),
				WithCase(errorCase{id: "1001", opCode: CodeInternal}),
				WithDetails(map[string]string{"key": "value"})),
			want: "AppError(code=INTERNAL(13), event=user.signup, case=1001, message='boom', details=map[key:value])",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.err.String(); got != c.want {
				t.Errorf("String() = %q\nwant      %q", got, c.want)
			}
		})
	}
}

func TestBuildErrorWithOptions(t *testing.T) {
	e := NewInternal("test.options", WithMessage("internal error"))
	if e.Code() != CodeInternal || e.Message() != "internal error" || e.Details() != nil {
		t.Errorf("unexpected basic error: %v", e)
	}

	d := map[string]any{"key1": "value1"}
	e = NewInternal("test.options", WithMessage("internal error"), WithDetails(d))
	got, ok := e.Details().(map[string]any)
	if !ok || got["key1"] != "value1" {
		t.Errorf("unexpected details: %v", e.Details())
	}

	d2 := map[string]any{"key1": "value1", "extra_info": map[string]string{"key2": "value2"}}
	e = NewInternal("test.options", WithMessage("internal error"), WithDetails(d2))
	got, ok = e.Details().(map[string]any)
	if !ok || got["key1"] != "value1" {
		t.Errorf("unexpected nested details: %v", e.Details())
	}
}

func TestNewIllegalArg(t *testing.T) {
	e := NewIllegalArg("test.illegalarg", WithMessage("illegal arg"))
	if e.Code() != CodeIllegalArg || e.Message() != "illegal arg" {
		t.Errorf("unexpected illegal-arg error: %v", e)
	}

	e = NewIllegalArg("test.illegalarg", WithMessage("Test error"))
	want := "AppError(code=ILLEGAL_ARG(29), event=test.illegalarg, case=None, message='Test error', details=None)"
	if got := e.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestAddMoreErrCtx(t *testing.T) {
	validate := func() error { return NewIllegalArg("test.morectx", WithMessage("illegal arg")) }

	err := validate()
	var appErr *AppError
	if errors.As(err, &appErr) {
		appErr.AddNote("Error while executing ...")
	}
	want := "AppError(code=ILLEGAL_ARG(29), event=test.morectx, case=None, message='Error while executing ... -> illegal arg', details=None)"
	if got := appErr.String(); got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

// TestFactoryFallsBackToCodeDescriptionForOmittedMessage exercises the
// optional-message contract: when WithMessage is not used, Message()
// returns Code.Description() so unstructured loggers still see a sensible
// string instead of "".
func TestFactoryFallsBackToCodeDescriptionForOmittedMessage(t *testing.T) {
	e := NewInternal("test.fallback")
	if e.Message() != CodeInternal.Description() {
		t.Errorf("omitted WithMessage should fall back to code description, got %q", e.Message())
	}
}

func TestAppErrorEvent_RequiredFieldSet(t *testing.T) {
	e := NewInternal("user.signup")
	if e.Event() != "user.signup" {
		t.Errorf("Event() = %q, want %q", e.Event(), "user.signup")
	}
}

// TestAppErrorEvent_PanicsOnEmpty asserts the construction-time contract:
// empty (or whitespace-only) event panics rather than silently producing
// an unclassified AppError. See newAppError.
func TestAppErrorEvent_PanicsOnEmpty(t *testing.T) {
	cases := []string{"", "   ", "\t\n"}
	for _, ev := range cases {
		t.Run("event="+strings.ReplaceAll(ev, "\n", "\\n"), func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("expected panic for event=%q, got none", ev)
				}
			}()
			_ = NewInternal(ev)
		})
	}
}

func TestWithMessageSetsMessage(t *testing.T) {
	e := NewInternal("test.event", WithMessage("custom message"))
	if e.Message() != "custom message" {
		t.Errorf("Message() = %q, want %q", e.Message(), "custom message")
	}
}
