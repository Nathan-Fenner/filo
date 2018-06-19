package main

import "fmt"
import "io/ioutil"
import "bytes"
import "strings"
import "regexp"
import "os"

type Line struct{
	Source []byte
	Number int
}
type Block struct {
	Moniker   string
	Lines     []Line
	Generics  []string
	Needs     map[Need]bool
	Instances map[Need]bool
}

type Need struct {
	Moniker string
	Args    string // split by spaces
}

type Request struct {
	Replace string
	Needs   map[Need]bool
	Exact   string
}

func namifyString(input []byte) string {
	result := ""
	special := map[byte]string {
		'_': "__",
		'*': "_ptr_",
		'[': "_slice_",
		']': "_of_",
		'<': "_recv_",
		'>': "_send_",
		'-': "_only_",
		'#': "#",
	}
	for _, b := range input {
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' {
			result += string(rune(b))
		} else if b == ' ' || b == '\t' {
			// nothing
		} else if special[b] != "" {
			result += special[b]
		} else {
			result += fmt.Sprintf("_b%d", b)
		}
	}
	return result
}

func flattenType(src []byte) string {
	// TODO: this mangles channels
	out := ""
	for _, b := range src {
		if b == ' ' || b == '\t' {
			continue
		} else {
			out += string(rune(b))
		}
	}
	return out
}

func replaceGenerics(line string, assignment map[string]string) string {
	for vrb, rep := range assignment {
		line = strings.Replace(line, vrb, rep, -1)
	}
	return line
}

func extractGenerics(input []byte, namify bool) Request {
	// step one: check to see if a generic even exists in here.
	match := regexp.MustCompile(`\w+::\[`).FindIndex(input)
	if match == nil {
		if namify {
			return Request{Replace: namifyString(input), Exact: flattenType(input)}
		}
		return Request{Replace: string(input), Exact: flattenType(input)}
	}
	// otherwise, start the search at the last index.
	before := string(input[:match[0]])
	if namify {
		before = namifyString([]byte(before))
	}

	moniker := string(input[match[0]:match[1]-3])

	pieces := []string{}
	piece := ""
	stack := 0
	after := ""
	for i := match[1]; i < len(input); i++ {
		b := input[i]
		if stack == 0 && (b == ']' || b == ',') { 
			pieces = append(pieces, piece)
			piece = ""
			if b == ']' {
				after = string(input[i+1:])
				break
			}
		} else {
			piece += string(rune(b))
			if b == '[' || b == '(' || b == '{' {
				stack++
			}
			if b == ']' || b == ')' || b == '}' {
				stack--
			}
		}
	}

	totalNeed := map[Need]bool{}
	replaceWith := moniker
	selfNeed := ""
	for _, piece := range pieces {
		extracted := extractGenerics([]byte(piece), true)
		for need := range extracted.Needs {
			totalNeed[need] = true
		}
		replaceWith += "_" + extracted.Replace
		selfNeed += " " + extracted.Exact
	}
	replacedAfter := extractGenerics([]byte(after), namify)
	for need := range replacedAfter.Needs {
		totalNeed[need] = true
	}
	totalNeed[Need{Moniker: moniker, Args: strings.Trim(selfNeed, " ")}] = true
	return Request{
		Replace: before + replaceWith + replacedAfter.Replace,
		Needs: totalNeed,
		Exact: replaceWith,
	}
}

func main() {

	if len(os.Args) < 3 || os.Args[1] != "gen" {
		fmt.Printf("Expected invocation 'filo gen [path/src.filo]'.\n")
		return
	}

	fileSrc := string(os.Args[2])
	if !strings.HasSuffix(fileSrc, ".filo") {
		fmt.Printf("Expected invocation 'filo gen [path/src.filo]'.\n")
		return
	}
	fileTrg := string(fileSrc[:len(fileSrc)-4]) + "go"

	src, err := ioutil.ReadFile(fileSrc)
	if err != nil {
		fmt.Printf("Couldn't read file: %s\n", err)
		return
	}
	lines := bytes.Split(src, []byte("\n"))

	blocks := []Block{}
	current := Block{}

	for number, line := range lines {
		if bytes.HasPrefix(line, []byte("func ")) || bytes.HasPrefix(line, []byte("type ")) {
			blocks = append(blocks, current)
			current = Block{}
		}
		current.Lines = append(current.Lines, Line{Source: line, Number: number+1})
	}
	blocks = append(blocks, current)
	
	defn := map[string]*Block{}
	stack := []Need{}
	for blockN := range blocks {
		block := &blocks[blockN]
		block.Needs = map[Need]bool{}
		if len(block.Lines) == 0 {
			continue
		}
		header := block.Lines[0].Source
		if match := regexp.MustCompile(`(type|func) \w+::\[[#A-Z, ]+\]`).Find(header); match != nil {
			open := bytes.Index(match, []byte("["))
			close := bytes.Index(match, []byte("]"))
			args := bytes.Split(match[open+1:close], []byte(","))
			cleaned := []string{}
			for _, arg := range args {
				cleaned = append(cleaned, string(bytes.Trim(arg, " \t")))
			}
			moniker := match[5:open-2]
			block.Moniker = string(moniker)
			block.Generics = cleaned
		}
	}

	// merge receivers with their type definitions (fixes templating them)
	for blockN := range blocks {
		block := &blocks[blockN]
		if len(block.Lines) == 0 {
			continue
		}
		match := regexp.MustCompile(`func \(\w+\s*\*?\s*(\w+)::`).FindSubmatch(block.Lines[0].Source)
		if match == nil {
			continue
		}
		// add it to where it belongs
		for i := range blocks {
			if blocks[i].Moniker == string(match[1]) {
				// add it
				blocks[i].Lines = append(blocks[i].Lines, block.Lines...)
				block.Lines = nil
				break
			}
		}
	}

	for blockN := range blocks {
		block := &blocks[blockN]
		for i, line := range block.Lines {
			replaced := extractGenerics(line.Source, false)
			for need := range replaced.Needs {
				block.Needs[need] = true
			}
			block.Lines[i].Source = []byte(replaced.Replace)
		}
		if len(block.Generics) > 0 {
			defn[block.Moniker] = block
			block.Instances = map[Need]bool{}
		} else {
			for need := range block.Needs {
				stack = append(stack, need)
			}
		}
	}

	// next, determine transitive dependencies.
	for len(stack) > 0 {
		if len(stack) > 10000 {
			fmt.Printf("Template stack size exceeded. You probably have a recursive template somewhere.\n")
			return
		}
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		def := defn[current.Moniker]
		if def == nil {
			fmt.Printf("No such generic name in scope %s (parameters were %s)\n", current.Moniker, current.Args)
			return
		}
		if def.Instances[current] {
			continue
		}
		def.Instances[current] = true // at least, it's in progress
		assignment := map[string]string{}
		args := strings.Split(current.Args, " ")

		if len(def.Generics) != len(args) {
			fmt.Printf("Generic %s has %d parameters but %d were given", def.Moniker, len(def.Generics), len(args))
			return
		}

		for i := range def.Generics {
			assignment[def.Generics[i]] = args[i]
		}

		for need := range def.Needs {
			modified := Need{
				Moniker: need.Moniker,
				Args: replaceGenerics(need.Args, assignment),
			}
			stack = append(stack, modified)
		}
	}

	out, err := os.Create(fileTrg) // TODO
    if err != nil {
        panic(err)
	}
	defer out.Close()

	for blockN := range blocks {
		block := &blocks[blockN]
		if len(block.Generics) > 0 {
			// instances
			for inst := range block.Instances {
				out.Write([]byte(fmt.Sprintf("// instance for %s::[%s]\n", inst.Moniker, inst.Args)))
				asn := map[string]string{}
				for i, arg := range strings.Split(inst.Args, " ") {
					asn[block.Generics[i]] = arg
				}
				for _, line := range block.Lines {
					out.Write([]byte(fmt.Sprintf("%s\n", replaceGenerics(string(line.Source), asn))))
				}
			}
		} else {
			for _, line := range block.Lines {
				out.Write([]byte(fmt.Sprintf("%s\n", line.Source)))
			}
		}
	}
}
