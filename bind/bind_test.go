package bind_test

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/fivethirty/go-server-things/bind"
)

type testTarget struct {
	Foo string `json:"foo" form:"foo" validate:"required"`
	Bar int    `json:"bar" form:"bar" validate:"required"`
}

const (
	foo = "hello"
	bar = 1
)

func testBind(t *testing.T, target testTarget, err error, expctedErr error) {
	t.Helper()
	if !errors.Is(err, expctedErr) {
		t.Errorf("expected error %v, got %v", expctedErr, err)
	}

	if err != nil {
		return
	}

	if target.Foo != foo {
		t.Errorf("expected %s, got %s", foo, target.Foo)
	}
	if target.Bar != bar {
		t.Errorf("expected %d, got %d", bar, target.Bar)
	}
}

func TestJSON(t *testing.T) {
	t.Parallel()

	test := []struct {
		name string
		json string
		err  error
	}{
		{
			name: "should parse json",
			json: `{"foo":"hello","bar":1}`,
		},
		{
			name: "should return decoding error with invalid json",
			json: `{"foo":"hel`,
			err:  bind.ErrDecoding,
		},
		{
			name: "should return decoding error with 2 json objects",
			json: `{"foo":"hello","bar":1}{"foo":"hello","bar":1}`,
			err:  bind.ErrDecoding,
		},
		{
			name: "should return decoding error with random text after json",
			json: `{"foo":"hello","bar":1}foo`,
			err:  bind.ErrDecoding,
		},
		{
			name: "should return validation error",
			json: `{"foo":"hello"}`,
			err:  bind.ErrValidating,
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			closer := io.NopCloser(strings.NewReader(tt.json))
			target := testTarget{}
			err := bind.JSON(closer, &target)
			testBind(t, target, err, tt.err)
		})
	}
}

func TestQuery(t *testing.T) {
	t.Parallel()

	test := []struct {
		name   string
		values url.Values
		err    error
	}{
		{
			name:   "should parse query",
			values: url.Values{"foo": []string{"hello"}, "bar": []string{"1"}},
		},
		{
			name:   "should return decoding error",
			values: url.Values{"foo": []string{"hello"}, "bar": []string{"world"}},
			err:    bind.ErrDecoding,
		},
		{
			name:   "should return validation error",
			values: url.Values{"foo": []string{"hello"}},
			err:    bind.ErrValidating,
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			target := testTarget{}
			err := bind.Query(tt.values, &target)
			testBind(t, target, err, tt.err)
		})
	}
}

func TestPostForm(t *testing.T) {
	t.Parallel()

	test := []struct {
		name string
		body string
		err  error
	}{
		{
			name: "should parse form",
			body: "foo=hello&bar=1",
		},
		{
			name: "should return decoding error with invalid types",
			body: "foo=hello&bar=world",
			err:  bind.ErrDecoding,
		},
		{
			name: "should return decoding error with invalid structure",
			body: "fo;o=hello",
			err:  bind.ErrDecoding,
		},
		{
			name: "should return validation error",
			body: "foo=hello",
			err:  bind.ErrValidating,
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := &http.Request{
				Method: http.MethodPost,
				Header: http.Header{
					"Content-Type": []string{"application/x-www-form-urlencoded"},
				},
				Body: io.NopCloser(strings.NewReader(tt.body)),
			}
			target := testTarget{}
			err := bind.PostForm(req, &target)
			testBind(t, target, err, tt.err)
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	test := []struct {
		name   string
		target testTarget
		err    error
	}{
		{
			name: "should return no error",
			target: testTarget{
				Foo: foo,
				Bar: bar,
			},
		},
		{
			name: "should return validation error",
			target: testTarget{
				Foo: foo,
			},
			err: bind.ErrValidating,
		},
	}

	for _, tt := range test {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := bind.Validate(&tt.target)
			if !errors.Is(err, tt.err) {
				t.Errorf("expected error %v, got %v", tt.err, err)
			}
		})
	}
}
