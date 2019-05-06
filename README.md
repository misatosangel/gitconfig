# gitconfig
Native golang implementation of gitconfig read/write under MIT license.

This pckage is very rough and ready just good enough to allow parsing
of `gitconfig` style files without needing git on the system or using a
heavyweight entire git implementation.

It doesn't deal with includes, but handles just about anything else. It
provides golang style tag annotations for reading config and routines
to just raw parse configuration files.

Example simple section struct:
------------------------------
```go
type Person struct {
	// Read a "user" section and assign values to this struct
	Name       string        `gcKey:"user.name"`
	Email      string        `gcKey:"user.email" gcDefault:"someone@example.com"`
	Age        int           `gcKey:"user.age" gcDefault:"5"`
	ServiceLen time.Duration `gcKey:"user.duration" gcDefault:"5m"`
	FavColour  string        `gcKey:"user.favouriteColour" gcRequired:"true"`
}
```

Example nesting:
----------------
```go
type Person struct {
	// read raw keys from the base level of the gitconfig file
	// or invoked subsection
	Name       string        `gcKey:"name"`
	Email      string        `gcKey:"email" gcDefault:"someone@example.com"`
	Age        int           `gcKey:"age" gcDefault:"5"`
	ServiceLen time.Duration `gcKey:"serviceLength" gcDefault:"5m"`
	FavColour  string        `gcKey:"favouriteColour" gcRequired:"true"`
}

type People struct {
	// read a department section for name and location
	Department string               `gcKey:"department.name"`
	Location   string               `gcKey:"department.location"`

	// read all "person" subsections" as Person objects 
	People     map[string]Person    `gcKey:"person.*"`
}
```

The library supports times and durations, as well as parsing to (unsigned)
integers and booleans.

Individual keys from multiple subsections can be pulled out to raw arrays
during parses follows, each key is filled in from subsections of `Hashes`:

```go
type ExampleHashes struct {
	// Creates a hash keyed on subsection name, value the (first) value of key1
	// in file order.
	Key1Hash  map[string]string   `gcKey:"Hashes.*.key1" `

	// Creates a hash keyed on subsection name, slice of string value of key1s
	// in file order
	Key1HashA map[string][]string `gcKey:"Hashes.*.key1" `

	// Creates a hash keyed on subsection name, value the (first) value of key2
	// in file order coerced as a signed integer.
	Key2Hash  map[string]int      `gcKey:"Hashes.*.key2" `
	
	// as above but with defaults filled in if the jeys are not present
	Key1HashD map[string]string   `gcKey:"Hashes.*.key1" gcDefault:"<missing>"`
	Key2HashD map[string]int      `gcKey:"Hashes.*.key2" gcDefault:"5"`
}
```

