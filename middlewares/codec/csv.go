package codec

import (
	"encoding/csv"
	"fmt"
	"reflect"
)

// CSVEncoder is a custom goahttp.Encoder. See docs for ResponseEncoder.
type CSVEncoder struct {
	enc *csv.Writer
}

// Encode implements the goahttp.Encoder interface
func (e *CSVEncoder) Encode(v interface{}) error {
	switch reflect.TypeOf(v).Kind() {
	case reflect.Interface, reflect.Ptr:
		return e.Encode(reflect.ValueOf(v).Elem().Interface())
	case reflect.Slice:
		return e.enc.WriteAll(recordizeSlice(v))
	case reflect.Map:
		return e.enc.WriteAll(recordizeMap(v))
	case reflect.Struct:
		data := reflect.ValueOf(v).FieldByName("Data")
		if data != (reflect.Value{}) {
			return e.Encode(data.Interface())
		}
		return nil
	default:
		return nil
	}
}

// Convert an interface{} containing a map into [][]string.
func recordizeMap(input interface{}) [][]string {
	object := reflect.ValueOf(input)
	values := object.MapRange()
	var record []string
	var header []string
	for values.Next() {
		header = append(header, fmt.Sprintf("%v", values.Key().Interface()))
		record = append(record, fmt.Sprintf("%v", values.Value().Interface()))
	}
	return [][]string{header, record}
}

// Convert an interface{} containing a slice of structs into [][]string.
func recordizeSlice(input interface{}) [][]string {
	if strs, isStringsSlice := input.([][]string); isStringsSlice {
		return strs
	}
	var records [][]string
	var header []string // The first record in records will contain the names of the fields
	object := reflect.ValueOf(input)

	if object.Len() == 0 {
		return nil
	}

	// The first record in the records slice should contain headers / field names
	first := object.Index(0).Elem()

	typ := first.Type()

	for i := 0; i < typ.NumField(); i++ {
		f := typ.Field(i).Type
		if f.Kind() == reflect.Ptr {
			f = f.Elem()
		}
		// The field is a nested struct: add header elements for its fields, too
		if f.Kind() == reflect.Struct {
			for j := 0; j < f.NumField(); j++ {
				header = append(header, typ.Field(i).Name+" "+f.Field(j).Name)
			}
		} else {
			header = append(header, typ.Field(i).Name)
		}
	}

	// append header to final CSV
	records = append(records, header)

	// Make a slice of objects to iterate through & populate the string slice
	var items []interface{}
	for i := 0; i < object.Len(); i++ {
		items = append(items, object.Index(i).Elem().Interface())
	}

	// Populate the rest of the items into <records>
	for _, v := range items {
		item := reflect.ValueOf(v)
		var record []string
		for i := 0; i < item.NumField(); i++ {
			// Fetch underlying value
			val := reflect.Indirect(item.Field(i))
			// Fetch underlying type
			f := typ.Field(i).Type
			if f.Kind() == reflect.Ptr {
				f = f.Elem()
			}

			// If it's a nested struct, read fields
			if f.Kind() == reflect.Struct {
				for j := 0; j < f.NumField(); j++ {
					var itm interface{} = ""
					if val != (reflect.Value{}) {
						fieldVal := reflect.Indirect(val.Field(j))
						if fieldVal != (reflect.Value{}) {
							itm = fieldVal.Interface()
						}
					}
					record = append(record, fmt.Sprintf("%+v", itm))
				}
			} else { // Otherwise just read the value
				var itm interface{} = ""
				if val != (reflect.Value{}) {
					itm = val.Interface()
				}
				record = append(record, fmt.Sprintf("%+v", itm))
			}
		}
		records = append(records, record)
	}
	return records
}
