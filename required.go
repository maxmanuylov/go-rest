package rest

import (
    "fmt"
    "reflect"
    "strings"
)

var requiredDataPrefix = "required@"

func checkRequiredFields(item interface{}, action string) error {
    if fieldPath := getMissedValuePath(reflect.ValueOf(item), action); fieldPath != "" {
        return fmt.Errorf("Filed is not specified: %s", fieldPath)
    }
    return nil
}

func getMissedValuePath(value reflect.Value, action string) string {
    switch value.Kind() {
    case reflect.Ptr:
        return getMissedValuePath(value.Elem(), action)

    case reflect.Array, reflect.Slice:
        for i := 0; i < value.Len(); i++ {
            if subPath := getMissedValuePath(value.Index(i), action); subPath != "" {
                return fmt.Sprintf("[%d]%s", i, subPath)
            }
        }

    case reflect.Map:
        for _, key := range value.MapKeys() {
            if subPath := getMissedValuePath(value.MapIndex(key), action); subPath != "" {
                return fmt.Sprintf("[%v]%s", value.Interface(), subPath)
            }
        }

    case reflect.Struct:
        valueType := value.Type()

        for i := 0; i < value.NumField(); i++ {
            field := value.Field(i)
            fieldType := valueType.Field(i)

            if isZero(field) {
                if isRequired(fieldType, action) {
                    return fmt.Sprintf(".%s", getFieldName(fieldType))
                }
            } else {
                if subPath := getMissedValuePath(field, action); subPath != "" {
                    return fmt.Sprintf(".%s%s", getFieldName(fieldType), subPath)
                }
            }
        }
    }

    return ""
}

func isZero(value reflect.Value) bool {
    return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

func isRequired(fieldType reflect.StructField, action string) bool {
    for _, data := range strings.Split(fieldType.Tag.Get("rest"), ",") {
        data = strings.TrimSpace(data)
        if strings.HasPrefix(data, requiredDataPrefix) {
            return contains(strings.Split(data[len(requiredDataPrefix):], ":"), action)
        }
    }
    return false
}

func contains(actions []string, action string) bool {
    ourAction := strings.ToLower(action)
    for _, _action := range actions {
        if ourAction == strings.ToLower(_action) {
            return true
        }
    }
    return false
}

func getFieldName(fieldType reflect.StructField) string {
    if jsonTag := strings.TrimSpace(fieldType.Tag.Get("json")); jsonTag != "" && jsonTag != "-" {
        if jsonName := strings.TrimSpace(strings.SplitN(jsonTag, ",", 2)[0]); jsonName != "" {
            return jsonName
        }
    }
    return fieldType.Name
}
