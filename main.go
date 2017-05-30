package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"strconv"
)

type syn struct {
	file     []byte
	position int64
	register [8]int64
	stack    []int64
}

// turn little-endian pairs into a slice of ints
func (s *syn) read(n int, position int64) []int16 {
	pos := int(position) * 2
	data := make([]int16, n)

	for i := 0; i < n; i++ {
		err := binary.Read(bytes.NewBuffer([]byte{s.file[pos+(i*2)], s.file[pos+(i*2+1)]}), binary.LittleEndian, &data[i])
		if err != nil {
			panic(err)
		}
	}

	return data
}

func (s *syn) parse(r int16) int64 {
	if r < 0 {
		i := s.modulo(r)
		return s.register[i]
	}

	return int64(r)
}

func (s *syn) modulo(r int16) int16 {
	var i int64
	if r >= 0 {
		panic(fmt.Sprintf("ERROR: %s is not a register\n", r))
	}
	i = 32768 - int64(math.Abs(float64(r)))
	return int16(i)
}

func (s *syn) push(val int64) error {
	s.stack = append(s.stack, val)
	return nil
}

func (s *syn) pop() int64 {
	l := len(s.stack) - 1
	val := s.stack[l]
	s.stack = s.stack[:l]
	return val
}

func (s *syn) debug(data []int16) {
	fmt.Println("=====")
	fmt.Printf("position: %+v\n", s.position)
	fmt.Printf("data: %+v\n", data)
	fmt.Printf("register: %+v\n", s.register)
	fmt.Printf("stack: %+v\n", s.stack)
}

func main() {
	var err error
	vm := syn{}
	vm.file, err = ioutil.ReadFile("challenge.bin")
	if err != nil {
		panic("You can't very well do the challenge if the damn file won't open!")
	}

	data := make([]int16, 4)

	for {
		// commands will be at most <code> a b c, little endian
		data = vm.read(4, vm.position)

		// debug
		if len(os.Args) > 1 && os.Args[1] == "-d" {
			vm.debug(data)
		}

		cmd := vm.parse(data[0]) % 32768
		if cmd == 0 { // halt
			fmt.Println("exiting with input 0")
			os.Exit(0)
		} else if cmd == 1 { // set
			vm.register[vm.modulo(data[1])] = vm.parse(data[2])
			vm.position += 3
		} else if cmd == 2 { // push
			a := vm.parse(data[1])
			vm.push(a)
			vm.position += 2
		} else if cmd == 3 { // pop
			a := vm.modulo(data[1])
			vm.register[a] = vm.pop()
			vm.position += 2
		} else if cmd == 4 { // eq
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])

			if b == c {
				vm.register[a] = 1
			} else {
				vm.register[a] = 0
			}
			vm.position += 4
		} else if cmd == 5 { // gt
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])

			if b > c {
				vm.register[a] = 1
			} else {
				vm.register[a] = 0
			}
			vm.position += 4
		} else if cmd == 6 { // jmp
			vm.position = vm.parse(data[1])
		} else if cmd == 7 { // jt
			if vm.parse(data[1]) != 0 {
				vm.position = vm.parse(data[2])
			} else {
				vm.position += 3
			}
		} else if cmd == 8 { // jf
			if vm.parse(data[1]) == 0 {
				vm.position = vm.parse(data[2])
			} else {
				vm.position += 3
			}
		} else if cmd == 9 { // add
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])
			vm.register[a] = (b + c) % 32768
			vm.position += 4
		} else if cmd == 10 { // mult
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])
			vm.register[a] = (b * c) % 32768
			vm.position += 4
		} else if cmd == 11 { // mod
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])
			vm.register[a] = b % c
			vm.position += 4
		} else if cmd == 12 { // and
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])

			vm.register[a] = b & c
			vm.position += 4
		} else if cmd == 13 { // or
			a := vm.modulo(data[1])
			b := vm.parse(data[2])
			c := vm.parse(data[3])

			vm.register[a] = b | c
			vm.position += 4
		} else if cmd == 14 { // not
			a := vm.modulo(data[1])
			b := vm.parse(data[2])

			// is this a bug/hacky workaround?
			vm.register[a] = (^b + 32768) % 32768
			vm.position += 3
		} else if cmd == 15 { // rmem
			a := vm.modulo(data[1])
			b := vm.parse(data[2])

			vm.register[a] = int64(vm.read(1, b)[0])
			vm.position += 3
		} else if cmd == 16 { // wmem
			// still an issue here?
			a := vm.parse(data[1])
			b := vm.parse(data[2])

			// write the little-endian pair to the raw bytes
			out := new(bytes.Buffer)
			err = binary.Write(out, binary.LittleEndian, b)
			if err != nil {
				panic(err)
			}
			raw := out.Bytes()
			vm.file[a*2] = raw[0]
			vm.file[a*2+1] = raw[1]

			vm.position += 3
		} else if cmd == 17 { // call
			a := vm.parse(data[1])
			vm.push(vm.position + 2)
			vm.position = a
		} else if cmd == 18 { // ret
			if len(vm.stack) == 0 {
				panic("stack is empty!")
			}
			vm.position = vm.pop()
		} else if cmd == 19 { // out
			fmt.Print(string(data[1]))
			//fmt.Printf("%s (ascii %v)\n", string(data[1]), data[1])
			vm.position += 2
		} else if cmd == 20 { // TBD
			a := vm.parse(data[1])

		} else if cmd == 21 { // noop
			vm.position++
		} else {
			fmt.Printf("I don't know how to handle %s\n", strconv.Itoa(int(cmd)))
			os.Exit(1)
		}
	}
}
