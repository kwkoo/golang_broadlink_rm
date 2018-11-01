package argparser

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
)

func ExampleParse() {
	type Config struct {
		Hostname string `env:"HOST" flag:"host" default:"localhost"`
		Port     int    `default:"8080"`
		Async    bool
	}

	c := Config{}
	err := Parse(&c)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Hostname: %v\n", c.Hostname)
}

func TestParse(t *testing.T) {
	type Config struct {
		Hostname string `env:"HOST" flag:"host" default:"localhost"`
		Port     int    `default:"8080"`
		Async    bool
	}

	tables := []struct {
		flags    []string
		env      []string
		expected Config
		isErr    bool
		stderr   bool
	}{
		{[]string{"-host", "abc", "-port", "8000", "-async"}, []string{"", "", ""}, Config{"abc", 8000, true}, false, false},        // flags set, env not set
		{[]string{}, []string{"def", "7000", "true"}, Config{"def", 7000, true}, false, false},                                      // flags not set, env set
		{[]string{"-host", "ghi"}, []string{"", "6000", ""}, Config{"ghi", 6000, false}, false, false},                              // some flags set, some env set
		{[]string{"-host", "abc", "-port", "8000", "-async"}, []string{"def", "7000", ""}, Config{"def", 7000, true}, false, false}, // both flags and env set, env should override flags
		{[]string{"-host", "abc", "-port", "text", "-async"}, []string{"", "", ""}, Config{}, false, true},                          // integer command line flag parsing error
		{[]string{}, []string{"def", "text", "true"}, Config{"def", 7000, true}, true, false},                                       // environment variable for int field set to non-integer
	}

	for index, table := range tables {
		t.Logf("Testing table %d", index)
		setFlags(table.flags)
		setEnv(table.env)

		// Needed because we are calling flag.Parse() each time we run a test.
		flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
		stderr := new(bytes.Buffer)
		flag.CommandLine.SetOutput(stderr)

		result := Config{}
		err := Parse(&result)

		if table.isErr {
			if err == nil {
				t.Error("Expected an error but did not get it")
			}
			continue
		}

		if stderr.Len() > 0 {
			if !table.stderr {
				t.Errorf("Test produced unexpected output to stderr: %v", stderr.String())
			}
			continue
		} else {
			if table.stderr {
				t.Error("Test was expected to output to stderr but it did not")
			}
		}
		if result.Hostname != table.expected.Hostname {
			t.Errorf("Expected hostname %v but got %v instead", table.expected.Hostname, result.Hostname)
		}
		if result.Port != table.expected.Port {
			t.Errorf("Expected port %v but got %v instead", table.expected.Port, result.Port)
		}
		if result.Async != table.expected.Async {
			t.Errorf("Expected async %v but got %v instead", table.expected.Async, result.Async)
		}
	}
}

func setFlags(args []string) {
	myargs := []string{"test"}
	myargs = append(myargs, args...)
	os.Args = myargs
}

func setEnv(values []string) {
	hostname := values[0]
	port := values[1]
	async := values[2]
	if len(hostname) == 0 {
		os.Unsetenv("HOST")
	} else {
		os.Setenv("HOST", hostname)
	}
	if len(port) == 0 {
		os.Unsetenv("PORT")
	} else {
		os.Setenv("PORT", port)
	}
	if len(async) == 0 {
		os.Unsetenv("ASYNC")
	} else {
		os.Setenv("ASYNC", async)
	}
}
