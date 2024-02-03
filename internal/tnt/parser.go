package tnt

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

type noop struct{}

const (
	TAG_NAME    = "tnt"
	TAG_REQUIRE = "require"
)

var (
	ErrNotEnoughFields    = errors.New("not enough fields")
	ErrNotEnoughVariables = errors.New("not enough destination variables")
)

func ConvertReplyToMap(data interface{}) map[string]interface{} {
	retval := make(map[string]interface{})
	if val, ok := data.(map[interface{}]interface{}); ok {
		for k, v := range val {
			if kstr, ok := k.(string); ok {
				retval[kstr] = v
			}
		}
	}
	return retval
}

func ConvertReplyToSlice(data interface{}) []string {
	retval := make([]string, 0)
	if val, ok := data.([]string); ok {
		retval = append(retval, val...)
		return retval
	}
	if val, ok := data.([]interface{}); ok {
		for _, v := range val {
			if vstr, ok := v.(string); ok {
				retval = append(retval, vstr)
			}
		}
	}
	return retval
}

/** SerializeReply convert map[inteface] to map[string]*/
func SerializeReply(v interface{}) (interface{}, error) {
	what := reflect.TypeOf(v)
	if what == nil {
		return nil, nil
	}
	val := reflect.ValueOf(v)
	switch what.Kind() {
	case reflect.Array, reflect.Slice:
		sarr := val.Len()
		array := make([]interface{}, 0)
		for i := 0; i < sarr; i++ {
			tmp, err := SerializeReply(val.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			array = append(array, tmp)
		}
		return array, nil
	case reflect.Struct, reflect.Chan:
		return nil, errors.New("dont support type")
	case reflect.Map:
		rmap := make(map[string]interface{})

		for _, k := range val.MapKeys() {
			tmp_key, err := SerializeReply(k.Interface())
			if err != nil {
				return nil, err
			}
			tmp_val, err := SerializeReply(val.MapIndex(k).Interface())
			if err != nil {
				return nil, err
			}

			rmap[fmt.Sprintf("%v", tmp_key)] = tmp_val
		}

		return rmap, nil
	}
	return v, nil
}

// StructToTntArray - scan struct and convert it to []interface{}
// ScanFieldsToStruct - scans tnt data fields to structure use `tnt:"N"` to specify
// the array element index to map. All given indices are optinal unless `require` keyword is given.
//
//	 type A struct {
//	   A0   string            `tnt:"0,require"`
//			A1   int64             `tnt:"1,require"`
//			A2   uint64            `tnt:"2"`
//			A3   int               `tnt:"3"`
//	 }
func StructToTntArray(src any) (fields []any, err error) {
	v := reflect.ValueOf(src)
	if v.Kind() != reflect.Ptr {
		err = errors.New("dst sounld be a *struct")
		return
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		err = errors.New("dst sounld be a *struct")
		return
	}

	t := v.Type()

	maxIndex := 0

	tmpField := make([]any, 0, t.NumField())

	fields = make([]any, t.NumField())

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get(TAG_NAME)
		if tag == "" {
			continue
		}
		tagParts := strings.Split(tag, ",")
		if len(tagParts) == 0 {
			continue
		}
		index, er := strconv.Atoi(tagParts[0])
		if er != nil {
			continue
		}
		if index > maxIndex {
			maxIndex = index
		}
		if index >= t.NumField() {
			err = errors.New("src count fields < index")
			return
		}
		tmpField = append(tmpField, v.Field(i).Interface())
		fields[index] = v.Field(i).Interface()
	}

	if len(tmpField) != maxIndex+1 {
		err = errors.New("src count count data")
	}
	fields = fields[:maxIndex+1]
	return
}

// ScanFieldsToStruct - scans tnt data fields to structure use `tnt:"N"` to specify
// the array element index to map. All given indices are optinal unless `require` keyword is given.
//
//	 type A struct {
//	   A0   string            `tnt:"0,require"`
//			A1   int64             `tnt:"1,require"`
//			A2   uint64            `tnt:"2"`
//			A3   int               `tnt:"3"`
//	 }
func ScanFieldsToStruct(fields []interface{}, dst interface{}) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Ptr {
		return errors.New("dst sounld be a *struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errors.New("dst sounld be a *struct")
	}
	t := v.Type()

	require := 0
	strFields := make([]*reflect.Value, len(fields))

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get(TAG_NAME)
		if tag == "" {
			continue
		}
		tagParts := strings.Split(tag, ",")
		if len(tagParts) == 0 {
			continue
		}
		index, err := strconv.Atoi(tagParts[0])
		if err != nil {
			continue
		}

		if len(tagParts) > 1 {
			for _, v := range tagParts[1:] {
				if v == TAG_REQUIRE && require < index {
					require = index
				}
			}
		}

		val := v.Field(i)
		if len(strFields) <= index {
			strFields = append(strFields, make([]*reflect.Value, index-len(strFields)+5)...)
		}
		strFields[index] = &val
	}

	var dummy noop
	args := make([]interface{}, len(strFields))
	for i, f := range strFields {
		if f != nil {
			args[i] = f.Addr().Interface()
		} else {
			args[i] = dummy
		}
	}

	return ScanFields(fields, require, args...)
}

func ScanFieldsAnyToStruct(fields any, dst interface{}) error {
	f, ok := fields.([]any)
	if !ok {
		return errors.New("fields canbe array interface")
	}
	return ScanFieldsToStruct(f, dst)
}

// ScanFields - scan tnt fields array into dst variables. Require option spicifies
// number of obligatory fields. Function returns an `ErrNotEnoughFields` otherwise.
func ScanFields(fields []interface{}, require int, dst ...interface{}) error {
	if len(fields) < require {
		return ErrNotEnoughFields
	}
	if len(dst) < require {
		return ErrNotEnoughVariables
	}

	for i, f := range fields {
		if i+1 > len(dst) {
			return nil
		}

		switch d := dst[i].(type) {
		case *string:
			x, ok := StringOrIntToString(f)
			if !ok {
				return fmt.Errorf("field #%d `%v`", i, fields)
			}
			*d = x
		case *int:
			x, ok := IntOrStringToInt(f)
			if !ok {
				return fmt.Errorf("field #%d `%#v` convert to int", i, fields[i])
			}
			*d = int(x)
		case *int64:
			x, ok := IntOrStringToInt(f)
			if !ok {
				return fmt.Errorf("field #%d `%#v` convert to int64", i, fields[i])
			}
			*d = x
		case *uint:
			x, ok := IntOrStringToUint(f)
			if !ok {
				return fmt.Errorf("field #%d `%#v` convert to uint", i, fields[i])
			}
			*d = uint(x)
		case *uint64:
			x, ok := IntOrStringToUint(f)
			if !ok {
				return fmt.Errorf("field #%d `%#v` convert to uint64", i, fields[i])
			}
			*d = x
		case *map[string]string:
			x, ok := MapToMapStrings(f)
			if !ok {
				return fmt.Errorf("field #%d `%#v` convert to map[string]string", i, fields[i])
			}
			*d = x
		case *[]string:
			*d = ConvertReplyToSlice(f)
		case *[]interface{}:
			x, ok := f.([]interface{})
			if !ok {
				return fmt.Errorf("field #%d `%v`", i, fields)
			}
			*d = x
		case noop:
			// do nothing
		default:
			// wrong dst type
			return fmt.Errorf("unknown destination #%d type, %t", i, dst[i])
		}
	}

	return nil
}

func MapToMapStrings(field interface{}) (map[string]string, bool) {
	maps := make(map[string]string)
	if mapsUni, ok := field.(map[interface{}]interface{}); ok {
		for k, v := range mapsUni {
			if v1, ok1 := StringOrIntToString(v); ok1 {
				k1, _ := k.(string)
				maps[k1] = v1
			}
		}
	}
	return maps, true
}

func IntOrStringToInt(field interface{}) (int64, bool) {
	var ret int64
	var ok bool
	if ret, ok = field.(int64); !ok {
		var retstr string
		if retstr, ok = field.(string); ok {
			i, _ := strconv.Atoi(retstr)
			ret = int64(i)
		} else {
			var retfloat float64
			if retfloat, ok = field.(float64); ok {
				ret = int64(retfloat)
			}
		}
	}
	return ret, ok
}

func IntOrStringToUint(field interface{}) (uint64, bool) {
	var ret uint64
	var ok bool
	if ret, ok = field.(uint64); !ok {
		var retstr string
		if retstr, ok = field.(string); ok {
			i, _ := strconv.Atoi(retstr)
			ret = uint64(i)
		} else {
			var retfloat float64
			if retfloat, ok = field.(float64); ok {
				ret = uint64(retfloat)
			}
		}
	}
	return ret, ok
}

func StringOrIntToString(field interface{}) (string, bool) {
	var ret string
	var ok bool
	if ret, ok = field.(string); !ok {
		var retInt uint64
		if retInt, ok = field.(uint64); ok {
			ret = fmt.Sprintf("%d", retInt)
		} else {
			var retfloat float64
			if retfloat, ok = field.(float64); ok {
				ret = fmt.Sprintf("%d", uint64(retfloat))
			}
		}
	}
	return ret, ok
}

// SliceIntefacesToStrings - converts slice of inteface{} to slice of strings
func SliceIntefacesToStrings(val []interface{}) (result []string, err error) {
	result = make([]string, len(val))
	for i, el := range val {
		switch x := el.(type) {
		case string:
			result[i] = x
		case int64:
			result[i] = strconv.FormatInt(x, 10)
		case uint64:
			result[i] = strconv.FormatUint(x, 10)
		case float64:
			result[i] = strconv.FormatFloat(x, 'f', -1, 64)
		case bool:
			result[i] = strconv.FormatBool(x)
		default:
			return nil, fmt.Errorf("value #%d @ %T, %v", i, el, val)
		}
	}

	return
}

// MapToTarantoolArgs - converts map[string]interface to []interface{} for our standart function un tarantool
func MapToTarantoolArgs(m map[string]interface{}) []interface{} {
	data := make([]interface{}, 0, len(m)*2)
	for k, v := range m {
		data = append(data, k, v)
	}
	return data
}
