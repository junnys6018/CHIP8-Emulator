// tinygo build -o web/chip8.wasm -target wasm ./chip8.go
package main

import (
	"fmt"
	"math/rand"
)

type Chip8 struct {
	memory        [4096]uint8
	V             [16]uint8
	I             uint16
	delayRegister uint8
	soundRegister uint8
	PC            uint16
	SP            uint8
	stack         [16]uint16
	screen        [32]uint64
	keys          uint16
	paused        bool
	timer         float64 // milliseconds
}

func (chip8 *Chip8) SetSprite(index, v0, v1, v2, v3, v4 uint8) {
	chip8.memory[index*5+0] = v0
	chip8.memory[index*5+1] = v1
	chip8.memory[index*5+2] = v2
	chip8.memory[index*5+3] = v3
	chip8.memory[index*5+4] = v4
}

var chip8 Chip8

//export InitChip8
func InitChip8(rom []uint8) {
	chip8 = Chip8{}

	chip8.PC = 0x200
	chip8.timer = 1000.0 / 60.0
	copy(chip8.memory[0x200:], rom)

	// Load font
	chip8.SetSprite(0x0, 0xF0, 0x90, 0x90, 0x90, 0xF0)
	chip8.SetSprite(0x1, 0x20, 0x60, 0x20, 0x20, 0x70)
	chip8.SetSprite(0x2, 0xF0, 0x10, 0xF0, 0x80, 0xF0)
	chip8.SetSprite(0x3, 0xF0, 0x10, 0xF0, 0x10, 0xF0)
	chip8.SetSprite(0x4, 0x90, 0x90, 0xF0, 0x10, 0x10)
	chip8.SetSprite(0x5, 0xF0, 0x80, 0xF0, 0x10, 0xF0)
	chip8.SetSprite(0x6, 0xF0, 0x80, 0xF0, 0x90, 0xF0)
	chip8.SetSprite(0x7, 0xF0, 0x10, 0x20, 0x40, 0x40)
	chip8.SetSprite(0x8, 0xF0, 0x90, 0xF0, 0x90, 0xF0)
	chip8.SetSprite(0x9, 0xF0, 0x90, 0xF0, 0x10, 0xF0)
	chip8.SetSprite(0xA, 0xF0, 0x90, 0xF0, 0x90, 0x90)
	chip8.SetSprite(0xB, 0xE0, 0x90, 0xE0, 0x90, 0xE0)
	chip8.SetSprite(0xC, 0xF0, 0x80, 0x80, 0x80, 0xF0)
	chip8.SetSprite(0xD, 0xE0, 0x90, 0x90, 0x90, 0xE0)
	chip8.SetSprite(0xE, 0xF0, 0x80, 0xF0, 0x80, 0xF0)
	chip8.SetSprite(0xF, 0xF0, 0x80, 0xF0, 0x80, 0x80)
}

func (chip8 *Chip8) ReadInstruction(addr uint16) uint16 {
	return uint16(chip8.memory[addr])<<8 | uint16(chip8.memory[addr+1])
}

func (chip8 *Chip8) IllegalInstruction() {
	fmt.Printf("Illegal Opcode called: %x\n", chip8.ReadInstruction(chip8.PC-2))
}

// called at 500HZ
//export Step
func Step() {
	chip8.timer -= 2.0
	if chip8.timer < 0.0 {
		if chip8.delayRegister > 0 {
			chip8.delayRegister--
		}
		if chip8.soundRegister > 0 {
			chip8.soundRegister--
		}
		chip8.timer += 1000.0 / 60.0
	}

	if chip8.paused {
		return
	}

	instruction := chip8.ReadInstruction(chip8.PC)
	chip8.PC += 2

	opcode := instruction >> 12
	nnn := instruction & 0x0FFF
	n := instruction & 0x000F
	x := (instruction & 0x0F00) >> 8
	y := (instruction & 0x00F0) >> 4
	kk := uint8(instruction & 0x00FF)

	switch opcode {
	case 0x0:
		if nnn == 0x0E0 /* CLS */ {
			for i := range chip8.screen {
				chip8.screen[i] = 0
			}
		} else if nnn == 0x0EE /* RET */ {
			chip8.SP = (chip8.SP - 1) & 0xF
			chip8.PC = chip8.stack[chip8.SP]
		} else {
			chip8.IllegalInstruction()
		}
	case 0x1 /* JP addr */ :
		chip8.PC = nnn
	case 0x2 /* CALL addr */ :
		chip8.stack[chip8.SP] = chip8.PC
		chip8.SP = (chip8.SP + 1) & 0xF
		chip8.PC = nnn
	case 0x3 /* SE Vx, byte */ :
		if chip8.V[x] == kk {
			chip8.PC += 2
		}
	case 0x4 /* SNE Vx, byte */ :
		if chip8.V[x] != kk {
			chip8.PC += 2
		}
	case 0x5 /* SE Vx, Vy */ :
		if n == 0 {
			if chip8.V[x] == chip8.V[y] {
				chip8.PC += 2
			}
		} else {
			chip8.IllegalInstruction()
		}
	case 0x6 /* LD Vx, byte */ :
		chip8.V[x] = kk
	case 0x7 /* ADD Vx, byte */ :
		chip8.V[x] = chip8.V[x] + kk
	case 0x8:
		switch n {
		case 0x0 /* LD Vx, Vy */ :
			chip8.V[x] = chip8.V[y]
		case 0x1 /* OR Vx, Vy */ :
			chip8.V[x] = chip8.V[x] | chip8.V[y]
		case 0x2 /* AND Vx, Vy */ :
			chip8.V[x] = chip8.V[x] & chip8.V[y]
		case 0x3 /* XOR Vx, Vy */ :
			chip8.V[x] = chip8.V[x] ^ chip8.V[y]
		case 0x4 /* ADD Vx, Vy */ :
			result := uint16(chip8.V[x]) + uint16(chip8.V[y])
			if result > 255 {
				chip8.V[0xF] = 1
			} else {
				chip8.V[0xF] = 0
			}
			chip8.V[x] = uint8(result & 0x00FF)
		case 0x5 /* SUB Vx, Vy */ :
			if chip8.V[x] > chip8.V[y] {
				chip8.V[0xF] = 1
			} else {
				chip8.V[0xF] = 0
			}
			chip8.V[x] = chip8.V[x] - chip8.V[y]
		case 0x6 /* SHR Vx {, Vy} */ :
			chip8.V[0xF] = chip8.V[x] & 0x01
			chip8.V[x] = chip8.V[x] >> 1
		case 0x7 /* SUBN Vx, Vy */ :
			if chip8.V[y] > chip8.V[x] {
				chip8.V[0xF] = 1
			} else {
				chip8.V[0xF] = 0
			}
			chip8.V[x] = chip8.V[y] - chip8.V[x]
		case 0xE /* SHL Vx {, Vy} */ :
			chip8.V[0xF] = chip8.V[x] >> 7
			chip8.V[x] = chip8.V[x] << 1
		default:
			chip8.IllegalInstruction()
		}
	case 0x9 /* SNE Vx, Vy */ :
		if n == 0 {
			if chip8.V[x] != chip8.V[y] {
				chip8.PC += 2
			}
		} else {
			chip8.IllegalInstruction()
		}
	case 0xA /* LD I, addr */ :
		chip8.I = nnn
	case 0xB /* JP V0, addr */ :
		chip8.PC = nnn + uint16(chip8.V[0]) // TODO: what to do if overflows 12 bits
	case 0xC /* RND Vx, byte */ :
		bytes := uint8(rand.Uint32())
		chip8.V[x] = bytes & kk
	case 0xD /* DRW Vx, Vy, nibble */ :
		Vx := chip8.V[x] % 64
		Vy := chip8.V[y] % 32

		chip8.V[0xF] = 0
		for i := uint16(0); i < n; i++ {
			pixels := uint64(chip8.memory[chip8.I+i])
			if Vx > 56 {
				pixels = (pixels >> (56 - Vx)) | (pixels << Vx)
			} else {
				pixels = pixels << (56 - Vx)
			}
			if chip8.screen[(uint16(Vy)+i)%32]&pixels != 0 {
				chip8.V[0xF] = 1
			}
			chip8.screen[(uint16(Vy)+i)%32] ^= pixels
		}
	case 0xE:
		if kk == 0x9E /* SKP Vx */ {
			if chip8.keys&(1<<chip8.V[x]) != 0 { // TODO: what if Vx >= 16?
				chip8.PC += 2
			}
		} else if kk == 0xA1 /* SKNP Vx */ {
			if chip8.keys&(1<<chip8.V[x]) == 0 { // TODO: what if Vx >= 16?
				chip8.PC += 2
			}
		} else {
			chip8.IllegalInstruction()
		}
	case 0xF:
		switch kk {
		case 0x07 /* LD Vx, DT */ :
			chip8.V[x] = chip8.delayRegister
		case 0x0A /* LD Vx, K */ :
			if chip8.keys != 0 {
				for i := 0; i < 16; i++ {
					if chip8.keys&(1<<i) != 0 {
						chip8.V[x] = uint8(i)
						break
					}
				}
			} else {
				chip8.paused = true
			}
		case 0x15 /* LD DT, Vx */ :
			chip8.delayRegister = chip8.V[x]
		case 0x18 /* LD ST, Vx */ :
			chip8.soundRegister = chip8.V[x]
		case 0x1E /* ADD I, Vx */ :
			chip8.I = chip8.I + uint16(chip8.V[x])
		case 0x29 /* LD F, Vx */ :
			// TODO: what happens if Vx >= 16?
			chip8.I = uint16(chip8.V[x]) * 5
		case 0x33 /* LD B, Vx */ :
			chip8.memory[chip8.I] = (chip8.V[x] / 100) % 10
			chip8.memory[chip8.I+1] = (chip8.V[x] / 10) % 10
			chip8.memory[chip8.I+2] = chip8.V[x] % 10
		case 0x55 /* LD [I], Vx */ :
			copy(chip8.memory[chip8.I:], chip8.V[:x+1])
			chip8.I += x + 1
		case 0x65 /* LD Vx, [I] */ :
			copy(chip8.V[:x+1], chip8.memory[chip8.I:])
			chip8.I += x + 1
		default:
			chip8.IllegalInstruction()
		}
	}
}

var image = [64 * 32]uint32{}

//export GetFrame
func GetFrame() *uint32 {
	for y := 0; y < 32; y++ {
		row := chip8.screen[y]
		for x := 0; x < 64; x++ {
			pixel := row&(1<<(63-x)) != 0
			if pixel {
				image[y*64+x] = 0xFFFFFFFF
			} else {
				image[y*64+x] = 0xFF000000
			}
		}
	}
	return &image[0]
}

//export SetKeys
func SetKeys(keys uint16) {
	chip8.keys = keys

	if keys != 0 && chip8.paused {
		x := chip8.memory[chip8.PC-2] & 0x0F
		for i := 0; i < 16; i++ {
			if keys&(1<<i) != 0 {
				chip8.V[x] = uint8(i)
				break
			}
		}
	}
}

func main() {
	fmt.Println("Chip8 WebAssembly Instance Started")
}
