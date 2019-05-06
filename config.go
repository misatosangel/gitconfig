// Copyright 2018-2019 "Misato's Angel" <misatos.arngel@gmail.com>.  All rights reserved.
// Use of this source code is governed the MIT license.
// license that can be found in the LICENSE file.
package gitconfig

import (
	"bufio"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

type Config struct {
	Sections   map[string]*ConfigSection
	BaseValues ConfigValueSet
	Imports    []string
}

type ConfigSection struct {
	Name         string
	OrigCaseName string
	SubSections  map[string]*ConfigSubSection
	Values       ConfigValueSet
}

type ConfigSubSection struct {
	Name   string
	Values ConfigValueSet
}

type ConfigValue struct {
	Name         string
	OrigCaseName string
	Value        []*string
}

type ConfigValueSet map[string]*ConfigValue

var durationType = reflect.TypeOf((*time.Duration)(nil)).Elem()

func NewConfig() *Config {
	return &Config{
		Sections:   make(map[string]*ConfigSection, 10),
		BaseValues: make(ConfigValueSet, 10),
		Imports:    make([]string, 0, 5),
	}
}

func NewConfigFromString(data string) (*Config, error) {
	r := strings.NewReader(data)
	p := Parser{
		Reader: bufio.NewScanner(r),
		Config: NewConfig(),
	}
	err := p.Read()
	if err != nil {
		return nil, err
	}
	return p.Config, nil
}

func NewConfigFromFile(file string) (*Config, error) {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return nil, err
	}
	fh, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	p := Parser{
		Reader: bufio.NewScanner(fh),
		Config: NewConfig(),
	}

	err = p.Read()
	if err != nil {
		return nil, err
	}
	return p.Config, nil
}

func (self *Config) String() string {
	out := self.BaseValues.String()
	for _, s := range self.Sections {
		out += s.String()
	}
	return out
}

// Load loads git config values to a struct annotated with "gitconfig" tags.
func (self *Config) Load(v interface{}) error {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("Passed a non-pointer: %v\n", v)
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return fmt.Errorf("Passed a pointer to a non-struct: %v\n", v)
	}
	return self.loadStruct(rv, "")
}

func (self *Config) loadSetValue(retval reflect.Value, key, defVal string, confVal *ConfigValue, required, haveDefault bool) error {
	tp := retval.Type()
	if tp == durationType {
		var s string
		if confVal == nil || !confVal.HasValues() {
			if required {
				return fmt.Errorf("Could not populate required %s no value for %s", tp.String(), key)
			}
			if !haveDefault {
				// leave existing value (if any) untouched
				return nil
			}
			s = defVal
		} else {
			s, _ = confVal.GetString()
		}
		parsed, err := time.ParseDuration(s)

		if err != nil {
			return fmt.Errorf("Could not parse value '%s' as duration for %s: %s\n", s, key, err.Error())
		}

		retval.SetInt(int64(parsed))
		return nil
	}
	switch tp.Kind() {
	case reflect.String:
		var s string
		if confVal == nil || !confVal.HasValues() {
			if required {
				return fmt.Errorf("Could not populate required %s no value for %s", tp.String(), key)
			}
			if !haveDefault {
				// leave existing value (if any) untouched
				return nil
			}
			s = defVal
		} else {
			s, _ = confVal.GetString()
		}
		retval.SetString(s)
		return nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var i uint64
		var err error
		if confVal == nil || !confVal.HasValues() {
			if required {
				return fmt.Errorf("Could not populate required %s no value for %s", tp.String(), key)
			}
			if !haveDefault {
				// leave existing value (if any) untouched
				return nil
			}
			i, err = strconv.ParseUint(defVal, 10, 64)
			if err != nil {
				return fmt.Errorf("Could not populate default %s field, default value %q did not parse as an Int", tp.String(), defVal)
			}
		} else {
			i, _, err = confVal.GetUint()
			if err != nil {
				return err
			}
		}
		retval.SetUint(i)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var i int64
		var err error
		if confVal == nil || !confVal.HasValues() {
			if required {
				return fmt.Errorf("Could not populate required %s no value for %s", tp.String(), key)
			}
			if !haveDefault {
				// leave existing value (if any) untouched
				return nil
			}
			i, err = strconv.ParseInt(defVal, 10, 64)
			if err != nil {
				return fmt.Errorf("Could not populate default %s field, default value %q did not parse as an Int", tp.String(), defVal)
			}
		} else {
			i, _, err = confVal.GetInt()
			if err != nil {
				return err
			}
		}
		retval.SetInt(i)

	case reflect.Bool:
		var b bool
		var err error
		if confVal == nil || !confVal.HasValues() {
			if required {
				return fmt.Errorf("Could not populate required %s no value for %s", tp.String(), key)
			}
			if !haveDefault {
				// leave existing value (if any) untouched
				return nil
			}
			b, err = strconv.ParseBool(defVal)
			if err != nil {
				return fmt.Errorf("Could not populate default %s field, default value %q did not parse as an Int", tp.String(), defVal)
			}
		} else {
			b, _, err = confVal.GetBool()
			if err != nil {
				return err
			}
		}
		retval.SetBool(b)

	case reflect.Slice:
		if confVal == nil || !confVal.HasValues() {
			if required {
				return fmt.Errorf("Could not populate required %s no value for %s", tp.String(), key)
			}
			if !haveDefault {
				// leave existing value (if any) untouched
				return nil
			}
			confVal = &ConfigValue{Value: []*string{&defVal}}
		}
		elemtp := tp.Elem()
		switch elemtp.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map:
			// slice of hash maps is not supported
			return fmt.Errorf("cannot populate field %s of type %s. Slices can only contain basic types.", key, elemtp.String())
		}

		for _, stringPtr := range confVal.Value {
			if stringPtr == nil {
				return fmt.Errorf("Could not populate %s null value for %s", tp.String(), key)
			}
			elemvalptr := reflect.New(elemtp)
			elemval := reflect.Indirect(elemvalptr)
			passConfVal := &ConfigValue{Value: []*string{stringPtr}}
			if err := self.loadSetValue(elemval, key, defVal, passConfVal, required, haveDefault); err != nil {
				return err
			}
			retval.Set(reflect.Append(retval, elemval))
		}
		return nil

	case reflect.Ptr:
		if confVal == nil || !confVal.HasValues() {
			if required {
				return fmt.Errorf("Could not populate required %s no value for %s", tp.String(), key)
			}
			if !haveDefault {
				// leave existing value (if any) untouched
				return nil
			}
			confVal = &ConfigValue{Value: []*string{&defVal}}
		}
		if retval.IsNil() {
			retval.Set(reflect.New(retval.Type().Elem()))
		}
		return self.loadSetValue(reflect.Indirect(retval), key, defVal, confVal, required, haveDefault)

	case reflect.Array:
		elemtp := tp.Elem()
		switch elemtp.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
			// slice of hash maps is not supported
			return fmt.Errorf("cannot populate field %s of type %s. Arrays can only contain basic types.", key, elemtp.String())
		}
		aLen := tp.Len()
		setLen := len(confVal.Value)
		if aLen < setLen {
			// get the last max values of the slice
			confVal = &ConfigValue{Name: confVal.Name, OrigCaseName: confVal.OrigCaseName, Value: confVal.Value[setLen-aLen:]}
			setLen = len(confVal.Value)
		}
		for i := 0; i < aLen; i++ {
			var passConfVal *ConfigValue
			if i < setLen {
				passConfVal = &ConfigValue{Value: []*string{confVal.Value[i]}}
			}
			valPtr := retval.Index(i)
			if err := self.loadSetValue(valPtr, key, defVal, passConfVal, required, haveDefault); err != nil {
				return err
			}
		}
		return nil

	case reflect.Map:
		elemtp := tp.Elem()
		kTp := tp.Key()
		switch kTp.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.Struct:
			return fmt.Errorf("cannot populate field %s of type map[%s]%s. Map keys can only contain basic types.", key, kTp.String(), elemtp.String())
		}
		amStruct := false
		sName := ""
		sKey := ""
		switch elemtp.Kind() {
		case reflect.Map:
			return fmt.Errorf("cannot populate field %s of type map[%s]%s. Map values cannot be another maps.", key, kTp.String(), elemtp.String())
		case reflect.Struct:
			amStruct = true
			keyLen := len(key)
			if strings.HasSuffix(key, ".*.") {
				sName = key[0 : keyLen-3]
			} else if strings.HasSuffix(key, ".*") {
				sName = key[0 : keyLen-2]
			} else if strings.Contains(key, ".*.") {
				return fmt.Errorf("cannot populate field %s of type map[%s]%s. Key must be of form '<section>' or '<setion>.*'.", key, kTp.String(), elemtp.String())
			} else {
				sName = key
			}
			if sName == "" {
				return fmt.Errorf("cannot populate field %s of type map[%s]%s. Key must be of form '<section>' or '<setion>.*'. <section> must be non-zero length.", key, kTp.String(), elemtp.String())
			}
		default:
			out := strings.Split(key, ".*.")
			if len(out) != 2 || out[0] == "" || out[1] == "" {
				return fmt.Errorf("cannot populate field %s of type map[%s]%s. Key must be of form '<section>.*.<key>'. Both <section> and <key> must be non-zero length.", key, kTp.String(), elemtp.String())
			}
			sName = out[0]
			sKey = out[1]
		}
		section := self.GetSection(sName, false)
		if section == nil {
			if required {
				return fmt.Errorf("cannot populate field %s of type map[%s]%s. Required section '%s' was not present.", key, kTp.String(), elemtp.String(), sName)
			}
			return nil
		}
		retval.Set(reflect.MakeMap(tp))
		for subSectName, subSection := range section.SubSections {
			kValPtr := reflect.New(kTp)
			kVal := reflect.Indirect(kValPtr)
			passConfVal := &ConfigValue{Value: []*string{&subSectName}}
			if err := self.loadSetValue(kVal, key, "", passConfVal, false, false); err != nil {
				return fmt.Errorf("cannot populate field %s of type map[%s]%s. Sub-section name '%s' could not be parsed as required key-type: %s", key, kTp.String(), elemtp.String(), subSectName, err.Error())
			}
			vValPtr := reflect.New(elemtp)
			vVal := reflect.Indirect(vValPtr)
			if amStruct {
				x := sName + "." + subSectName
				if err := self.loadStruct(vVal, x); err != nil {
					return fmt.Errorf("cannot populate field %s of type map[%s]%s. Contents of sub-section name '%s' could not be parsed as required value-type: %s", key, kTp.String(), elemtp.String(), subSectName, err.Error())
				}
			} else {
				passConfVal = subSection.GetKeyValuesRaw(sKey)
				if err := self.loadSetValue(vVal, sName+"."+subSectName+"."+sKey, defVal, passConfVal, required, haveDefault); err != nil {
					return fmt.Errorf("cannot populate field %s of type map[%s]%s. Contents of sub-section name '%s' could not be parsed as required value-type: %s", key, kTp.String(), elemtp.String(), subSectName, err.Error())
				}
			}
			retval.SetMapIndex(kVal, vVal)
		}
		return nil

	case reflect.Struct:
		if err := self.loadStruct(retval, key); err != nil {
			return fmt.Errorf("cannot populate field %s of type struct %s: %s\n", key, tp.String(), err.Error())
		}
		return nil

	default:
		return fmt.Errorf("cannot populate field %s of type %s", key, tp.String())
	}
	return nil
}

func (self *Config) loadStruct(rv reflect.Value, ns string) error {
	t := rv.Type()

	errs := LoadError{}
	for i := 0; i < t.NumField(); i++ {
		ft := t.Field(i)
		fv := rv.Field(i)

		if fv.CanSet() == false {
			continue
		}

		key := ft.Tag.Get("gcKey")
		if key == "" {
			continue
		}
		if ns != "" {
			key = ns + "." + key
		}
		req := ft.Tag.Get("gcRequired")
		required := false
		haveDefault := false
		def := ""
		if req != "" {
			var err error
			required, err = strconv.ParseBool(req)
			if err != nil {
				return fmt.Errorf("Could not parse required:\"%s\" as boolean in field %q\n", req, ft.Name)
			}
		}
		if !required {
			def, haveDefault = ft.Tag.Lookup("gcDefault")
		}
		confValue := self.GetKeyValuesRaw(key)
		if err := self.loadSetValue(fv, key, def, confValue, required, haveDefault); err != nil {
			errs[key] = fmt.Errorf("Could not populate %s field %q: %s", ft.Type.String(), ft.Name, err.Error())
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return errs
}

// Get a section by name (case insensitive) optionally creating it if not there
func (self *Config) GetSection(section string, createEmpty bool) *ConfigSection {
	slc := strings.ToLower(section)
	s := self.Sections[slc]
	if s != nil || !createEmpty {
		return s
	}
	sect := &ConfigSection{
		Name:         slc,
		OrigCaseName: section,
		SubSections:  make(map[string]*ConfigSubSection, 5),
		Values:       make(ConfigValueSet, 5),
	}
	self.Sections[slc] = sect
	return sect
}

// Get a subsection by name (main section case insensitive), optionally creating if not there
func (self *Config) GetSubSection(section, subSection string, createEmpty bool) *ConfigSubSection {
	s := self.GetSection(section, createEmpty)
	if s == nil {
		return nil
	}
	ss := s.SubSections[subSection]
	if ss != nil || !createEmpty {
		return ss
	}
	ss = &ConfigSubSection{
		Name:   subSection,
		Values: make(ConfigValueSet, 5),
	}
	s.SubSections[subSection] = ss
	return ss
}

// Attempts to get the value store for the given section/subSection
func (self *Config) GetConfigValueSet(section, subSection string, createEmpty bool) *ConfigValueSet {
	if section == "" {
		return &self.BaseValues
	}
	if subSection == "" {
		s := self.GetSection(section, createEmpty)
		if s == nil {
			return nil
		}
		return &s.Values
	}
	ss := self.GetSubSection(section, subSection, createEmpty)
	if ss == nil {
		return nil
	}
	return &ss.Values
}

// Convert a key in format section.subsection.key to relevant strings
// Will presume section.key (if only one '.')
// Or just key (if no '.')
// The last isn't valid gitconfig for acess, but gitconfig does allow storage
// and values stored with no section can be seen with e.g. `gitconfig --get-regexp .`
// This will also lowercase section and key names.
func ParseSectionKey(full_key string) (string, string, string) {
	out := strings.Split(full_key, ".")
	cnt := len(out)
	switch cnt {
	case 1:
		return "", "", strings.ToLower(out[0])
	case 2:
		return strings.ToLower(out[0]), "", strings.ToLower(out[1])
	case 3:
		return strings.ToLower(out[0]), out[1], strings.ToLower(out[2])
	default:
		return strings.ToLower(out[0]), strings.Join(out[1:cnt-1], "."), strings.ToLower(out[cnt-1])
	}
}

// Attempts to get the value store for the given section/subSection
func (self *Config) GetConfigValues(section, subSection, key string, createEmpty bool) *ConfigValue {
	valSet := self.GetConfigValueSet(section, subSection, createEmpty)
	if valSet == nil {
		return nil
	}
	return valSet.GetConfigValues(key, createEmpty)
}

func (self *Config) AddKeyValue(section, subSection, key string, value *string) {
	cvs := self.GetConfigValues(section, subSection, key, true)
	cvs.Value = append(cvs.Value, value)
}

// Getters go here, first raw
func (self *Config) GetKeyValuesRaw(key string) *ConfigValue {
	s, ss, k := ParseSectionKey(key)
	if k == "" {
		return nil
	}
	return self.GetConfigValues(s, ss, k, false)
}

// Get a set of strings of all the values as an array
// The array will be nil if the key does not exist
func (self *Config) GetKeyValuesStrings(key string) []string {
	cvs := self.GetKeyValuesRaw(key)
	if cvs == nil {
		return nil
	}
	return cvs.ValuesAsStrings()
}

// Get a set of ints of all the values as an array
// The array will be nil if the key does not exist.
// If any value is not parseable as an int, an error will be thrown.
func (self *Config) GetKeyValuesInts(key string) ([]int64, error) {
	cvs := self.GetKeyValuesRaw(key)
	if cvs == nil {
		return nil, nil
	}
	return cvs.ValuesAsInts()
}

// Get a set of bools of all the values as an array
// The array will be nil if the key does not exist.
// If any value is not parseable as a bool, an error will be thrown.
func (self *Config) GetKeyValuesBools(key string) ([]bool, error) {
	cvs := self.GetKeyValuesRaw(key)
	if cvs == nil {
		return nil, nil
	}
	return cvs.ValuesAsBools()
}

// Get the last specified value of the key as a string.
// The empty/unset value is the same as an empty string.
// If the *key* does not exist, the second return value will be false.
func (self *Config) GetKeyValueAsString(key string) (string, bool) {
	cvs := self.GetKeyValuesRaw(key)
	if cvs == nil {
		return "", false
	}
	return cvs.GetString()
}

// Get the last specified value of the key as an integer.
// The empty/unset value will cause an error.
// If the *key* does not exist, the second return value will be false.
func (self *Config) GetKeyValueAsInt(key string) (int64, bool, error) {
	cvs := self.GetKeyValuesRaw(key)
	if cvs == nil {
		return 0, false, nil
	}
	return cvs.GetInt()
}

// Get the last specified value of the key as a bool.
// The empty/unset value is the same as false.
// If the *key* does not exist, the second return value will be false.
func (self *Config) GetKeyValueAsBool(key string) (bool, bool, error) {
	cvs := self.GetKeyValuesRaw(key)
	if cvs == nil {
		return false, false, nil
	}
	return cvs.GetBool()
}

func (self *ConfigValueSet) String() string {
	out := ""
	for _, cv := range *self {
		values := cv.Value
		if len(values) == 0 {
			continue
		}
		key := cv.OrigCaseName
		for _, v := range values {
			out += "\t" + key
			if v != nil {
				escaped := EscapeValueString(*v)
				l := len(escaped)
				if l > 1 {
					// requote if trailing space or containing special chars
					last, _ := utf8.DecodeLastRuneInString(escaped)
					if unicode.IsSpace(last) || strings.ContainsAny(escaped, "#;!$`") {
						escaped = "\"" + escaped + "\""
					}
				}
				out += " = " + escaped
			}
			out += "\n"
		}
	}
	return out
}

func (self *ConfigSubSection) GetKeyValuesRaw(key string) *ConfigValue {
	return self.Values.GetConfigValues(key, false)
}

func (self *ConfigSection) String() string {
	out := self.Values.String()
	if out != "" {
		out = "[" + self.OrigCaseName + "]\n" + out
	}
	for _, ss := range self.SubSections {
		ssOut := ss.Values.String()
		if ssOut != "" {
			out += "[" + self.OrigCaseName + " \"" + EscapeValueString(ss.Name) + "\"]\n" + ssOut
		}
	}
	return out
}

func EscapeValueString(in string) string {
	quoted := strings.Replace(in, "\\", "\\\\", -1)
	quoted = strings.Replace(quoted, "\"", "\\\"", -1)
	quoted = strings.Replace(quoted, "\t", "\\\t", -1)
	quoted = strings.Replace(quoted, "\n", "\\\n", -1)
	return quoted
}

func (self *ConfigValueSet) GetConfigValues(key string, createEmpty bool) *ConfigValue {
	lcKey := strings.ToLower(key)
	vals := (*self)[lcKey]
	if vals != nil || !createEmpty {
		return vals
	}
	vals = &ConfigValue{
		Name:         lcKey,
		OrigCaseName: key,
		Value:        make([]*string, 0, 10),
	}
	(*self)[lcKey] = vals
	return vals
}

func (self *ConfigValue) CountValues() uint64 {
	return uint64(len(self.Value))
}

func (self *ConfigValue) HasValues() bool {
	cnt := len(self.Value)
	if cnt == 0 {
		return false
	}
	return true
}

func (self *ConfigValue) GetString() (string, bool) {
	out := self.ValuesAsStrings()
	l := len(out)
	if l == 0 {
		return "", false
	}
	return out[l-1], true
}

func (self *ConfigValue) GetInt() (int64, bool, error) {
	out, err := self.ValuesAsInts()
	if err != nil {
		return 0, false, err
	}
	l := len(out)
	if l == 0 {
		return 0, false, nil
	}
	return out[l-1], true, nil
}

func (self *ConfigValue) GetUint() (uint64, bool, error) {
	out, err := self.ValuesAsUints()
	if err != nil {
		return 0, false, err
	}
	l := len(out)
	if l == 0 {
		return 0, false, nil
	}
	return out[l-1], true, nil
}

func (self *ConfigValue) GetBool() (bool, bool, error) {
	out, err := self.ValuesAsBools()
	if err != nil {
		return false, false, err
	}
	l := len(out)
	if l == 0 {
		return false, false, nil
	}
	return out[l-1], true, nil
}

func (self *ConfigValue) ValuesAsStrings() []string {
	cnt := len(self.Value)
	if cnt == 0 {
		return []string{}
	}
	out := make([]string, cnt)
	for i, v := range self.Value {
		if v == nil {
			continue
		}
		out[i] = *v
	}
	return out
}

func (self *ConfigValue) ValuesAsUints() ([]uint64, error) {
	cnt := len(self.Value)
	if cnt == 0 {
		return []uint64{}, nil
	}
	out := make([]uint64, cnt)
	for i, v := range self.Value {
		if v == nil {
			return out, fmt.Errorf("Cannot convert empty value to int\n")
		}
		val, err := strconv.ParseUint(*v, 10, 64)
		if err != nil {
			return out, err
		}
		out[i] = val
	}
	return out, nil
}

func (self *ConfigValue) ValuesAsInts() ([]int64, error) {
	cnt := len(self.Value)
	if cnt == 0 {
		return []int64{}, nil
	}
	out := make([]int64, cnt)
	for i, v := range self.Value {
		if v == nil {
			return out, fmt.Errorf("Cannot convert empty value to int\n")
		}
		val, err := strconv.ParseInt(*v, 10, 64)
		if err != nil {
			return out, err
		}
		out[i] = val
	}
	return out, nil
}

// gitconfig treats all integers as true, except 0
// empty and 0-length values are false
// also recognises yes and no
func (self *ConfigValue) ValuesAsBools() ([]bool, error) {
	cnt := len(self.Value)
	if cnt == 0 {
		return []bool{}, nil
	}
	out := make([]bool, cnt)
	for i, v := range self.Value {
		if v == nil {
			out[i] = false
			continue
		}
		// check zero len
		if l := len(*v); l == 0 {
			out[i] = false
			continue
		}
		// check integer
		val, err := strconv.ParseInt(*v, 10, 32)
		if err != nil {
			if val == 0 {
				out[i] = false
			} else {
				out[i] = true
			}
			continue
		}
		lc := strings.ToLower(*v)
		switch lc {
		case "true", "yes":
			out[i] = true
		case "false", "no":
			out[i] = false
		default:
			return out, fmt.Errorf("Cannot convert '%s' to bool. Can deal with <empty>/<numeric>/true/yes/false/no\n", *v)
		}
	}
	return out, nil
}
