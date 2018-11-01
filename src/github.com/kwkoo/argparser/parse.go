package argparser

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"unsafe"
)

// Parse will take in a pointer to a struct and set each field to a value in
// the environment or a flag from the command line. The environment variable
// will always take precedence over the command line flag.
//
// If a field is of type bool, it will be set to true as long as the
// corresponding environment variable is set, irrespective of the environment
// variable's value.
//
// Set the appropriate tag in each field to tell Parse how to handle the field.
// Parse accepts the following tags: env, flag, default, usage.
//
// The env tag specifies the environment variable name which corresponds to
// the field. If this is not specified, Parse uses the uppercase version of
// the field name.
//
// The flag tag specifies the command line flag name which corresponds to the
// field. If this is not specified, Parse uses the lowercase version of the
// field name.
//
// The default tag specifies a default value for the field. This value is used
// if the corresponding environment variable and command line flag do not
// exist.
//
// The usage tag specifies the usage text for the command line flag.
//
func Parse(ptrtostruct interface{}) error {
	ptrtostructval := reflect.ValueOf(ptrtostruct)
	if ptrtostructval.Kind() != reflect.Ptr {
		return fmt.Errorf("argument must be a pointer to struct - got %v instead", ptrtostructval.Kind())
	}

	structval := ptrtostructval.Elem()
	if structval.Kind() != reflect.Struct {
		return fmt.Errorf("argument must be a pointer to struct - got a pointer to %v instead", structval.Kind())
	}

	var dummyflag string
	parseflags := false
	structtype := structval.Type()
	fieldcount := structtype.NumField()
	for i := 0; i < fieldcount; i++ {
		structfield := structtype.FieldByIndex([]int{i})
		structfieldkind := structfield.Type.Kind()

		// We only support fields of type string, int, and bool.
		if structfieldkind != reflect.String && structfieldkind != reflect.Int && structfieldkind != reflect.Bool {
			continue
		}

		// Skip invalid fields and fields that cannot be set.
		field := structval.FieldByIndex([]int{i})
		if !field.IsValid() || !field.CanSet() {
			continue
		}

		envkey := structfield.Tag.Get("env")
		if len(envkey) == 0 {
			envkey = strings.ToUpper(structfield.Name)
		}
		flagkey := structfield.Tag.Get("flag")
		if len(flagkey) == 0 {
			flagkey = strings.ToLower(structfield.Name)
		}
		envval, envkeyexists := os.LookupEnv(envkey)
		if envkeyexists {
			if structfieldkind == reflect.String {
				field.SetString(envval)
			} else if structfieldkind == reflect.Int {
				val, err := strconv.Atoi(envval)
				if err != nil {
					return fmt.Errorf("environment variable %v must be an integer - instead it is: %v", envkey, envval)
				}
				field.SetInt(int64(val))
			} else if structfieldkind == reflect.Bool {
				field.SetBool(true)
			}

			// Bypass flag provided but not defined error from flag package.
			flag.StringVar(&dummyflag, flagkey, "", "")
			continue
		}

		// Skip field if this field cannot be converted to a pointer (necessary
		// for flag call).
		if !field.CanAddr() {
			continue
		}

		usage := structfield.Tag.Get("usage")
		defaultval := structfield.Tag.Get("default")

		if structfieldkind == reflect.String {
			parseflags = true
			flag.StringVar((*string)(unsafe.Pointer(field.Addr().Pointer())), flagkey, defaultval, usage)
		} else if structfieldkind == reflect.Int {
			parseflags = true
			var converteddefault int
			if len(defaultval) > 0 {
				var err error
				converteddefault, err = strconv.Atoi(defaultval)
				if err != nil {
					return fmt.Errorf("field %v is of type int but the default tag is not an int: %v", flagkey, defaultval)
				}
			}
			flag.IntVar((*int)(unsafe.Pointer(field.Addr().Pointer())), flagkey, converteddefault, usage)
		} else if structfieldkind == reflect.Bool {
			parseflags = true
			var converteddefault bool
			if len(defaultval) > 0 {
				converteddefault = true
			}
			flag.BoolVar((*bool)(unsafe.Pointer(field.Addr().Pointer())), flagkey, converteddefault, usage)
		}
	}
	if parseflags {
		flag.Parse()
	}

	return nil
}
