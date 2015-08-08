package main

import (
	//"encoding/json"
	"fmt"
	"github.com/BurntSushi/toml"
)

func testAnalyze(part map[string]interface{}, v interface{}) {
	// data, err := json.Marshal(&part)
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return
	// }
	// //fmt.Printf("%s\n", string(data))

	// err = json.Unmarshal(data, v)
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return
	// }

}
func main() {
	var testSimple = []byte(`
age = 250
[Part1]
plato = "cat 1"
cauchy = "cat 2"
`)
	type Part1 struct {
		Plato  string
		Cauchy string
	}
	mm := make(map[string]toml.Primitive)
	md, err := toml.Decode(string(testSimple), mm)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	p1 := &Part1{}
	err = md.PrimitiveDecode(mm["Part1"], p1)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Printf("%s\t%s\n", p1.Plato, p1.Cauchy)

	// mm := make(map[string]interface{})
	// err := toml.Unmarshal(testSimple, mm)
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	return
	// }
	// //fmt.Println(mm["Part1"])

	// part1 := mm["Part1"].(map[string]interface{})
	// //fmt.Println(part1["plato"])
	// p1 := &Part1{}
	// testAnalyze(part1, p1)

	// fmt.Printf("%s\t%s\n", p1.Plato, p1.Cauchy)
}
