// Command difftest is the Go side of the cross-implementation differential
// test. It reads tab-separated cases on stdin and prints one result line per
// case in exactly the format of the Rust reference harness, so the two outputs
// can be compared with diff.
//
//	D <id-hex> <decimal>   decimal conversion  -> OK <value> <rounded> <atom-hex> | ERR
//	A <id-hex> <value>     atom construction   -> OK <atom-hex> | ERR
//	X <hex>                atom decoding       -> OK <id> <value> | ERR
package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	omwei "github.com/omwei-org/omwei-iot-go"
)

func main() {
	in := bufio.NewReaderSize(os.Stdin, 1<<20)
	out := bufio.NewWriter(os.Stdout)
	defer out.Flush()
	for {
		line, err := in.ReadString('\n')
		if line != "" {
			fmt.Fprintln(out, run(strings.TrimSuffix(line, "\n")))
		}
		if err != nil {
			return
		}
	}
}

func run(line string) string {
	p := strings.SplitN(line, "\t", 3)
	for len(p) < 3 {
		p = append(p, "")
	}
	switch p[0] {
	case "D":
		id, _ := strconv.ParseUint(p[1], 16, 16)
		r, err := omwei.FromDecimalWithReport(uint16(id), p[2])
		if err != nil {
			return "ERR"
		}
		e := omwei.Encode(r.Atom)
		return fmt.Sprintf("OK %d %t %s", r.Atom.CanonicalValue, r.Rounded, hex.EncodeToString(e[:]))
	case "A":
		id, _ := strconv.ParseUint(p[1], 16, 16)
		v, _ := strconv.ParseInt(p[2], 10, 32)
		a, err := omwei.NewAtom(uint16(id), int32(v))
		if err != nil {
			return "ERR"
		}
		e := omwei.Encode(a)
		return "OK " + hex.EncodeToString(e[:])
	case "X":
		b, _ := hex.DecodeString(p[1])
		a, err := omwei.Decode(b)
		if err != nil {
			return "ERR"
		}
		return fmt.Sprintf("OK %d %d", a.DescriptorID, a.CanonicalValue)
	}
	return "BAD"
}
