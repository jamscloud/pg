package orm

import (
	"reflect"

	"github.com/go-pg/pg/types"
)

func indirectType(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func sliceElemType(v reflect.Value) reflect.Type {
	elemType := v.Type().Elem()
	if elemType.Kind() == reflect.Interface && v.Len() > 0 {
		return reflect.Indirect(v.Index(0).Elem()).Type()
	} else {
		return indirectType(elemType)
	}
}

func typeByIndex(t reflect.Type, index []int) reflect.Type {
	for _, x := range index {
		switch t.Kind() {
		case reflect.Ptr:
			t = t.Elem()
		case reflect.Slice:
			t = indirectType(t.Elem())
		}
		t = t.Field(x).Type
	}
	return indirectType(t)
}

func fieldByIndex(v reflect.Value, index []int) reflect.Value {
	for i, x := range index {
		if i > 0 {
			v = indirectNew(v)
		}
		v = v.Field(x)
	}
	return v
}

func indirectNew(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

func columns(b []byte, table types.Q, prefix string, fields []*Field) []byte {
	for i, f := range fields {
		if i > 0 {
			b = append(b, ", "...)
		}

		if len(table) > 0 {
			b = append(b, table...)
			b = append(b, '.')
		}
		b = types.AppendField(b, prefix+f.SQLName, 1)
	}
	return b
}

func walk(v reflect.Value, index []int, fn func(reflect.Value)) {
	v = reflect.Indirect(v)
	switch v.Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			visitField(v.Index(i), index, fn)
		}
	default:
		visitField(v, index, fn)
	}
}

func visitField(v reflect.Value, index []int, fn func(reflect.Value)) {
	v = reflect.Indirect(v)
	if len(index) > 0 {
		v = v.Field(index[0])
		if v.Kind() == reflect.Ptr && v.IsNil() {
			return
		}
		walk(v, index[1:], fn)
	} else {
		fn(v)
	}
}

func appendChildValues(b []byte, v reflect.Value, index []int, fields []*Field) []byte {
	seen := make(map[string]struct{})
	walk(v, index, func(v reflect.Value) {
		start := len(b)

		b = append(b, '(')
		for i, f := range fields {
			if i > 0 {
				b = append(b, ", "...)
			}
			b = f.AppendValue(b, v, 1)
		}
		b = append(b, "), "...)

		if _, ok := seen[string(b[start:])]; ok {
			b = b[:start]
		} else {
			seen[string(b[start:])] = struct{}{}
		}
	})
	if len(seen) > 0 {
		b = b[:len(b)-2] // trim ", "
	}
	return b
}

func dstValues(model tableModel, fields []*Field) map[string][]reflect.Value {
	mp := make(map[string][]reflect.Value)
	var id []byte
	walk(model.Root(), model.ParentIndex(), func(v reflect.Value) {
		id = modelId(id[:0], v, fields)
		mp[string(id)] = append(mp[string(id)], v.FieldByIndex(model.Relation().Field.Index))
	})
	return mp
}

func modelId(b []byte, v reflect.Value, fields []*Field) []byte {
	for i, f := range fields {
		if i > 0 {
			b = append(b, ',')
		}
		b = f.AppendValue(b, v, 0)
	}
	return b
}

func modelIdMap(b []byte, m map[string]string, prefix string, fields []*Field) []byte {
	for i, f := range fields {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, m[prefix+f.SQLName]...)
	}
	return b
}

func appendColumns(b []byte, fields []*Field) []byte {
	for i, f := range fields {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = append(b, f.Column...)
	}
	return b
}

func appendTableColumns(b []byte, table *Table) []byte {
	for i, f := range table.Fields {
		if i > 0 {
			b = append(b, ", "...)
		}
		b = append(b, table.Alias...)
		b = append(b, '.')
		b = append(b, f.Column...)
	}
	return b
}
