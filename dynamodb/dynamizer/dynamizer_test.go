package dynamizer

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

type mySimpleStruct struct {
	String  string
	Int     int
	Uint    uint
	Float32 float32
	Float64 float64
	Bool    bool
	Null    *interface{}
}

type myComplexStruct struct {
	Simple []mySimpleStruct
}

type dynamizerTestInput struct {
	input    interface{}
	expected string
}

var dynamizerTestInputs = []dynamizerTestInput{
	// Scalar tests
	dynamizerTestInput{
		input:    map[string]interface{}{"string": "some string"},
		expected: `{"string":{"S":"some string"}}`},
	dynamizerTestInput{
		input:    map[string]interface{}{"bool": true},
		expected: `{"bool":{"BOOL":true}}`},
	dynamizerTestInput{
		input:    map[string]interface{}{"bool": false},
		expected: `{"bool":{"BOOL":false}}`},
	dynamizerTestInput{
		input:    map[string]interface{}{"null": nil},
		expected: `{"null":{"NULL":true}}`},
	dynamizerTestInput{
		input:    map[string]interface{}{"float": 3.14},
		expected: `{"float":{"N":"3.14"}}`},
	dynamizerTestInput{
		input:    map[string]interface{}{"float": math.MaxFloat32},
		expected: `{"float":{"N":"340282346638528860000000000000000000000"}}`},
	dynamizerTestInput{
		input:    map[string]interface{}{"float": math.MaxFloat64},
		expected: `{"float":{"N":"179769313486231570000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"}}`},
	dynamizerTestInput{
		input:    map[string]interface{}{"int": int(12)},
		expected: `{"int":{"N":"12"}}`},
	// List
	dynamizerTestInput{
		input:    map[string]interface{}{"list": []interface{}{"a string", 12, 3.14, true, nil, false}},
		expected: `{"list":{"L":[{"S":"a string"},{"N":"12"},{"N":"3.14"},{"BOOL":true},{"NULL":true},{"BOOL":false}]}}`},
	// Map
	dynamizerTestInput{
		input:    map[string]interface{}{"map": map[string]interface{}{"nestedint": 12}},
		expected: `{"map":{"M":{"nestedint":{"N":"12"}}}}`},
	dynamizerTestInput{
		input:    &map[string]interface{}{"map": map[string]interface{}{"nestedint": 12}},
		expected: `{"map":{"M":{"nestedint":{"N":"12"}}}}`},
	// Structs
	dynamizerTestInput{
		input:    mySimpleStruct{},
		expected: `{"Bool":{"BOOL":false},"Float32":{"N":"0"},"Float64":{"N":"0"},"Int":{"N":"0"},"Null":{"NULL":true},"String":{"S":""},"Uint":{"N":"0"}}`},
	dynamizerTestInput{
		input:    &mySimpleStruct{},
		expected: `{"Bool":{"BOOL":false},"Float32":{"N":"0"},"Float64":{"N":"0"},"Int":{"N":"0"},"Null":{"NULL":true},"String":{"S":""},"Uint":{"N":"0"}}`},
	dynamizerTestInput{
		input:    myComplexStruct{},
		expected: `{"Simple":{"NULL":true}}`},
	dynamizerTestInput{
		input:    myComplexStruct{Simple: []mySimpleStruct{mySimpleStruct{}, mySimpleStruct{}}},
		expected: `{"Simple":{"L":[{"M":{"Bool":{"BOOL":false},"Float32":{"N":"0"},"Float64":{"N":"0"},"Int":{"N":"0"},"Null":{"NULL":true},"String":{"S":""},"Uint":{"N":"0"}}},{"M":{"Bool":{"BOOL":false},"Float32":{"N":"0"},"Float64":{"N":"0"},"Int":{"N":"0"},"Null":{"NULL":true},"String":{"S":""},"Uint":{"N":"0"}}}]}}`},
}

func TestToDynamo(t *testing.T) {
	for _, test := range dynamizerTestInputs {
		testToDynamo(t, test.input, test.expected)
	}
}

func testToDynamo(t *testing.T, in interface{}, expectedString string) {
	var expected interface{}
	var buf bytes.Buffer
	buf.WriteString(expectedString)
	if err := json.Unmarshal(buf.Bytes(), &expected); err != nil {
		t.Error(err)
	}
	actual, err := ToDynamo(in)
	if err != nil {
		t.Error(err)
	}
	compareObjects(t, expected, actual)
}

func TestFromDynamo(t *testing.T) {
	// Using the same inputs from TestToDynamo, test the reverse mapping.
	for _, test := range dynamizerTestInputs {
		testFromDynamo(t, test.expected, test.input)
	}
}

func testFromDynamo(t *testing.T, inputString string, expected interface{}) {
	var item DynamoItem
	var buf bytes.Buffer
	buf.WriteString(inputString)
	if err := json.Unmarshal(buf.Bytes(), &item); err != nil {
		t.Error(err)
	}
	var actual map[string]interface{}
	if err := FromDynamo(item, &actual); err != nil {
		t.Error(err)
	}
	compareObjects(t, expected, actual)
}

func TestStruct(t *testing.T) {
	// Test that we get a typed struct back
	expected := mySimpleStruct{String: "this is a string", Int: 1000000, Uint: 18446744073709551615, Float64: 3.14}
	dynamized, err := ToDynamo(expected)
	if err != nil {
		t.Error(err)
	}
	var actual mySimpleStruct
	err = FromDynamo(dynamized, &actual)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Did not get back the expected typed struct")
	}
}

// What we're trying to do here is compare the JSON encoded values, but we can't
// to a simple encode + string compare since JSON encoding is not ordered. So
// what we do is JSON encode, then JSON decode into untyped maps, and then
// finally do a recursive comparison.
func compareObjects(t *testing.T, expected interface{}, actual interface{}) {
	expectedBytes, eerr := json.Marshal(expected)
	if eerr != nil {
		t.Error(eerr)
		return
	}
	actualBytes, aerr := json.Marshal(actual)
	if aerr != nil {
		t.Error(aerr)
		return
	}
	var expectedUntyped, actualUntyped map[string]interface{}
	eerr = json.Unmarshal(expectedBytes, &expectedUntyped)
	if eerr != nil {
		t.Error(eerr)
		return
	}
	aerr = json.Unmarshal(actualBytes, &actualUntyped)
	if aerr != nil {
		t.Error(aerr)
		return
	}
	if !reflect.DeepEqual(expectedUntyped, actualUntyped) {
		t.Errorf("Expected %s, got %s", string(expectedBytes), string(actualBytes))
	}
}
