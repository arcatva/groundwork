// Package spec holds the skill's machine-readable contracts and its index, as Go types. The same struct
// tags drive both the validation `groundwork check` runs and the JSON Schemas `groundwork gen` writes
// into the skill, so the two cannot drift apart.
//
// A field's `check` tag lists its rules: `required` (present and not empty), `enum=a|b` (one of these),
// `min=N` (at least N items).
package spec

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type rules struct {
	required bool
	enum     []string
	min      int
}

func parse(tag string) rules {
	var r rules
	for _, part := range strings.Split(tag, ",") {
		key, value, _ := strings.Cut(strings.TrimSpace(part), "=")
		switch key {
		case "required":
			r.required = true
		case "enum":
			r.enum = strings.Split(value, "|")
		case "min":
			r.min, _ = strconv.Atoi(value)
		}
	}
	return r
}

func jsonName(f reflect.StructField) string {
	name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
	if name == "" {
		return f.Name
	}
	return name
}

// Checker is a type with rules across its fields, beyond what its tags say.
type Checker interface {
	Check() []error
}

// Validate checks v against its fields' tags, then its own Check, recursively, and returns every
// problem with the path to it.
func Validate(v any) error {
	var problems []error
	validate(reflect.ValueOf(v), "", &problems)
	return errors.Join(problems...)
}

func validate(v reflect.Value, path string, problems *[]error) {
	for v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			fv := v.Field(i)
			at := join(path, jsonName(f))
			r := parse(f.Tag.Get("check"))
			if r.required && fv.IsZero() {
				*problems = append(*problems, fmt.Errorf("%s: required", at))
				continue
			}
			if len(r.enum) > 0 && fv.Kind() == reflect.String && fv.String() != "" && !contains(r.enum, fv.String()) {
				*problems = append(*problems, fmt.Errorf("%s: %q is not one of %s", at, fv.String(), strings.Join(r.enum, ", ")))
			}
			if r.min > 0 && fv.Kind() == reflect.Slice && fv.Len() < r.min {
				*problems = append(*problems, fmt.Errorf("%s: %d items, at least %d", at, fv.Len(), r.min))
			}
			validate(fv, at, problems)
		}
		if c, ok := v.Interface().(Checker); ok {
			for _, err := range c.Check() {
				*problems = append(*problems, fmt.Errorf("%s: %w", orRoot(path), err))
			}
		}
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			validate(v.Index(i), fmt.Sprintf("%s[%d]", path, i), problems)
		}
	}
}

func join(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

func orRoot(path string) string {
	if path == "" {
		return "(root)"
	}
	return path
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if y == x {
			return true
		}
	}
	return false
}

// Schema is the JSON Schema of t's type, from the same tags Validate reads. `desc` tags become
// descriptions, so the schema documents the contract for whoever writes it.
func Schema(v any, title string) map[string]any {
	s := schemaOf(reflect.TypeOf(v))
	s["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	s["title"] = title
	return s
}

func schemaOf(t reflect.Type) map[string]any {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Int, reflect.Int64:
		return map[string]any{"type": "integer"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Slice:
		return map[string]any{"type": "array", "items": schemaOf(t.Elem())}
	case reflect.Struct:
		props := map[string]any{}
		var required []string
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			name := jsonName(f)
			p := schemaOf(f.Type)
			r := parse(f.Tag.Get("check"))
			if len(r.enum) > 0 {
				p["enum"] = r.enum
			}
			if r.min > 0 {
				p["minItems"] = r.min
			}
			if r.required {
				required = append(required, name)
				if f.Type.Kind() == reflect.String {
					p["minLength"] = 1
				}
			}
			if d := f.Tag.Get("desc"); d != "" {
				p["description"] = d
			}
			props[name] = p
		}
		s := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
		if len(required) > 0 {
			s["required"] = required
		}
		return s
	}
	return map[string]any{}
}
