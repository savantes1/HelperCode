package helpers

import (
	"context"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/savantes1/outcap"
)

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
				c, _ := outcap.NewContainer('\n')

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
										"' displayed unexpected output to the terminal. Unexpected output line: " + string(j+1))

									// t.Error("Function '" + testFuncs[i].Name +
									// 	"' displayed unexpected output line to the terminal. Unexpected line was \"" + c.OutData[j] + "\".")
		
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








// Method anatomy testing struct
type MethodAnatomyTest struct {
	Name         string
	ArgTypes     []reflect.Type
	ReturnTypes  []reflect.Type
}

// Runs standard struct method anatomy tests using provided values
// IMPORTANT: testObject must be a pointer to the struct object being tested!
func RunMethodAnatomyTests(testObject interface{}, methodTests []MethodAnatomyTest, t *testing.T) {

	for i := 0; i < len(methodTests); i++ {

		method := reflect.ValueOf(testObject).MethodByName(methodTests[i].Name)

		if method.IsValid() {

			if method.Type().NumIn() == len(methodTests[i].ArgTypes) {

				for j := 0; j < len(methodTests[i].ArgTypes); j++ {

					param := method.Type().In(j)

					if param != methodTests[i].ArgTypes[j] {

						t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTests[i].Name +
							"' has unexpected parameter type at position " + strconv.Itoa(j) + ". Expected type " +
							methodTests[i].ArgTypes[j].String() + ", found type " + param.String())
					}
				}

			} else {

				t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTests[i].Name +
					"' has unexpected number of parameters. Expected " + strconv.Itoa(len(methodTests[i].ArgTypes)) +
					" parameter(s), found " + strconv.Itoa(method.Type().NumIn()) + " parameter(s)")
			}

			if method.Type().NumOut() == len(methodTests[i].ReturnTypes) {

				for j := 0; j < len(methodTests[i].ReturnTypes); j++ {

					param := method.Type().Out(j)

					if param != methodTests[i].ReturnTypes[j] {

						t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTests[i].Name +
							"' returned unexpected data type for return " + strconv.Itoa(j) + ". Expected type " +
							methodTests[i].ReturnTypes[j].String() + ", received type " + param.String())
					}

				}

			} else {
				t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTests[i].Name +
					"' returns unexpected number of values. Expected " + strconv.Itoa(len(methodTests[i].ReturnTypes)) +
					" value(s), received " + strconv.Itoa(method.Type().NumOut()) + " value(s)")
			}

		} else {
			t.Error(reflect.TypeOf(testObject).Elem().Name() + " struct definition missing '" + methodTests[i].Name + "' method")
		}

	}
}

// Method output testing struct
type MethodOutputTest struct {
	Name          string
	Args          []reflect.Value
	StdinStrings  []string
	IgnoreStdout  bool
	StdoutStrings []string
	IgnoreReturns bool
	Returns       []reflect.Value
}



// Converts MethodOutputTest object to MethodAnatomyTest object
func convertMethodOutputTestToAnatomyTest(ot MethodOutputTest) MethodAnatomyTest {

	var argTypes []reflect.Type
	var returnTypes []reflect.Type

	for _, el := range ot.Args {
		argTypes = append(argTypes, el.Type())
	}

	for _, el := range ot.Returns {
		returnTypes = append(returnTypes, el.Type())
	}

	return MethodAnatomyTest {

		Name: ot.Name,
		ArgTypes: argTypes,
		ReturnTypes: returnTypes,
	}
}



// Runs standard struct method output tests using provided values
// IMPORTANT: testObject must be a pointer to the struct object being tested!
func RunMethodOutputTests(testObject interface{}, methodTests []MethodOutputTest, t *testing.T) {

	for i := 0; i < len(methodTests); i++ {

		// Run the anatomy test on the method first
		RunMethodAnatomyTests(testObject, []MethodAnatomyTest{convertMethodOutputTestToAnatomyTest(methodTests[i])}, t)

		// Don't even bother running the actual output tests if the anatomy tests failed
		if !t.Failed() {

			method := reflect.ValueOf(testObject).MethodByName(methodTests[i].Name)

			if method.IsValid() {

				//TODO: handle errors, maybe?
				c, _ := outcap.NewContainer('\n')

				// Create context for goroutine with timeout in case the method
				// tries to process more stdin input than what is expected. Running
				// in goroutine with context allows for easily timing out after 3 seconds.
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

				var returnVals []reflect.Value

				go func() {

					// In case this method uses random numbers, make sure to set
					// the seed to 1 so the "random" numbers will be predictable
					// (deterministic) and will match the expected output in the
					// methodReturns values
					rand.Seed(1)

					// Call the method...
					// Note: This logic does not support variadic functions!
					returnVals = method.Call(methodTests[i].Args)
					cancel()

				}()

				// Write each input string to method
				for _, s := range methodTests[i].StdinStrings {
					c.WriteToStdin(s)
				}

				select {
				case <-ctx.Done():
					c.Stop() // stop redirecting stdin/stdout
				}

				// If method timed out, it probably means that there was an unexpected fmt.Scanln()
				if ctx.Err() == context.DeadlineExceeded {

					t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTests[i].Name +
						"' timed out before completing. Do you have an extra fmt.Scanln, perhaps?")

				} else {


					if !methodTests[i].IgnoreReturns {

						for j := 0; j < len(methodTests[i].Returns); j++ {

							// For testing logic
							//fmt.Println("Expected:", methodReturns[i][j].Interface(), "Actual:", returnVals[j].Interface())
	
							if methodTests[i].Returns[j].Interface() != returnVals[j].Interface() {
	
								t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTests[i].Name +
									"' returned unexpected value. Specifically, return value position " + strconv.Itoa(j) + ".")
							}
	
						}
					}




					if !methodTests[i].IgnoreStdout {

						if len(methodTests[i].StdoutStrings) == len(c.OutData) {

							for j := 0; j < len(methodTests[i].StdoutStrings); j++ {
		
								if methodTests[i].StdoutStrings[j] != strings.TrimSpace(c.OutData[j]) {

									t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTests[i].Name +
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


							t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTests[i].Name +
								"' displayed unexpected number of output lines to the terminal. Expected " + 
								strconv.Itoa(len(methodTests[i].StdoutStrings)) +
								" line(s), found " + strconv.Itoa(len(c.OutData))  + " line(s)")
						}
					}				
				}

			} else {
				t.Error(reflect.TypeOf(testObject).Elem().Name() + " struct definition missing '" + methodTests[i].Name + "' method")
			}
		}
	}
}