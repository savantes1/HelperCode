package helpers

import (
	"context"
	"math/rand"
	"reflect"
	"regexp"
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

// TODO: combine this and RunFunctionOutputTests after refactoring code that directly called RunFunctionOutputTests
func RunFunctionOutputTestsWithRandomSeed(testFuncs []FuncOutputTest, randomSeed int64, t *testing.T) {

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
					// the seed to specified seed value so the "random" numbers will 
					// be predictable (deterministic) and will match the expected output
					rand.Seed(randomSeed)

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
										"' displayed unexpected output to the terminal. Unexpected output line: " + strconv.Itoa(j+1))

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


// Runs standard function output tests using provided values
func RunFunctionOutputTests(testFuncs []FuncOutputTest, t *testing.T) {

	RunFunctionOutputTestsWithRandomSeed(testFuncs, 1, t)
	
}








// Method anatomy testing struct
type MethodAnatomyTest struct {
	Name         string
	ArgTypes     []reflect.Type
	ReturnTypes  []reflect.Type
}



// Runs standard struct method anatomy test using provided values.
// Returns true if anatomy passes tests. Otherwise returns false.
// IMPORTANT: testObject must be a pointer to the struct object being tested!
func RunMethodAnatomyTest(testObject interface{}, methodTest MethodAnatomyTest, t *testing.T) bool {

	passedTests := true

	method := reflect.ValueOf(testObject).MethodByName(methodTest.Name)

	if method.IsValid() {

		if method.Type().NumIn() == len(methodTest.ArgTypes) {

			for j := 0; j < len(methodTest.ArgTypes); j++ {

				param := method.Type().In(j)

				if param != methodTest.ArgTypes[j] {

					t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTest.Name +
						"' has unexpected parameter type at position " + strconv.Itoa(j) + ". Expected type " +
						methodTest.ArgTypes[j].String() + ", found type " + param.String())
					
					passedTests = false
				}
			}

		} else {

			t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTest.Name +
				"' has unexpected number of parameters. Expected " + strconv.Itoa(len(methodTest.ArgTypes)) +
				" parameter(s), found " + strconv.Itoa(method.Type().NumIn()) + " parameter(s)")

			passedTests = false
		}

		if method.Type().NumOut() == len(methodTest.ReturnTypes) {

			for j := 0; j < len(methodTest.ReturnTypes); j++ {

				param := method.Type().Out(j)

				if param != methodTest.ReturnTypes[j] {

					t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTest.Name +
						"' returned unexpected data type for return " + strconv.Itoa(j) + ". Expected type " +
						methodTest.ReturnTypes[j].String() + ", received type " + param.String())
					
					passedTests = false
				}

			}

		} else {
			t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTest.Name +
				"' returns unexpected number of values. Expected " + strconv.Itoa(len(methodTest.ReturnTypes)) +
				" value(s), received " + strconv.Itoa(method.Type().NumOut()) + " value(s)")

			passedTests = false
		}

	} else {
		t.Error(reflect.TypeOf(testObject).Elem().Name() + " struct definition missing '" + methodTest.Name + "' method")
		passedTests = false
	}

	return passedTests

}


// Runs standard struct method anatomy tests using provided values
// IMPORTANT: testObject must be a pointer to the struct object being tested!
func RunMethodAnatomyTests(testObject interface{}, methodTests []MethodAnatomyTest, t *testing.T) {

	for i := 0; i < len(methodTests); i++ {
		RunMethodAnatomyTest(testObject, methodTests[i], t)
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



// Runs standard struct method output test.
// Returns provided object after method has been invoked for further evaluation.  
// IMPORTANT: testObject must be a pointer to the struct object being tested!
func RunMethodOutputTest(testObject interface{}, methodTest MethodOutputTest, randomSeed int64, t *testing.T) reflect.Value {

	// Run the anatomy test on the method first.
	// Don't even bother running the actual output tests if the anatomy tests failed
	if RunMethodAnatomyTest(testObject, convertMethodOutputTestToAnatomyTest(methodTest), t) {

		method := reflect.ValueOf(testObject).MethodByName(methodTest.Name)

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
				// the seed to specified seed value so the "random" numbers will 
				// be predictable (deterministic) and will match the expected 
				// output in the methodReturns values
				rand.Seed(randomSeed)

				// Call the method...
				// Note: This logic does not support variadic functions!
				returnVals = method.Call(methodTest.Args)
				cancel()

			}()

			// Write each input string to method
			for _, s := range methodTest.StdinStrings {
				c.WriteToStdin(s)
			}

			select {
			case <-ctx.Done():
				c.Stop() // stop redirecting stdin/stdout
			}

			// If method timed out, it probably means that there was an unexpected fmt.Scanln()
			if ctx.Err() == context.DeadlineExceeded {

				t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTest.Name +
					"' timed out before completing. Do you have an extra fmt.Scanln, perhaps?")

			} else {


				if !methodTest.IgnoreReturns {

					for j := 0; j < len(methodTest.Returns); j++ {

						// For testing logic
						//fmt.Println("Expected:", methodReturns[i][j].Interface(), "Actual:", returnVals[j].Interface())

						if methodTest.Returns[j].Interface() != returnVals[j].Interface() {

							t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTest.Name +
								"' returned unexpected value. Specifically, return value position " + strconv.Itoa(j) + ".")
						}

					}
				}




				if !methodTest.IgnoreStdout {

					if len(methodTest.StdoutStrings) == len(c.OutData) {

						for j := 0; j < len(methodTest.StdoutStrings); j++ {
	
							if methodTest.StdoutStrings[j] != strings.TrimSpace(c.OutData[j]) {

								t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTest.Name +
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


						t.Error(reflect.TypeOf(testObject).Elem().Name() + " method '" + methodTest.Name +
							"' displayed unexpected number of output lines to the terminal. Expected " + 
							strconv.Itoa(len(methodTest.StdoutStrings)) +
							" line(s), found " + strconv.Itoa(len(c.OutData))  + " line(s)")
					}
				}				
			}

		} else {
			t.Error(reflect.TypeOf(testObject).Elem().Name() + " struct definition missing '" + methodTest.Name + "' method")
		}
	}
	
	return reflect.ValueOf(testObject)

}


// TODO: combine this and RunMethodOutputTests after refactoring code that directly called RunMethodOutputTests
func RunMethodOutputTestsWithRandomSeed(testObject interface{}, methodTests []MethodOutputTest, randomSeed int64, t *testing.T) {

	for i := 0; i < len(methodTests); i++ {

		RunMethodOutputTest(testObject, methodTests[i], randomSeed, t)
	}

}



// Runs standard struct method output tests using provided values
// IMPORTANT: testObject must be a pointer to the struct object being tested!
func RunMethodOutputTests(testObject interface{}, methodTests []MethodOutputTest, randomSeed int64, t *testing.T) {

	RunMethodOutputTestsWithRandomSeed(testObject, methodTests, 1, t)
	
}




// Removes all comments from the specified text
// and returns filtered text
func RemoveAllComments(text string) string {

	// match all comment blocks
	re := regexp.MustCompile(`(?s)/\*.*?\*/`)

	filteredText := re.ReplaceAllString(string(text), "")

	// match all single line comments
	re = regexp.MustCompile(`//.*`)

	return re.ReplaceAllString(filteredText, "")
}

// Tests specified code text to confirm using proper
// random number seeding template
func RunRandomNumberTemplateTest(text string, t *testing.T) {

	filteredText := RemoveAllComments(text)

	// regex: \s = [\t\n\f\r ]
	// regex: \S = [^\t\n\f\r ]

	matched, _ := regexp.MatchString(`(?s)func[ ]+init[ ]*\([ ]*\)[ ]*\{\s*rand.Seed\(int64\(time.Now\(\).Nanosecond\(\)\)\)\s*\}`, filteredText)

	if !matched {
		t.Error("Program doesn't seed random number generator as demonstrated in \"Random Numbers\" example code")
	}

	re := regexp.MustCompile(`rand.Seed`)
	matches := re.FindAllStringSubmatch(filteredText, -1)

	if len(matches) > 1 {
		t.Error("Random number generator seeded more than once")
	}
}


// Tests specified code text to make sure the correct number of 
// the specified objects are instantiated in the code
func RunInstantiateObjectsTest(text string, objectName string, minObjectCount int, maxObjectCount int, t *testing.T) {
	RunInstantiateObjectsTestWithFunctionName(text, objectName, minObjectCount, maxObjectCount, "", t)
}




// Tests specified code text to make sure the correct number of
// the specified objects are instantiated in the code.
func RunInstantiateObjectsTestWithFunctionName(text string, objectName string, minObjectCount int, maxObjectCount int, objInstantiatedFuncName string, t *testing.T) {

	filteredText := RemoveAllComments(text)

	// keep track of the number of objects instantiated
	var objectCounter int = 0

	// find all of the objects instantiated the traditional way
	re := regexp.MustCompile(`var[ ]+[\w]+[ ]+` + objectName)
	matches := re.FindAllStringSubmatch(filteredText, -1)

	objectCounter += len(matches)

	// find all of the objects instantiated using the short syntax
	re = regexp.MustCompile(`(?s):=[ ]*`+objectName+`[ ]*\{.*?\}`)
	matches = re.FindAllStringSubmatch(filteredText, -1)

	objectCounter += len(matches)

	// Figure out if there are too many or too few of the instantiated objects
	if minObjectCount == maxObjectCount {
		if objectCounter != minObjectCount {

			errorMessage := "Program must instantiate " + strconv.Itoa(minObjectCount) + " \"" + objectName + "\" object variable(s)"
			if objInstantiatedFuncName != "" {
				errorMessage += " in function \"" + objInstantiatedFuncName + "\""
			}

			t.Error(errorMessage)
		}
	} else {

		if objectCounter < minObjectCount {

			errorMessage := "Program must instantiate at least " + strconv.Itoa(minObjectCount) + " \"" + objectName + "\" object variable(s)"
			if objInstantiatedFuncName != "" {
				errorMessage += " in function \"" + objInstantiatedFuncName + "\""
			}

			t.Error(errorMessage)
		}

		if objectCounter > maxObjectCount {

			errorMessage := "Program cannot instantiate more than " + strconv.Itoa(maxObjectCount) + " \"" + objectName + "\" object variable(s)"
			if objInstantiatedFuncName != "" {
				errorMessage += " in function \"" + objInstantiatedFuncName + "\""
			}

			t.Error(errorMessage)
		}
	}
}

// Returns the specified function body text.
// First return value = true if function was found, otherwise false
// Second return value = function body text
func GetFunctionBodyText(text string, functionName string) (bool, string) {
	
	// remove any comments first
	noCommentsText := RemoveAllComments(text)

	re := regexp.MustCompile(`(?s)func[^\n]*`+functionName+`[^\n]*\([^\n]*\)[^\n]*\{`)
	
	functionSignatureIndexes := re.FindStringIndex(noCommentsText)

	// FindStringIndex will be nil if no match found
	// This means the function name wasn't found
	if functionSignatureIndexes == nil {
		return false, ""
	}
	
	// Count the number curly braces
	// We start at one for the opening curly brace for the function
	curlyBraceCount := 1

	// Remove all text before the function opening curly brace
	filteredText := noCommentsText[functionSignatureIndexes[1]:]

	var functionEndingIndex int

	// Loop through each character in the text
	// counting each opening and closing curly brace
	for i, ch := range filteredText {

		// rune 123 = '{'
		if ch == 123 {
			curlyBraceCount += 1
		}

		// rune 125 = '}'
		if ch == 125 {
			curlyBraceCount -= 1
		}

		// This only hits 0 when function closing
		// curly brace is hit
		if curlyBraceCount == 0 {
			functionEndingIndex = i
			break
		}
	}

	return true, filteredText[:functionEndingIndex]
}


// FlagType enum
type FlagType int

// FlagType enum values
const (
	IntFlag FlagType = iota
	StringFlag FlagType = iota
	FloatFlag FlagType = iota
	BoolFlag FlagType = iota
)

// Tests specified code text to make sure specified command 
// line flags are mapped to a variable of the appropriate type
func RunValidateFlagArgTest(text string, flagType FlagType, flagName string, t *testing.T) {

	filteredText := RemoveAllComments(text)

	var functionName string 
	switch flagType {
	case IntFlag:
		functionName = "IntVar"
	case StringFlag:
		functionName = "StringVar"
	case FloatFlag:
		functionName = "Float64Var"
	case BoolFlag:
		functionName = "BoolVar"
	}

	if len(functionName) > 0 {

		re := regexp.MustCompile(`flag[\t ]*\.[\t ]*`+functionName+`[\t ]*\(.+?,[\t ]*"`+flagName+`"`)
		if !re.MatchString(filteredText) {
			t.Error("Must use \"flag\" package to map \""+flagName+"\" command line argument")
		}

	} else {
		panic("Unexpected Flag Type")
	}

}


