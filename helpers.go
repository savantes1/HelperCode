package helpers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

// container is used to keep track of redirected stdout and stderr
// and hold output collected. Once the Stop() method is called,
// stdout and stderr are restored and collected output is available
// via the Data field.
// IMPORTANT: container is not reusable for collecting output after
// Stop() method is called. If you need to collect output after Stop()
// create new container via NewContainer() function.
type container struct {
	delimiters []rune

	backupStdout *os.File
	writerStdout *os.File
	backupStderr *os.File
	writerStderr *os.File
	backupStdin  *os.File
	readerStdin  *os.File
	writerStdin  io.Writer

	outData      string
	errorData    string
	outChannel   chan string
	errorChannel chan string

	OutData   []string
	ErrorData []string
}

func NewContainer(delims ...rune) (*container, error) {

	rStdout, wStdout, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	rStderr, wStderr, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	rStdin, wStdin, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	c := &container{
		delimiters: delims,

		backupStdout: os.Stdout,
		writerStdout: wStdout,

		backupStderr: os.Stderr,
		writerStderr: wStderr,

		backupStdin: os.Stdin,
		readerStdin: rStdin,
		writerStdin: wStdin,

		outChannel:   make(chan string),
		errorChannel: make(chan string),
	}

	os.Stdin = c.readerStdin
	os.Stdout = c.writerStdout
	os.Stderr = c.writerStderr

	go func(outChan chan string, errorChan chan string, readerStdout *os.File, readerStderr *os.File) {
		var bufStdout bytes.Buffer

		// try to copy buffer from stdout to out channel
		// if it fails, print message to the stderr (not great solution, but can't think of better one)
		if _, err := io.Copy(&bufStdout, readerStdout); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		if bufStdout.Len() > 0 {
			outChan <- bufStdout.String()
		}

		var bufStderr bytes.Buffer

		// try to copy buffer from stderr to out channel
		// ironically, if it fails, print message to stderr...
		if _, err := io.Copy(&bufStderr, readerStderr); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		if bufStderr.Len() > 0 {
			errorChan <- bufStderr.String()
		}
	}(c.outChannel, c.errorChannel, rStdout, rStderr)

	go func(c *container) {
		for {
			select {
			case out := <-c.outChannel:
				c.outData += out
			case err := <-c.errorChannel:
				c.errorData += err
			}
		}
	}(c)

	return c, nil
}

// Write string to redirected stdin
// IMPORTANT: There is a known limitation with redirecting stdin
// when trying to feed string data to the fmt.Scanln/fmt.Scan functions
// after already feeding data to bufio.Scanner.Scan.
// In this scenario, the bufio.Scanner.Scan will read THE ENTIRE
// string and leave nothing left for the fmt.Scan function, thereby
// leaving it waiting indefinitely for input. :( 
func (c *container) WriteToStdin(input string) error {

	_, err := fmt.Fprint(c.writerStdin, input)
	if err != nil {
		return err
	}

	return nil
}



// Stop() closes redirected stdout and stderr and restores them.
// Also formats collected output data in container.
func (c *container) Stop() {

	if c.writerStdout != nil {
		_ = c.writerStdout.Close()
	}

	if c.writerStderr != nil {
		_ = c.writerStderr.Close()
	}

	if c.readerStdin != nil {
		_ = c.readerStdin.Close()
	}

	// Give it a sec to finish collecting data from buffers?
	time.Sleep(10 * time.Millisecond)

	os.Stdout = c.backupStdout
	os.Stderr = c.backupStderr
	os.Stdin = c.backupStdin

	// Separate captured stdout by delimeters
	c.OutData = strings.FieldsFunc(c.outData,
		func(r rune) bool {

			for _, elem := range c.delimiters {
				if r == elem {
					return true
				}
			}

			return false
		},
	)

	// // Remove empty items
	// for _, elem := range temp {
	// 	if elem != "" {
	// 		c.Data = append(c.Data, elem)
	// 	}
	// }

	c.ErrorData = strings.Split(c.errorData, "\n")

	if c.ErrorData[len(c.ErrorData)-1] == "" {
		c.ErrorData = c.ErrorData[:len(c.ErrorData)-1]
	}

}






// Function anatomy testing struct
type FuncAnatomyTest struct {
	Name		string
	Obj			interface{}
	ArgTypes	[]reflect.Type
	ReturnTypes []reflect.Type
}

// Runs standard function anatomy tests
func RunFunctionAnatomyTests(testFuncs []FuncAnatomyTest, t *testing.T) {

	for i := 0; i < len(testFuncs); i++ {

		function := reflect.ValueOf(testFuncs[i].Obj)

		if function.IsValid() {

			if function.Type().NumIn() == len(testFuncs[i].ArgTypes) {

				for j := 0; j < len(testFuncs[i].ArgTypes); j++ {

					param := function.Type().In(j)

					if param != testFuncs[i].ArgTypes[j] {
						t.Error("Function '" + testFuncs[i].Name + "' has unexpected parameter type at position " +
							strconv.Itoa(j) + ". Expected type " + testFuncs[i].ArgTypes[j].String() +
							", found type " + param.String())
					}

				}

			} else {
				t.Error("Function '" + testFuncs[i].Name +
					"' has unexpected number of parameters. Expected " + strconv.Itoa(len(testFuncs[i].ArgTypes)) +
					" parameter(s), found " + strconv.Itoa(function.Type().NumIn()) + " parameter(s)")
			}

			if function.Type().NumOut() == len(testFuncs[i].ReturnTypes) {

				for j := 0; j < len(testFuncs[i].ReturnTypes); j++ {

					returnParam := function.Type().Out(j)

					if returnParam != testFuncs[i].ReturnTypes[j] {

						t.Error("Function '" + testFuncs[i].Name +
							"' returned unexpected data type for return " + strconv.Itoa(j) + ". Expected type " +
							testFuncs[i].ReturnTypes[j].String() + ", received type " + returnParam.String())
					}
				}

			} else {

				t.Error("Function '" + testFuncs[i].Name +
					"' returns unexpected number of values. Expected " + strconv.Itoa(len(testFuncs[i].ReturnTypes)) +
					" value(s), found " + strconv.Itoa(function.Type().NumOut())  + " value(s)")
			}

		} else {
			t.Error("'" + testFuncs[i].Name + "' function definition missing.")
		}

	}
}









// Function output testing struct
type FuncOutputTest struct {
	Name          string
	Obj           interface{}
	Args          []reflect.Value
	StdinStrings  []string
	IgnoreStdout  bool
	StdoutStrings []string
	IgnoreReturns bool
	Returns       []reflect.Value
}

// Converts FuncOutputTest object to FuccAnatomyTest object
func convertFuncOutputTestToAnatomyTest(ot FuncOutputTest) FuncAnatomyTest {

	var argTypes []reflect.Type
	var returnTypes []reflect.Type

	for _, el := range ot.Args {
		argTypes = append(argTypes, el.Type())
	}

	for _, el := range ot.Returns {
		returnTypes = append(returnTypes, el.Type())
	}

	return FuncAnatomyTest {

		Name: ot.Name,
		Obj: ot.Obj,
		ArgTypes: argTypes,
		ReturnTypes: returnTypes,
	}
}



// Runs standard function output tests using provided values
func RunFunctionOutputTests(testFuncs []FuncOutputTest, t *testing.T) {

	for i := 0; i < len(testFuncs); i++ {

		// Run the anatomy test on the function first
		RunFunctionAnatomyTests([]FuncAnatomyTest{ convertFuncOutputTestToAnatomyTest(testFuncs[i])}, t)

		// Don't even bother running the actual output tests if the anatomy tests failed
		if !t.Failed() {


			function := reflect.ValueOf(testFuncs[i].Obj)

			if function.IsValid() {

				//TODO: handle errors, maybe?
				c, _ := NewContainer('\n')

				// Create context for goroutine with timeout in case the function
				// tries to process more stdin input than what is expected. Running
				// in goroutine with context allows for easily timing out after 3 seconds.
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

				var returnVals []reflect.Value

				go func() {

					// In case this function uses random numbers, make sure to set
					// the seed to 1 so the "random" numbers will be predictable
					// (deterministic) and will match the expected output
					rand.Seed(1)

					// Call the function...
					// Note: This logic does not support variadic functions!
					returnVals = function.Call(testFuncs[i].Args)
					cancel()

				}()

				// Write each input string to function
				for _, s := range testFuncs[i].StdinStrings {
					c.WriteToStdin(s)
				}

				// Wait for goroutine to finish
				select {
				case <-ctx.Done():
					c.Stop() // stop redirecting stdin/stdout
				}

				// If method timed out, it probably means that there was an unexpected fmt.Scanln()
				if ctx.Err() == context.DeadlineExceeded {

					t.Error("Function '" + testFuncs[i].Name +
						"' timed out before completing. Do you have an extra fmt.Scanln, perhaps?")

				} else {

					if !testFuncs[i].IgnoreReturns {

						for j := 0; j < len(testFuncs[i].Returns); j++ {

							// For testing logic
							//fmt.Println("Expected:", testObjs[i].Returns[j].Interface(), "Actual:", returnVals[j].Interface())

							if testFuncs[i].Returns[j].Interface() != returnVals[j].Interface() {

								t.Error("Function '" + testFuncs[i].Name +
									"' returned unexpected value. Specifically, return value position " + strconv.Itoa(j) + ".")
							}
						}
					}

					if !testFuncs[i].IgnoreStdout {

						if len(testFuncs[i].StdoutStrings) == len(c.OutData) {

							for j := 0; j < len(testFuncs[i].StdoutStrings); j++ {
		
								if testFuncs[i].StdoutStrings[j] != strings.TrimSpace(c.OutData[j]) {

									t.Error("Function '" + testFuncs[i].Name +
										"' displayed unexpected line to the terminal. Unexpected line was \"" + c.OutData[j] + "\".")
		
									// t.Error("Function '" + testFuncs[i].Name +
									// 	"' displayed unexpected line to the terminal. Expected \"" +
									// 	testFuncs[i].StdoutStrings[j] + "\", found \"" + c.OutData[j] + "\".")
								}
							}

						} else {

							// For testing
							// for _, line := range c.OutData {
							// 	fmt.Println(line)
							// }


							t.Error("Function '" + testFuncs[i].Name +
								"' displayed unexpected number of output lines to the terminal. Expected " + 
								strconv.Itoa(len(testFuncs[i].StdoutStrings)) +
								" line(s), found " + strconv.Itoa(len(c.OutData))  + " line(s)")
						}
					}

				}

			} else {
				t.Error("'" + testFuncs[i].Name + "' function definition missing.")
			}

		}

	}
}