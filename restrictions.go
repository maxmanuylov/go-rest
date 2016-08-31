package rest

import (
    "fmt"
    "reflect"
    "strings"
)

const (
    oneOfPrefix    = "#"
    nonEmptyArray  = "nonempty"
    requiredPrefix = "required@"
)

type problem int

const (
    itemNotSpecified      problem = iota
    severalItemsSpecified
    emptyArray
)

func checkRestrictions(item interface{}, action string) error {
    if fields := getProblemFields(reflect.ValueOf(item), action, false); fields != nil {
        if len(fields.paths) == 1 {
            if fields.problem == emptyArray {
                return fmt.Errorf("Array is empty: %s", fields.paths[0])
            } else { // itemNotSpecified
                return fmt.Errorf("Field is not specified: %s", fields.paths[0])
            }
        } else if fields.problem == severalItemsSpecified {
            return fmt.Errorf("Only one of the following fields can be specified:\n%s", strings.Join(fields.paths, "\n"))
        } else { // itemNotSpecified
            return fmt.Errorf("One of the following fields must be specified:\n%s", strings.Join(fields.paths, "\n"))
        }
    }
    return nil
}

func getProblemFields(value reflect.Value, action string, checkArrayIsNotEmpty bool) *problemFields {
    switch value.Kind() {
    case reflect.Ptr:
        return getProblemFields(value.Elem(), action, checkArrayIsNotEmpty)

    case reflect.Array, reflect.Slice:
        if checkArrayIsNotEmpty && value.Len() == 0 {
            return &problemFields{
                paths: []string{""},
                problem: emptyArray,
            }
        }
        for i := 0; i < value.Len(); i++ {
            if fields := getProblemFields(value.Index(i), action, checkArrayIsNotEmpty); fields != nil {
                return fields.withPrefix(fmt.Sprintf("[%d]", i))
            }
        }

    case reflect.Map:
        for _, key := range value.MapKeys() {
            if fields := getProblemFields(value.MapIndex(key), action, false); fields != nil {
                return fields.withPrefix(fmt.Sprintf("[%v]", value.Interface()))
            }
        }

    case reflect.Struct:
        valueType := value.Type()
        oneOfIndex := make(map[string]*oneOfData)

        for i := 0; i < value.NumField(); i++ {
            field := value.Field(i)
            fieldType := valueType.Field(i)
            r := getRestrictions(fieldType, action)

            if isZero(field) {
                if r.oneOfKey != "" {
                    getOrCreateOneOf(oneOfIndex, r.oneOfKey).addZeroField(getFieldName(fieldType), r.required)
                } else if r.required {
                    return &problemFields{
                        paths: []string{fmt.Sprintf(".%s", getFieldName(fieldType))},
                        problem: itemNotSpecified,
                    }
                }
            } else {
                if r.oneOfKey != "" {
                    getOrCreateOneOf(oneOfIndex, r.oneOfKey).addSpecifiedField(getFieldName(fieldType), r.required)
                }
                if fields := getProblemFields(field, action, r.nonEmptyArray); fields != nil {
                    return fields.withPrefix(fmt.Sprintf(".%s", getFieldName(fieldType)))
                }
            }
        }

        for _, data := range oneOfIndex {
            if fields := data.checkProblems(); fields != nil {
                return fields.withPrefix(".")
            }
        }
    }

    return nil
}

func getOrCreateOneOf(oneOfIndex map[string]*oneOfData, oneOfKey string) *oneOfData {
    data, ok := oneOfIndex[oneOfKey]
    if !ok {
        data = &oneOfData{}
        oneOfIndex[oneOfKey] = data
    }
    return data
}

func isZero(value reflect.Value) bool {
    return reflect.DeepEqual(value.Interface(), reflect.Zero(value.Type()).Interface())
}

func getRestrictions(fieldType reflect.StructField, action string) *restrictions {
    r := &restrictions{}

    for _, data := range strings.Split(fieldType.Tag.Get("rest"), ",") {
        data = strings.TrimSpace(data)
        if strings.HasPrefix(data, oneOfPrefix) {
            r.oneOfKey = data[len(oneOfPrefix):]
        } else if strings.HasPrefix(data, requiredPrefix) {
            if actionIsSuitable(data[len(requiredPrefix):], action) {
                r.required = true
            }
        } else if data == nonEmptyArray {
            r.nonEmptyArray = true
        }
    }

    return r
}

func actionIsSuitable(actionSpec, action string) bool {
    return actionSpec == "*" || contains(strings.Split(actionSpec, ":"), action)
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

/* *** */

type oneOfData struct {
    fieldNames []string
    specified  int
    required   bool
}

func (data *oneOfData) addSpecifiedField(fieldName string, required bool) {
    data.addZeroField(fieldName, required)
    data.specified++
}

func (data *oneOfData) addZeroField(fieldName string, required bool) {
    data.fieldNames = append(data.fieldNames, fieldName)
    if required {
        data.required = true
    }
}

func (data *oneOfData) checkProblems() *problemFields {
    if data.specified == 0 {
        if data.required {
            return data.toProblem(itemNotSpecified)
        }
    } else if data.specified > 1 {
        return data.toProblem(severalItemsSpecified)
    }
    return nil
}

func (data *oneOfData) toProblem(problem problem) *problemFields {
    return &problemFields{
        paths: data.fieldNames,
        problem: problem,
    }
}

/* *** */

type restrictions struct {
    nonEmptyArray bool
    required      bool
    oneOfKey      string
}

/* *** */

type problemFields struct {
    paths   []string
    problem problem
}

func (fields *problemFields) withPrefix(prefix string) *problemFields {
    newFields := &problemFields{
        paths: make([]string, len(fields.paths)),
        problem: fields.problem,
    }

    for i, path := range fields.paths {
        newFields.paths[i] = fmt.Sprintf("%s%s", prefix, path)
    }

    return newFields
}
