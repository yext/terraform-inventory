package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
)

type counterSorter struct {
	resources []*Resource
}

func (cs counterSorter) Len() int {
	return len(cs.resources)
}

func (cs counterSorter) Swap(i, j int) {
	cs.resources[i], cs.resources[j] = cs.resources[j], cs.resources[i]
}

func (cs counterSorter) Less(i, j int) bool {
	return cs.resources[i].counter < cs.resources[j].counter
}

type allGroup struct {
	Hosts []string               `json:"hosts"`
	Vars  map[string]interface{} `json:"vars"`
}

func appendUniq(strs []string, item string) []string {
	if len(strs) == 0 {
		strs = append(strs, item)
		return strs
	}
	sort.Strings(strs)
	i := sort.SearchStrings(strs, item)
	if i == len(strs) || (i < len(strs) && strs[i] != item) {
		strs = append(strs, item)
	}
	return strs
}

func gatherResources(s *state) (map[string][]string, map[string]map[string]string) {
	inventoryMap := make(map[string][]string)
	varsMap := make(map[string]map[string]string)

	for _, res := range s.resources() {
		groups := res.Groups()
		addr := res.Address()

		vars := res.Vars()
		varsMap[addr] = vars

		for _, group := range groups {
			_, ok := inventoryMap[group]
			if !ok {
				inventoryMap[group] = []string{}
			}
			inventoryMap[group] = appendUniq(inventoryMap[group], addr)
		}
	}

	// If you want to sort the hosts within the groups, do it here

	return inventoryMap, varsMap
}

func cmdList(stdout io.Writer, stderr io.Writer, s *state) int {
	groups, vars := gatherResources(s)
	inventory := make(map[string]interface{})
	for k, v := range groups {
		inventory[k] = v
	}
	inventory["_meta"] = vars
	return output(stdout, stderr, inventory)
}

func cmdInventory(stdout io.Writer, stderr io.Writer, s *state) int {
	groups, vars := gatherResources(s)
	group_names := []string{}
	for group, _ := range groups {
		group_names = append(group_names, group)
	}
	sort.Strings(group_names)
	for _, group := range group_names {

		// switch grp := groups[group].(type) {
		// case []string:
		writeLn("["+group+"]", stdout, stderr)
		for _, host := range groups[group] {
			_, err := io.WriteString(stdout, host)
			checkErr(err, stderr)
			for k, v := range vars[host] {
				_, err := io.WriteString(stdout, " "+k+"="+v)
				checkErr(err, stderr)
			}
			_, err = io.WriteString(stdout, "\n")
			checkErr(err, stderr)
		}

		writeLn("", stdout, stderr)
	}

	return 0
}

func writeLn(str string, stdout io.Writer, stderr io.Writer) {
	_, err := io.WriteString(stdout, str+"\n")
	checkErr(err, stderr)
}

func checkErr(err error, stderr io.Writer) int {
	if err != nil {
		fmt.Fprintf(stderr, "Error writing inventory: %s\n", err)
		return 1
	}
	return 0
}

func cmdHost(stdout io.Writer, stderr io.Writer, s *state, hostname string) int {
	for _, res := range s.resources() {
		if hostname == res.Address() {
			return output(stdout, stderr, res.Attributes())
		}
	}

	fmt.Fprintf(stdout, "{}")
	return 1
}

// output marshals an arbitrary JSON object and writes it to stdout, or writes
// an error to stderr, then returns the appropriate exit code.
func output(stdout io.Writer, stderr io.Writer, whatever interface{}) int {
	b, err := json.Marshal(whatever)
	if err != nil {
		fmt.Fprintf(stderr, "Error encoding JSON: %s\n", err)
		return 1
	}

	_, err = stdout.Write(b)
	if err != nil {
		fmt.Fprintf(stderr, "Error writing JSON: %s\n", err)
		return 1
	}

	return 0
}
