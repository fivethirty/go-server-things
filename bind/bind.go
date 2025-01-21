package bind

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/go-playground/form"
	"github.com/go-playground/validator/v10"
)

var v = validator.New()

var ErrDecoding error = errors.New("can't decode")
var ErrValidating error = errors.New("can't validate")

func JSON(body io.ReadCloser, target any) error {
	decoder := json.NewDecoder(body)

	err := decoder.Decode(target)
	if err != nil {
		return fmt.Errorf("JSON: %w %w", ErrDecoding, err)
	}

	err = decoder.Decode(&map[string]any{})
	if !errors.Is(err, io.EOF) {
		return fmt.Errorf("JSON: %w %w", ErrDecoding, err)
	}

	if err := validate(target); err != nil {
		return fmt.Errorf("JSON: %w", err)
	}
	return nil
}

var formDecoder *form.Decoder = form.NewDecoder()

func Query(values url.Values, target any) error {
	if err := query(values, target); err != nil {
		return fmt.Errorf("Query: %w", err)
	}
	return nil
}

func PostForm(r *http.Request, target any) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("PostForm: %w %w", ErrDecoding, err)
	}
	if err := query(r.PostForm, target); err != nil {
		return fmt.Errorf("PostForm: %w", err)
	}
	return nil
}

func query(values url.Values, target any) error {
	if err := formDecoder.Decode(target, values); err != nil {
		return fmt.Errorf("%w %w", ErrDecoding, err)
	}

	if err := validate(target); err != nil {
		return fmt.Errorf("Query: %w", err)
	}
	return nil
}

func Validate(target any) error {
	if err := validate(target); err != nil {
		return fmt.Errorf("Validate: %w", err)
	}
	return nil
}

func validate(target any) error {
	if err := v.Struct(target); err != nil {
		return fmt.Errorf("%w %w", ErrValidating, err)
	}
	return nil
}
