package struct2flag

import (
	"flag"
	"reflect"
	"strings"
)

func Bind(fs *flag.FlagSet, cfg interface{}) {
	v := reflect.ValueOf(cfg).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		field := t.Field(i)
		name := strings.ToLower(field.Name)
		desc, ok := field.Tag.Lookup("flag")
		if !ok {
			continue
		}
		if first, rest, ok := strings.Cut(desc, ","); ok {
			name = first
			desc = rest
		}
		switch f.Kind() {
		case reflect.Bool:
			fs.BoolVar(f.Addr().Interface().(*bool), name, f.Bool(), desc)
		case reflect.Int:
			fs.IntVar(f.Addr().Interface().(*int), name, int(f.Int()), desc)
		case reflect.String:
			fs.StringVar(f.Addr().Interface().(*string), name, f.String(), desc)
		}
	}
}

func BindDefault(cfg interface{}) {
	Bind(flag.CommandLine, cfg)
}
