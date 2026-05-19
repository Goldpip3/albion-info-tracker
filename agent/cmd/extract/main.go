// extract is a one-shot CLI for decrypting Albion's GameData .bin files
// and dumping the embedded XML to stdout or a directory of JSON files.
//
// Usage:
//
//	extract decrypt <path/to/items.bin>            (XML to stdout)
//	extract dump <albion-install-root> <outdir>    (decrypt items/spells/mobs)
package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Goldpip3/albion-info-tracker/agent/internal/gamedata"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "decrypt":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		decrypt(os.Args[2])
	case "dump":
		if len(os.Args) < 4 {
			usage()
			os.Exit(2)
		}
		dump(os.Args[2], os.Args[3])
	case "spells":
		if len(os.Args) < 3 {
			usage()
			os.Exit(2)
		}
		spells(os.Args[2])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Usage:")
	fmt.Fprintln(os.Stderr, "  extract decrypt <path/to/file.bin>          XML to stdout")
	fmt.Fprintln(os.Stderr, "  extract dump <albion-install> <outdir>      decrypt items+spells+mobs as JSON")
	fmt.Fprintln(os.Stderr, "  extract spells <albion-install>             list spell catalog (Index UniqueName)")
}

func decrypt(path string) {
	xmlBytes, err := gamedata.DecryptAndDecompress(path)
	if err != nil {
		fail("decrypt", err)
	}
	os.Stdout.Write(xmlBytes)
}

func dump(installRoot, outDir string) {
	if err := gamedata.IsValidInstall(installRoot, gamedata.ServerLive); err != nil {
		fail("validate install", err)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fail("mkdir", err)
	}
	binDir := gamedata.BinDir(installRoot, gamedata.ServerLive)
	for _, name := range []string{"items", "spells", "mobs"} {
		src := filepath.Join(binDir, name+".bin")
		dst := filepath.Join(outDir, name+".json")
		xmlBytes, err := gamedata.DecryptAndDecompress(src)
		if err != nil {
			fail(name, err)
		}
		jsonBytes, err := xmlToJSON(xmlBytes)
		if err != nil {
			fail(name+" xml→json", err)
		}
		if err := os.WriteFile(dst, jsonBytes, 0o644); err != nil {
			fail(name+" write", err)
		}
		fmt.Fprintf(os.Stderr, "%s → %s (%d bytes)\n", src, dst, len(jsonBytes))
	}
}

func spells(installRoot string) {
	cat, err := gamedata.LoadSpellCatalog(installRoot, gamedata.ServerLive)
	if err != nil {
		fail("load spells", err)
	}
	// Print as TSV: index<tab>uniquename — easy to grep.
	for i := 0; i < 100000; i++ {
		if name := cat.Name(i); name != "" {
			fmt.Printf("%d\t%s\n", i, name)
		}
	}
}

// xmlToJSON walks the XML stream and emits a hierarchical JSON tree where
// every element becomes an object with its attributes inlined as top-level
// fields and its children grouped under their tag name. Repeated child tags
// become arrays. Not a full XML→JSON spec, but good enough for the simple
// row-oriented Albion GameData files.
func xmlToJSON(xmlBytes []byte) ([]byte, error) {
	dec := xml.NewDecoder(strings.NewReader(string(xmlBytes)))
	root, err := readElement(dec, nil)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(root, "", "  ")
}

func readElement(dec *xml.Decoder, start *xml.StartElement) (map[string]any, error) {
	if start == nil {
		// find the first start element
		for {
			tok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			if s, ok := tok.(xml.StartElement); ok {
				start = &s
				break
			}
		}
	}
	node := make(map[string]any, len(start.Attr)+2)
	for _, a := range start.Attr {
		node["@"+a.Name.Local] = a.Value
	}
	for {
		tok, err := dec.Token()
		if err != nil {
			return node, nil
		}
		switch t := tok.(type) {
		case xml.StartElement:
			child, err := readElement(dec, &t)
			if err != nil {
				return nil, err
			}
			key := t.Name.Local
			switch existing := node[key].(type) {
			case nil:
				node[key] = child
			case []any:
				node[key] = append(existing, child)
			default:
				node[key] = []any{existing, child}
			}
		case xml.EndElement:
			return node, nil
		}
	}
}

func fail(stage string, err error) {
	fmt.Fprintf(os.Stderr, "extract %s: %v\n", stage, err)
	os.Exit(1)
}
