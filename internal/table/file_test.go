package maps

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/foxcpp/maddy/internal/config"
	"github.com/foxcpp/maddy/internal/testutils"
)

func TestReadFile(t *testing.T) {
	test := func(file string, expected map[string]string) {
		t.Helper()

		f, err := ioutil.TempFile("", "maddy-tests-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())
		defer f.Close()
		if _, err := f.WriteString(file); err != nil {
			t.Fatal(err)
		}

		actual := map[string]string{}
		err = readFile(f.Name(), actual)
		if expected == nil {
			if err == nil {
				t.Errorf("expected failure, got %+v", actual)
			}
			return
		}
		if err != nil {
			t.Errorf("unexpected failure: %v", err)
			return
		}

		if !reflect.DeepEqual(actual, expected) {
			t.Errorf("wrong results\n want %+v\n got %+v", expected, actual)
		}
	}

	test("a: b", map[string]string{"a": "b"})
	test("a@example.org: b@example.com", map[string]string{"a@example.org": "b@example.com"})
	test(`"a @ a"@example.org: b@example.com`, map[string]string{`"a @ a"@example.org`: "b@example.com"})
	test(`a@example.org: "b @ b"@example.com`, map[string]string{`a@example.org`: `"b @ b"@example.com`})
	test(`"a @ a": "b @ b"`, map[string]string{`"a @ a"`: `"b @ b"`})
	test("a: b, c", map[string]string{"a": "b, c"})
	test(": b", nil)
	test(":", nil)
	test("aaa", map[string]string{"aaa": ""})
	test(": b", nil)
	test("     testing@example.com   :  arbitrary-whitespace@example.org   ",
		map[string]string{"testing@example.com": "arbitrary-whitespace@example.org"})
	test(`# skip comments
a: b`, map[string]string{"a": "b"})
	test(`# and empty lines

a: b`, map[string]string{"a": "b"})
	test("# with whitespace too\n    \na: b", map[string]string{"a": "b"})
}

func TestFileReload(t *testing.T) {
	t.Parallel()

	const file = `cat: dog`

	f, err := ioutil.TempFile("", "maddy-tests-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(file); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	mod, err := NewFile("", "", nil, []string{f.Name()})
	if err != nil {
		t.Fatal(err)
	}
	m := mod.(*File)
	m.log = testutils.Logger(t, "file_map")
	defer m.Close()

	if err := mod.Init(&config.Map{Block: &config.Node{}}); err != nil {
		t.Fatal(err)
	}

	// This delay is somehow important. Not sure why.
	time.Sleep(250 * time.Millisecond)

	if err := ioutil.WriteFile(f.Name(), []byte("dog: cat"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 10; i++ {
		time.Sleep(reloadInterval + 50*time.Millisecond)
		m.mLck.RLock()
		if m.m["dog"] != "" {
			m.mLck.RUnlock()
			break
		}
		m.mLck.RUnlock()
	}

	m.mLck.RLock()
	defer m.mLck.RUnlock()
	if m.m["dog"] == "" {
		t.Fatal("New m were not loaded")
	}
}

func TestFileReload_Broken(t *testing.T) {
	t.Parallel()

	const file = `cat: dog`

	f, err := ioutil.TempFile("", "maddy-tests-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	if _, err := f.WriteString(file); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	mod, err := NewFile("", "", nil, []string{f.Name()})
	if err != nil {
		t.Fatal(err)
	}
	m := mod.(*File)
	m.log = testutils.Logger(t, FileModName)
	defer m.Close()

	if err := mod.Init(&config.Map{Block: &config.Node{}}); err != nil {
		t.Fatal(err)
	}

	// This delay is somehow important. Not sure why.
	time.Sleep(250 * time.Millisecond)

	if err := ioutil.WriteFile(f.Name(), []byte(":"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	time.Sleep(3 * reloadInterval)

	m.mLck.RLock()
	defer m.mLck.RUnlock()
	if m.m["cat"] == "" {
		t.Fatal("New m were loaded or map changed", m.m)
	}
}

func TestFileReload_Removed(t *testing.T) {
	t.Parallel()

	const file = `cat: dog`

	f, err := ioutil.TempFile("", "maddy-tests-")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(file); err != nil {
		f.Close()
		t.Fatal(err)
	}
	f.Close()

	mod, err := NewFile("", "", nil, []string{f.Name()})
	if err != nil {
		t.Fatal(err)
	}
	m := mod.(*File)
	m.log = testutils.Logger(t, FileModName)
	defer m.Close()

	if err := mod.Init(&config.Map{Block: &config.Node{}}); err != nil {
		t.Fatal(err)
	}

	// This delay is somehow important. Not sure why.
	time.Sleep(250 * time.Millisecond)

	os.Remove(f.Name())

	time.Sleep(3 * reloadInterval)

	m.mLck.RLock()
	defer m.mLck.RUnlock()
	if m.m["cat"] != "" {
		t.Fatal("Old m are still loaded")
	}
}

func init() {
	reloadInterval = 250 * time.Millisecond
}
