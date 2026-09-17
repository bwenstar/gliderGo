package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/bwenstar/gliderGo/internal/house"
)

// typesCmd prints the object type table. It exists for hand-authoring: the text
// format names objects by their original C identifier, and nothing else in the
// tree lists all 117 of them next to the union variant each one selects, which is
// what tells an author which fields the line must carry.
func typesCmd(args []string) error {
	fs := flag.NewFlagSet("types", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	filter := strings.ToLower(strings.Join(fs.Args(), " "))

	names := house.ObjectNames()
	codes := make([]int, 0, len(names))
	for code := range names {
		if filter != "" && !strings.Contains(strings.ToLower(names[code]), filter) &&
			!strings.HasPrefix(fmt.Sprintf("0x%02x", uint16(code)), filter) {
			continue
		}
		codes = append(codes, int(code))
	}
	if len(codes) == 0 {
		return fmt.Errorf("no object type matches %q", filter)
	}
	sort.Ints(codes)

	// The variants come first, as their own short table: nine rows explain the
	// fields, and repeating them down 117 code rows would bury the codes.
	used := map[house.Group]bool{}
	for _, code := range codes {
		used[house.GroupOf(int16(code))] = true
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "variant\tunion\tfields every object line of the variant must carry")
	for g := house.GroupBlower; g <= house.GroupClutter; g++ {
		if used[g] {
			fmt.Fprintf(w, "%s\t.%s\t%s\n", g, g.Member(), groupFields(g))
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	fmt.Println()

	w = tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
	fmt.Fprintln(w, "code\tname\tvariant")
	for _, code := range codes {
		fmt.Fprintf(w, "0x%02X\t%s\t%s\n", code, names[int16(code)], house.GroupOf(int16(code)))
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return nil
}

// groupFields renders a variant's required fields as an author would type them.
// The names and their order come from the house package, which is also what the
// parser demands, so this column cannot promise a field the parser rejects. Only
// the arity hints are added here: `at 200 64`, not a bare `at`.
func groupFields(g house.Group) string {
	names := g.Fields()
	if names == nil {
		return "raw <10 hex bytes>"
	}
	hints := map[string]string{"at": "at V H", "rect": "rect T L B R"}
	for i, n := range names {
		if h, ok := hints[n]; ok {
			names[i] = h
		}
	}
	return strings.Join(names, "  ")
}
