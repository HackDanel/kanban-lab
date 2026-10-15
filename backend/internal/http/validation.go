package httpserver

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/mail"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/google/uuid"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

const apiBase = "/api/v1"

func init() {
	// Same rule as the generated api.Email type uses when decoding, so that a
	// bad address is reported by the validator together with its field name.
	openapi3.DefineStringFormatValidator("email", openapi3.NewCallbackValidator(func(value string) error {
		if _, err := mail.ParseAddress(value); err != nil {
			return errors.New("укажите корректный email")
		}
		return nil
	}))
	openapi3.DefineStringFormatValidator("uuid", openapi3.NewCallbackValidator(func(value string) error {
		if _, err := uuid.Parse(value); err != nil {
			return errors.New("некорректный идентификатор")
		}
		return nil
	}))
}

// paramName extracts the parameter name from the binding errors of the generated router.
func paramName(err error) (string, bool) {
	var (
		format   *api.InvalidParamFormatError
		unmarsh  *api.UnmarshalingParamError
		required *api.RequiredParamError
		header   *api.RequiredHeaderError
		tooMany  *api.TooManyValuesForParamError
	)
	switch {
	case errors.As(err, &format):
		return format.ParamName, true
	case errors.As(err, &unmarsh):
		return unmarsh.ParamName, true
	case errors.As(err, &required):
		return required.ParamName, true
	case errors.As(err, &header):
		return header.ParamName, true
	case errors.As(err, &tooMany):
		return tooMany.ParamName, true
	}
	return "", false
}

// requestValidator checks every API request against the OpenAPI contract:
// parameters, bodies and the security requirement of the operation.
func requestValidator(log *slog.Logger) (func(http.Handler) http.Handler, error) {
	spec, err := api.GetSpec()
	if err != nil {
		return nil, err
	}
	// The validator matches full request paths, so the server base path is
	// folded into the paths instead of being matched against the Host header.
	spec.Servers = nil
	paths := openapi3.NewPaths()
	for path, item := range spec.Paths.Map() {
		paths.Set(apiBase+path, item)
	}
	spec.Paths = paths

	return nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		Options: openapi3filter.Options{
			MultiError: true,
			AuthenticationFunc: func(ctx context.Context, in *openapi3filter.AuthenticationInput) error {
				if actorFrom(in.RequestValidationInput.Request.Context()).ID == uuid.Nil {
					return errors.New("missing or invalid access token")
				}
				return nil
			},
		},
		// Uploads are streamed by the handler; the service enforces authentication.
		Skipper: func(r *http.Request) bool {
			return strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/")
		},
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, r *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
			writeError(w, r, log, validationError(err, opts.StatusCode))
		},
		MultiErrorHandler: func(me openapi3.MultiError) (int, error) {
			return http.StatusBadRequest, me
		},
	}), nil
}

var errPayloadTooLarge = domain.E(domain.KindPayloadTooLarge, "Слишком большой запрос")

// isTooLarge reports whether reading the body hit the size limit. kin-openapi
// does not always wrap the reader error, so the message is checked as well.
func isTooLarge(err error) bool {
	var maxBytes *http.MaxBytesError
	return errors.As(err, &maxBytes) || strings.Contains(err.Error(), "http: request body too large")
}

// validationError converts kin-openapi errors into the API error format.
func validationError(err error, status int) error {
	if isTooLarge(err) {
		return errPayloadTooLarge
	}
	var fields []domain.FieldError
	unauthorized := false
	var walk func(error, string)
	walk = func(err error, field string) {
		var multi openapi3.MultiError
		var security *openapi3filter.SecurityRequirementsError
		var request *openapi3filter.RequestError
		var schema *openapi3.SchemaError
		switch {
		case errors.As(err, &multi):
			for _, e := range multi {
				walk(e, field)
			}
		case errors.As(err, &security):
			unauthorized = true
		case errors.As(err, &request):
			if request.Parameter != nil {
				field = request.Parameter.Name
			}
			if request.Err == nil {
				fields = append(fields, domain.FieldError{Field: field, Message: request.Reason})
				return
			}
			walk(request.Err, field)
		case errors.As(err, &schema):
			if pointer := schema.JSONPointer(); len(pointer) > 0 {
				field = strings.Join(pointer, ".")
			}
			fields = append(fields, domain.FieldError{Field: field, Message: schema.Reason})
		default:
			if name, ok := paramName(err); ok {
				fields = append(fields, domain.FieldError{Field: name, Message: "Некорректное значение параметра"})
				return
			}
			fields = append(fields, domain.FieldError{Field: field, Message: err.Error()})
		}
	}
	walk(err, "")

	switch {
	case unauthorized:
		return domain.ErrUnauthorized
	case status == http.StatusNotFound:
		return domain.E(domain.KindNotFound, "Метод API не найден")
	case status == http.StatusMethodNotAllowed:
		return domain.E(domain.KindNotFound, "Метод не поддерживается для этого адреса")
	}
	return &domain.Error{Kind: domain.KindValidation, Message: "Некорректные данные", Fields: fields}
}
