// Copyright 2018-2019 "Misato's Angel" <misatos.arngel@gmail.com>.
// Use of this source code is governed the MIT license.
// license that can be found in the LICENSE file.

package gitconfig

import (
	"sort"
	"strings"
	"testing"
	"time"
)

type People struct {
	Department string               `gcKey:"department.name"`
	Location   string               `gcKey:"department.location"`
	People     map[string]SubPerson `gcKey:"person.*"`
}

type SubPerson struct {
	Name       string        `gcKey:"name"`
	Email      string        `gcKey:"email" gcDefault:"someone@example.com"`
	Age        int           `gcKey:"age" gcDefault:"5"`
	ServiceLen time.Duration `gcKey:"serviceLength" gcDefault:"5m"`
	FavColour  string        `gcKey:"favouriteColour" gcRequired:"true"`
}

type Person struct {
	Name       string        `gcKey:"user.name"`
	Email      string        `gcKey:"user.email" gcDefault:"someone@example.com"`
	Age        int           `gcKey:"user.age" gcDefault:"5"`
	ServiceLen time.Duration `gcKey:"user.duration" gcDefault:"5m"`
	FavColour  string        `gcKey:"user.favouriteColour" gcRequired:"true"`
}

type TestArrays struct {
	AsFourArray    [4]string `gcKey:"arrays.key1" gcRequired:"false"`
	AsFourArrayDef [4]string `gcKey:"arrays.key1" gcDefault:"<missing>" gcRequired:"false"`
	AsTwoArray     [2]string `gcKey:"arrays.key1" gcRequired:"false"`
	AsTwoArrayDef  [2]string `gcKey:"arrays.key1" gcDefault:"<missing>" gcRequired:"false"`
	AsSixArray     [6]string `gcKey:"arrays.key1" gcRequired:"false"`
	AsSixArrayDef  [6]string `gcKey:"arrays.key1" gcDefault:"<missing>" gcRequired:"false"`
	AsSlice        []string  `gcKey:"arrays.key1" gcRequired:"false"`
	AsSliceM       []string  `gcKey:"arrays.keyMissing" gcRequired:"false"`
	AsSliceMD      []string  `gcKey:"arrays.keyMissing" gcDefault:"<missing>" gcRequired:"false"`
	IntSlice       []int     `gcKey:"arrays.key2" gcRequired:"false"`
}

type TestHashes struct {
	Key1Hash  map[string]string   `gcKey:"Hashes.*.key1" `
	Key1HashA map[string][]string `gcKey:"Hashes.*.key1" `
	Key2Hash  map[string]int      `gcKey:"Hashes.*.key2" `
	Key1HashD map[string]string   `gcKey:"Hashes.*.key1" gcDefault:"<missing>"`
	Key2HashD map[string]int      `gcKey:"Hashes.*.key2" gcDefault:"5"`
}

func TestParseSectionKey(t *testing.T) {
	testOneKey(t, "foo.x", "foo", "", "x")
	testOneKey(t, "fOo.Y", "foo", "", "y")                                                        // check case
	testOneKey(t, "someThing.Like.THAT", "something", "Like", "that")                             // check case with subsection
	testOneKey(t, "someThing.L\"ike.THAT", "something", "L\"ike", "that")                         // check subsection with quotes
	testOneKey(t, "someThing.sub.Area.here", "something", "sub.Area", "here")                     // check subsection with quotes
	testOneKey(t, "someThing.sub.Area.With.Many.here", "something", "sub.Area.With.Many", "here") // check subsection with quotes
}

func testOneKey(t *testing.T, key, sec, subSec, final string) {
	s, ss, f := ParseSectionKey(key)
	if s != sec {
		t.Errorf("Key: '%s' expected section: '%s' but got '%s'\n", key, sec, s)
	}
	if ss != subSec {
		t.Errorf("Key: '%s' expected sub-section: '%s' but got '%s'\n", key, subSec, ss)
	}
	if f != final {
		t.Errorf("Full Key: '%s' expected final key: '%s' but got '%s'\n", key, final, f)
	}
}

func TestParseConfig(t *testing.T) {
	s := "# comment line\n" +
		"[foo]\n" +
		"# commented line\n" +
		"; commented line\n" +
		"    c = word ; some comment\n" + // check basic value
		"    cc = word # some comment\n" + // check basic value
		"    ccc = word;some comment\n" + // check basic value
		"    cccc = word#some comment\n" + // check basic value
		"    q = \"word#some comment\"\n" + // check quoted comment
		"    qq = \"word;some comment\"\n" + // check quoted comment
		"    qqq = \"word # some comment\"\n" + // check quoted comment
		"    qqqq = \"word ; some comment\"\n" + // check quoted comment
		"    x = y\n" + // check basic value
		"    A = B C\n" + // check internal space preserved
		"    B =   zz  zz  \n" + // check trailing space ignored
		"    runOver = B\\\n" + // check newline wrap allowed
		" C\n" +
		"[some] key = value\n" +
		"[something \"Somewhere\"]\n" +
		"    some-key = some-value\n" +
		"    another-key = another-value\n" +
		"[something \"Some\\\"Quote.and random\"]\n" +
		"    a = b\n" +
		"[arrays]\n" +
		"    key1 = a\n" +
		"    key1 = b\n" +
		"    key1 = c\n"

	config, err := NewConfigFromString(s)
	if err != nil {
		t.Errorf("Failed to parse config:\n===\n%s\n===\n%s", s, err.Error())
		return
	}
	testValue(t, config, "foo.c", "word", true)
	testValue(t, config, "foo.cc", "word", true)
	testValue(t, config, "foo.ccc", "word", true)
	testValue(t, config, "foo.cccc", "word", true)
	testValue(t, config, "foo.q", "word#some comment", true)
	testValue(t, config, "foo.qq", "word;some comment", true)
	testValue(t, config, "foo.qqq", "word # some comment", true)
	testValue(t, config, "foo.qqqq", "word ; some comment", true)
	testValue(t, config, "FOO.X", "y", true)
	testValue(t, config, "FOO.A", "B C", true)
	testValue(t, config, "FOO.runOver", "B C", true)
	testValue(t, config, "foo.B", "zz  zz", true)
	testValue(t, config, "some.key", "value", true)
	testValue(t, config, "something.Somewhere.some-key", "some-value", true)
	testValue(t, config, "something.somewhere.some-key", "", false)
	testValue(t, config, "something.Some\"Quote.and random.a", "b", true)
	testValue(t, config, "arrays.key1", "c", true)

}

func TestLoadStructs(t *testing.T) {
	configStr := "[department]\n" +
		"    name = Somewhere\n" +
		"    location = England\n" +
		"[person \"Joe\"]\n" +
		"    name = Joe Bloggs\n" +
		"    age = 23\n" +
		"    email = Joe.Bloggs@company.com\n" +
		"    serviceLength = 24h\n" +
		"    favouriteColour = blue\n" +
		"[person \"Joanne\"]\n" +
		"    name = Joanne Bloggs\n" +
		"    age = 22\n" +
		"    email = Joanne.Bloggs@company.com\n" +
		"    serviceLength = 1024h\n" +
		"    favouriteColour = green\n"
	config, err := NewConfigFromString(configStr)
	if err != nil {
		t.Errorf("Failed to parse config:\n===\n%s\n===\n%s", configStr, err.Error())
		return
	}

	var p People
	err = config.Load(&p)
	if err != nil {
		t.Errorf("Failed to test hashes from:\n===\n%s\n===\n%s", configStr, err.Error())
		return
	}

	if p.Department != "Somewhere" {
		t.Errorf("Expect Department Name 'Somewhere' but got '%s'\n", p.Department)
	}
	if p.Location != "England" {
		t.Errorf("Expect Location 'England' but got '%s'\n", p.Location)
	}
	keys := make([]string, 0, 5)
	for name, _ := range p.People {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	if len(keys) != 2 || keys[0] != "Joanne" || keys[1] != "Joe" {
		t.Errorf("Expected two people with names 'Joanne' and 'Joe' but got %d people with names: %s\n", len(keys), strings.Join(keys, ", "))
	}
	if p.People["Joe"].Name != "Joe Bloggs" {
		t.Errorf("Expected Joe's name to be 'Joe Bloggs' but got: '%s'\n", p.People["Joe"].Name)
	}
	if p.People["Joe"].Email != "Joe.Bloggs@company.com" {
		t.Errorf("Expected Joe's name to be 'Joe.Bloggs@company.com' but got: '%s'\n", p.People["Joe"].Email)
	}
	if p.People["Joe"].Age != 23 {
		t.Errorf("Expected Joe's age to be '23' but got: '%d'\n", p.People["Joe"].Age)
	}
	if p.People["Joe"].ServiceLen != (24 * time.Hour) {
		t.Errorf("Expected Joe's name to be '%d' but got: '%d'\n", 24*time.Hour, p.People["Joe"].ServiceLen)
	}
	if p.People["Joe"].FavColour != "blue" {
		t.Errorf("Expected Joe's favourite colour to be 'blue' but got: '%s'\n", p.People["Joe"].FavColour)
	}

	if p.People["Joanne"].Name != "Joanne Bloggs" {
		t.Errorf("Expected Joanne's name to be 'Joanne Bloggs' but got: '%s'\n", p.People["Joanne"].Name)
	}
	if p.People["Joanne"].Email != "Joanne.Bloggs@company.com" {
		t.Errorf("Expected Joanne's name to be 'Joanne.Bloggs@company.com' but got: '%s'\n", p.People["Joanne"].Email)
	}
	if p.People["Joanne"].Age != 22 {
		t.Errorf("Expected Joanne's age to be '22' but got: '%d'\n", p.People["Joanne"].Age)
	}
	if p.People["Joanne"].ServiceLen != (1024 * time.Hour) {
		t.Errorf("Expected Joanne's name to be '%d' but got: '%d'\n", 1024*time.Hour, p.People["Joanne"].ServiceLen)
	}
	if p.People["Joanne"].FavColour != "green" {
		t.Errorf("Expected Joanne's favourite colour to be 'green' but got: '%s'\n", p.People["Joanne"].FavColour)
	}
}

func TestLoadHashMap(t *testing.T) {
	configStr := "[Hashes \"one\"]\n" +
		"    key1 = a\n" +
		"    key2 = 1\n" +
		"[Hashes \"two\"]\n" +
		"    key1 = ignored\n" +
		"    key1 = b\n" +
		"    key2 = 2\n" +
		"[Hashes \"three\"]\n" +
		"    key1 = c\n" +
		"[Hashes \"four\"]\n" +
		"    key2 = 3\n"
	config, err := NewConfigFromString(configStr)
	if err != nil {
		t.Errorf("Failed to parse config:\n===\n%s\n===\n%s", configStr, err.Error())
		return
	}

	var th TestHashes
	err = config.Load(&th)
	if err != nil {
		t.Errorf("Failed to test hashes from:\n===\n%s\n===\n%s", configStr, err.Error())
		return
	}
	checkStringHashVal(t, th.Key1Hash, "Key1Hash (Hashes.*.key1)", "zero", "", false)
	checkStringHashVal(t, th.Key1Hash, "Key1Hash (Hashes.*.key1)", "one", "a", true)
	checkStringHashVal(t, th.Key1Hash, "Key1Hash (Hashes.*.key1)", "two", "b", true)
	checkStringHashVal(t, th.Key1Hash, "Key1Hash (Hashes.*.key1)", "three", "c", true)
	checkStringHashVal(t, th.Key1Hash, "Key1Hash (Hashes.*.key1)", "four", "", true)

	checkStringHashVal(t, th.Key1HashD, "Key1HashD (Hashes.*.key1)", "zero", "", false)
	checkStringHashVal(t, th.Key1HashD, "Key1HashD (Hashes.*.key1)", "one", "a", true)
	checkStringHashVal(t, th.Key1HashD, "Key1HashD (Hashes.*.key1)", "two", "b", true)
	checkStringHashVal(t, th.Key1HashD, "Key1HashD (Hashes.*.key1)", "three", "c", true)
	checkStringHashVal(t, th.Key1HashD, "Key1HashD (Hashes.*.key1)", "four", "<missing>", true)

	checkIntHashVal(t, th.Key2Hash, "Key2Hash (Hashes.*.key2)", "zero", 0, false)
	checkIntHashVal(t, th.Key2Hash, "Key2Hash (Hashes.*.key2)", "one", 1, true)
	checkIntHashVal(t, th.Key2Hash, "Key2Hash (Hashes.*.key2)", "two", 2, true)
	checkIntHashVal(t, th.Key2Hash, "Key2Hash (Hashes.*.key2)", "three", 0, true)
	checkIntHashVal(t, th.Key2Hash, "Key2Hash (Hashes.*.key2)", "four", 3, true)

	checkIntHashVal(t, th.Key2HashD, "Key2HashD (Hashes.*.key2)", "zero", 0, false)
	checkIntHashVal(t, th.Key2HashD, "Key2HashD (Hashes.*.key2)", "one", 1, true)
	checkIntHashVal(t, th.Key2HashD, "Key2HashD (Hashes.*.key2)", "two", 2, true)
	checkIntHashVal(t, th.Key2HashD, "Key2HashD (Hashes.*.key2)", "three", 5, true)
	checkIntHashVal(t, th.Key2HashD, "Key2HashD (Hashes.*.key2)", "four", 3, true)

	checkStringAHashVal(t, th.Key1HashA, "Key1HashA (Hashes.*.key1)", "zero", nil, false)
	checkStringAHashVal(t, th.Key1HashA, "Key1HashA (Hashes.*.key1)", "one", []string{"a"}, true)
	checkStringAHashVal(t, th.Key1HashA, "Key1HashA (Hashes.*.key1)", "two", []string{"ignored", "b"}, true)
	checkStringAHashVal(t, th.Key1HashA, "Key1HashA (Hashes.*.key1)", "three", []string{"c"}, true)
	checkStringAHashVal(t, th.Key1HashA, "Key1HashA (Hashes.*.key1)", "four", []string{}, true)

}

func checkStringAHashVal(t *testing.T, th map[string][]string, name, k string, v []string, shouldExist bool) {
	val, exists := th[k]
	if exists != shouldExist {
		t.Errorf("Failed %s existence mismatch for key '%s', got %t expected %t\n", name, k, exists, shouldExist)
		return
	}
	if !exists {
		return
	}
	valLen := len(val)
	vLen := len(v)
	valJoin := strings.Join(val, ", ")
	vJoin := strings.Join(v, ", ")
	if valLen != vLen {
		t.Errorf("Failed %s key '%s' should have been %d in length (value: '%s') but got %d (value: '%s')\n", name, k, vLen, vJoin, valLen, valJoin)
	}
	if vJoin != valJoin {
		t.Errorf("Failed %s key '%s' should have been '%s' but got '%s'\n", name, k, vJoin, valJoin)
	}
}

func checkStringHashVal(t *testing.T, th map[string]string, name, k, v string, shouldExist bool) {
	val, exists := th[k]
	if exists != shouldExist {
		t.Errorf("Failed %s existence mismatch for key '%s', got %t expected %t\n", name, k, exists, shouldExist)
		return
	}
	if !exists {
		return
	}
	if val != v {
		t.Errorf("Failed %s key '%s' should have been '%s' but got '%s'\n", name, k, v, val)
	}
}

func checkIntHashVal(t *testing.T, th map[string]int, name, k string, v int, shouldExist bool) {
	val, exists := th[k]
	if exists != shouldExist {
		t.Errorf("Failed %s existence mismatch for key '%s', got %t expected %t\n", name, k, exists, shouldExist)
		return
	}
	if !exists {
		return
	}
	if val != v {
		t.Errorf("Failed %s key '%s' should have been '%d' but got '%d'\n", name, k, v, val)
	}
}

func TestLoadArray(t *testing.T) {
	configStr := "[arrays]\n" +
		"    key1 = a\n" +
		"    key1 = b\n" +
		"    key1 = c\n" +
		"    key1 = d\n" +
		"    key2 = 1\n" +
		"    key2 = 2\n" +
		"    key2 = 3\n"
	config, err := NewConfigFromString(configStr)
	if err != nil {
		t.Errorf("Failed to parse config:\n===\n%s\n===\n%s", configStr, err.Error())
		return
	}
	var ta TestArrays
	err = config.Load(&ta)
	if err != nil {
		t.Errorf("Failed to test arrays from:\n===\n%s\n===\n%s", configStr, err.Error())
		return
	}
	intSliLen := len(ta.IntSlice)
	if intSliLen != 3 {
		t.Errorf("arrays.key2 (intslice) should have length 3 but got %d\n", intSliLen)
	}
	for i, val := range ta.IntSlice {
		if val != i+1 {
			t.Errorf("arrays.key2 (intslice) index %d should have been %d but got %d\n", i, i+1, val)
		}
	}
	expected := []string{"a", "b", "c", "d"}
	for i := 0; i < 4; i++ {
		if i < 2 {
			// short array should get the last 2 values
			if ta.AsTwoArray[i] != expected[i+2] {
				t.Errorf("arrays.key1 (AsTwoArray) index %d should have been '%s' but got '%s'\n", i, expected[i+2], ta.AsTwoArray[i])
			}
			if ta.AsTwoArrayDef[i] != expected[i+2] {
				t.Errorf("arrays.key1 (AsTwoArrayDef) index %d should have been '%s' but got '%s'\n", i, expected[i+2], ta.AsTwoArrayDef[i])
			}
		}
		if ta.AsFourArray[i] != expected[i] {
			t.Errorf("arrays.key1 (AsFourArray) index %d should have been '%s' but got '%s'\n", i, expected[i], ta.AsFourArray[i])
		}
		if ta.AsFourArrayDef[i] != expected[i] {
			t.Errorf("arrays.key1 (AsFourArrayDef) index %d should have been '%s' but got '%s'\n", i, expected[i], ta.AsFourArrayDef[i])
		}
		if ta.AsSixArray[i] != expected[i] {
			t.Errorf("arrays.key1 (AsSixArray) index %d should have been '%s' but got '%s'\n", i, expected[i], ta.AsSixArray[i])
		}
		if ta.AsSixArrayDef[i] != expected[i] {
			t.Errorf("arrays.key1 (AsSixArrayDef) index %d should have been '%s' but got '%s'\n", i, expected[i], ta.AsSixArrayDef[i])
		}
	}
	for i := 4; i < 6; i++ {
		if ta.AsSixArray[i] != "" {
			t.Errorf("arrays.key1 (AsSixArray) index %d should have been '%s' but got '%s'\n", i, "", ta.AsSixArray[i])
		}
		if ta.AsSixArrayDef[i] != "<missing>" {
			t.Errorf("arrays.key1 (AsSixArrayDef) index %d should have been '%s' but got '%s'\n", i, "<missing>", ta.AsSixArrayDef[i])
		}
	}

	asLen := len(ta.AsSliceM)
	if asLen != 0 {
		t.Errorf("arrays.keyMissing should produce empty slice, but got one of len: %d\n", asLen)
	}
	asLen = len(ta.AsSliceMD)
	if asLen != 1 {
		t.Errorf("arrays.keyMissing with default should produce single entry slice, but got one of len: %d\n", asLen)
	} else {
		if ta.AsSliceMD[0] != "<missing>" {
			t.Errorf("arrays.keyMissing with default first entry should be <missing> but got %s\n", ta.AsSliceMD[0])
		}
	}
}

func TestLoad(t *testing.T) {
	configStr := "[user]\n" +
		"    name = Joe Bloggs\n" +
		"    favouriteColour = Blue\n"
	config, err := NewConfigFromString(configStr)
	if err != nil {
		t.Errorf("Failed to parse config:\n===\n%s\n===\n%s", configStr, err.Error())
		return
	}
	var p Person
	err = config.Load(&p)
	if err != nil {
		t.Errorf("Failed to load from:\n===\n%s\n===\n%s", configStr, err.Error())
		return
	}
	if p.Name != "Joe Bloggs" {
		t.Errorf("Expect Name 'Joe Bloggs' but got '%s'\n", p.Name)
	}
	if p.Email != "someone@example.com" {
		t.Errorf("Expect Email 'someone@example.com' but got '%s'\n", p.Email)
	}
	if p.Age != 5 {
		t.Errorf("Expect Age '5' but got '%d'\n", p.Age)
	}
	if p.FavColour != "Blue" {
		t.Errorf("Expect FavColour 'Blue' but got '%s'\n", p.FavColour)
	}

	configStr = "[user]\n" +
		"    name = Joe Bloggs\n"
	config, err = NewConfigFromString(configStr)
	if err != nil {
		t.Errorf("Failed to parse config:\n===\n%s\n===\n%s", configStr, err.Error())
		return
	}
	err = config.Load(&p)
	if err == nil {
		t.Errorf("Expect error on missing favourite colour, but no error given\nConfig:\n%s", config.String())
	} else {
		loadErr, ok := err.(LoadError)
		if !ok {
			t.Errorf("Expect gitconfig.LoadError on return, but got %T\n", err)
			return
		}
		lErr := loadErr["user.favouriteColour"]
		if lErr == nil {
			t.Errorf("Expect 'user.favouriteColour' error on missing favourite colour, but other errors given: %s\n", err.Error())
		}
		if !strings.Contains(lErr.Error(), "Could not populate required") {
			t.Errorf("Unexpected error on missing favourite colour, expected complaint about required, but got: '%s'\n", lErr.Error())
		}
	}
}

func testValue(t *testing.T, config *Config, key, value string, exists bool) {
	got, existed := config.GetKeyValueAsString(key)
	if existed != exists {
		if existed {
			t.Errorf("Expect %s to not exist, but got '%s'", key, got)
		} else {
			t.Errorf("Expect %s = '%s', but did not exist, struct is:\n%s", key, value, config.String())
		}
		return
	}
	if got != value {
		t.Errorf("Expect %s = '%s', but got '%s'", key, value, got)
	}

}
