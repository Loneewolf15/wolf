package ir

import "reflect"

// DeepClone creates a deep copy of a WIR node using reflection.
// This is used for LLVM monomorphization (instantiating templates).
func DeepClone[T any](v T) T {
	res := cloneValue(reflect.ValueOf(v))
	return res.Interface().(T)
}

func cloneValue(v reflect.Value) reflect.Value {
	if !v.IsValid() {
		return v
	}

	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return v
		}
		// Special case: we don't clone irStmt/irExpr interface methods.
		elem := cloneValue(v.Elem())
		ptr := reflect.New(v.Type().Elem())
		ptr.Elem().Set(elem)
		return ptr

	case reflect.Struct:
		cloned := reflect.New(v.Type()).Elem()
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			// Skip unexported fields
			if v.Type().Field(i).PkgPath != "" {
				continue
			}
			cloned.Field(i).Set(cloneValue(field))
		}
		return cloned

	case reflect.Slice:
		if v.IsNil() {
			return v
		}
		cloned := reflect.MakeSlice(v.Type(), v.Len(), v.Cap())
		for i := 0; i < v.Len(); i++ {
			cloned.Index(i).Set(cloneValue(v.Index(i)))
		}
		return cloned

	case reflect.Map:
		if v.IsNil() {
			return v
		}
		cloned := reflect.MakeMap(v.Type())
		for _, key := range v.MapKeys() {
			cloned.SetMapIndex(cloneValue(key), cloneValue(v.MapIndex(key)))
		}
		return cloned

	case reflect.Interface:
		if v.IsNil() {
			return v
		}
		// Extract the underlying value, clone it, and wrap it back in an empty interface value
		// then convert to the specific interface type.
		elem := cloneValue(v.Elem())
		// reflect.Value representing the interface wrapper around the cloned element.
		// e.g. if v contains an *ir.Ident, elem is an *ir.Ident, and we need ir.Expr
		// But in Go >= 1.18 we can just use Value.Convert
		return elem.Convert(v.Type())

	default:
		// basic types (int, string, bool, etc.) are passed by value
		return v
	}
}

// ReplaceTypeNames recursively replaces WIR type strings based on the environment map.
// This executes the monomorphization substitution "T" -> "int".
func ReplaceTypeNames(v any, typeEnv map[string]string) {
	replaceValuesHelper(reflect.ValueOf(v), typeEnv)
}

func replaceValuesHelper(v reflect.Value, typeEnv map[string]string) {
	if !v.IsValid() {
		return
	}

	switch v.Kind() {
	case reflect.Ptr, reflect.Interface:
		if !v.IsNil() {
			replaceValuesHelper(v.Elem(), typeEnv)
		}

	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			field := v.Field(i)
			if !field.CanSet() {
				// unexported
				continue
			}
			fieldName := v.Type().Field(i).Name

			// Replace type annotations in specific fields
			switch fieldName {
			case "Type", "ElemType", "KeyType", "ValueType", "TypeName":
				if field.Kind() == reflect.String {
					if newType, ok := typeEnv[field.String()]; ok {
						field.SetString(newType)
					}
				}
			case "ReturnTypes":
				if field.Kind() == reflect.Slice && field.Type().Elem().Kind() == reflect.String {
					for j := 0; j < field.Len(); j++ {
						elem := field.Index(j)
						if newType, ok := typeEnv[elem.String()]; ok {
							elem.SetString(newType)
						}
					}
				}
			}

			// Recurse into children
			replaceValuesHelper(field, typeEnv)
		}

	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			replaceValuesHelper(v.Index(i), typeEnv)
		}
	}
}
