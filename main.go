package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strconv"

	"github.com/pkg/term"
	//"github.com/jroimartin/gocui"
)

type VM struct {
	debug    bool
	file     []byte
	Position int64 `json:"position"`
	Register [8]int64
	Stack    []int64
}

func New(filename string) *VM {
	vm := VM{}
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		panic("You can't very well do the challenge if the damn file won't open!")
	}
	vm.file = file

	if len(os.Args) > 1 && os.Args[1] == "-d" {
		vm.debug = true
	}

	return &vm
}

func (vm *VM) Run() {
	data := make([]int16, 4)
	for {
		// commands will be at most <code> a b c, little endian
		data = vm.read(4, vm.Position)

		// debug
		if vm.debug {
			vm.Printd(data)
		}

		cmd := vm.parse(data[0]) % 32768
		if cmd == 0 { // halt
			fmt.Println("exiting with input 0")
			os.Exit(0)
		} else if cmd == 1 { // set
			vm.Register[vm.modulo(data[1])] = vm.parse(data[2])
			vm.Position += 3
		} else if cmd == 2 { // push
			a := vm.parse(data[1])
			vm.push(a)
			vm.Position += 2
		} else if cmd == 3 { // pop
			a := vm.modulo(data[1])
			vm.Register[a] = vm.pop()
			vm.Position += 2
		} else if cmd == 4 { // eq
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])

			if b == c {
				vm.Register[a] = 1
			} else {
				vm.Register[a] = 0
			}
			vm.Position += 4
		} else if cmd == 5 { // gt
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])

			if b > c {
				vm.Register[a] = 1
			} else {
				vm.Register[a] = 0
			}
			vm.Position += 4
		} else if cmd == 6 { // jmp
			vm.Position = vm.parse(data[1])
		} else if cmd == 7 { // jt
			if vm.parse(data[1]) != 0 {
				vm.Position = vm.parse(data[2])
			} else {
				vm.Position += 3
			}
		} else if cmd == 8 { // jf
			if vm.parse(data[1]) == 0 {
				vm.Position = vm.parse(data[2])
			} else {
				vm.Position += 3
			}
		} else if cmd == 9 { // add
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])
			vm.Register[a] = (b + c) % 32768
			vm.Position += 4
		} else if cmd == 10 { // mult
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])
			vm.Register[a] = (b * c) % 32768
			vm.Position += 4
		} else if cmd == 11 { // mod
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])
			vm.Register[a] = b % c
			vm.Position += 4
		} else if cmd == 12 { // and
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])

			vm.Register[a] = b & c
			vm.Position += 4
		} else if cmd == 13 { // or
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])

			vm.Register[a] = b | c
			vm.Position += 4
		} else if cmd == 14 { // not
			a := vm.modulo(data[1])
			b := vm.parse(data[2])

			// is this a bug/hacky workaround?
			vm.Register[a] = (^b + 32768) % 32768
			vm.Position += 3
		} else if cmd == 15 { // rmem
			a := vm.modulo(data[1])
			b := vm.parse(data[2])

			vm.Register[a] = int64(vm.read(1, b)[0])
			vm.Position += 3
		} else if cmd == 16 { // wmem
			a := vm.parse(data[1])
			b := vm.parse(data[2])

			vm.write(a, b)
			vm.Position += 3
		} else if cmd == 17 { // call
			a := vm.parse(data[1])
			vm.push(vm.Position + 2)
			vm.Position = a
		} else if cmd == 18 { // ret
			if len(vm.Stack) == 0 {
				panic("stack is empty!")
			}
			vm.Position = vm.pop()
		} else if cmd == 19 { // out
			fmt.Print(string(vm.parse(data[1])))
			//fmt.Printf("%s (ascii %v)\n", string(data[1]), data[1])
			vm.Position += 2
		} else if cmd == 20 { // in
			a := int64(data[1])
			char := vm.getch()
			if char == 3 { // ^C
				os.Exit(1)
			} else if char == 4 { // ^D
				vm.debug = !vm.debug
				fmt.Printf("> debug mode %v\n", vm.debug)
				char = vm.getch()
			} else if char == 13 { // CR/NL equivalence
				char = 10
			} else if char == 19 { // ^S for save
				err := ioutil.WriteFile("saved.bin", []byte(vm.file), 0644)
				if err != nil {
					panic(err)
				}
				props, err := json.Marshal(vm)
				if err != nil {
					panic(err)
				}
				err = ioutil.WriteFile("saved.json", props, 0644)
				fmt.Println("> data saved")
				char = vm.getch()
			} else if char == 12 { // ^L for load
				fmt.Println("can't do that right now")
				/*
					vm.file, err = ioutil.ReadFile("saved.bin")
					raw, err := ioutil.ReadFile("saved.json")
					if err != nil {
						panic(err)
					}
					err = json.Unmarshal(raw, &vm)
					if err != nil {
						panic(err)
					}
					fmt.Println("> saved data loaded")
					char = vm.getch()
				*/
			}

			fmt.Print(string(char))
			//fmt.Println(char)
			vm.write(a, char)
			vm.Position += 2
		} else if cmd == 21 { // noop
			vm.Position++
		} else {
			fmt.Printf("I don't know how to handle %s\n", strconv.Itoa(int(cmd)))
			os.Exit(1)
		}
	}
}

// turn little-endian pairs into a slice of ints
func (v *VM) read(n int, position int64) []int16 {
	pos := int(position) * 2
	data := make([]int16, n)

	for i := 0; i < n; i++ {
		err := binary.Read(bytes.NewBuffer([]byte{v.file[pos+(i*2)], v.file[pos+(i*2+1)]}), binary.LittleEndian, &data[i])
		if err != nil {
			panic(err)
		}
	}

	return data
}

func (v *VM) write(pos int64, data int64) error {
	if pos < 0 {
		v.Register[v.modulo(int16(pos))] = data
		return nil
	}
	// write the little-endian pair to the raw bytes
	out := new(bytes.Buffer)
	err := binary.Write(out, binary.LittleEndian, data)
	if err != nil {
		panic(err)
	}
	raw := out.Bytes()
	v.file[pos*2] = raw[0]
	v.file[pos*2+1] = raw[1]

	return nil
}

func (v *VM) parse(r int16) int64 {
	if r < 0 {
		i := v.modulo(r)
		return v.Register[i]
	}

	return int64(r)
}

func (v *VM) modulo(r int16) int16 {
	var i int64
	if r >= 0 {
		panic(fmt.Sprintf("ERROR: %v is not a register\n", r))
	}
	i = 32768 - int64(math.Abs(float64(r)))
	return int16(i)
}

func (v *VM) push(val int64) error {
	v.Stack = append(v.Stack, val)
	return nil
}

func (v *VM) pop() int64 {
	l := len(v.Stack) - 1
	val := v.Stack[l]
	v.Stack = v.Stack[:l]
	return val
}

func (v *VM) getch() int64 {
	t, _ := term.Open("/dev/tty")
	term.RawMode(t)
	bytes := make([]byte, 3)
	_, err := t.Read(bytes)
	t.Restore()
	t.Close()
	if err != nil {
		panic(nil)
	}
	return int64(bytes[0])
}

func (v *VM) Printd(data []int16) {
	fmt.Println("=====")
	fmt.Printf("position: %+v\n", v.Position)
	fmt.Printf("data: %+v\n", data)
	fmt.Printf("register: %+v\n", v.Register)
	fmt.Printf("stack: %+v\n", v.Stack)
}

func main() {
	vm := New("challenge.bin")
	vm.Run()
}
