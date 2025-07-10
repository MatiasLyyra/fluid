package main

import (
	"fmt"
	"math/rand"
	"runtime"
	"slices"
	"unsafe"

	"github.com/MatiasLyyra/fluid/gsort"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/go-gl/gl/v4.3-core/gl"
)

const capacity = 1024

func main() {
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.InitWindow(600, 600, "Prefix sum")
	if err := gl.Init(); err != nil {
		panic(fmt.Sprintf("glInit should succeed: %v", err))
	}

	testCases := [1 << 20]struct {
		desc string
		size int
	}{}
	for i := range testCases {
		testCases[i].desc = fmt.Sprintf("Sort data size %d", i+1)
		testCases[i].size = i + 1<<18
	}
	r := rand.New(rand.NewSource(0))
	gs := gsort.New(1 << 20)
	defer gs.Free()
	sb := rl.LoadShaderBuffer((1<<20)*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)
	defer gs.Free()
	for _, tC := range testCases {
		var p runtime.Pinner
		data := make([]uint32, tC.size)
		data2 := make([]uint32, tC.size)
		for i := range data {
			data[i] = r.Uint32()
			data2[i] = data[i]
		}
		p.Pin(unsafe.SliceData(data2))
		rl.UpdateShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(data2)), uint32(len(data2))*4, 0)
		p.Unpin()

		gs.Sort(sb, tC.size)

		p.Pin(unsafe.SliceData(data2))
		rl.ReadShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(data2)), uint32(len(data2))*4, 0)
		p.Unpin()

		slices.Sort(data)
		for i := range data {
			val1 := data[i]
			val2 := data2[i]
			if val1 != val2 {
				panic(fmt.Sprintf("values at differ idx %d: %d != %d", i, val1, val2))
			}
		}
	}

	// pfs := gsort.New(capacity)

	// data := [capacity]uint32{}
	// data2 := [capacity]uint32{}
	// for i := range data {
	// 	data[i] = rand.Uint32()
	// }

	// var p runtime.Pinner
	// buf := rl.LoadShaderBuffer(uint32(capacity*4), nil, rl.DynamicCopy)
	// p.Pin(unsafe.SliceData(data[:]))
	// rl.UpdateShaderBuffer(buf, unsafe.Pointer(unsafe.SliceData(data[:])), uint32(len(data))*4, 0)
	// p.Unpin()

	// now := time.Now()
	// pfs.Sort(buf, len(data))
	// log.Printf("GPU Sorting took %d us", time.Since(now).Nanoseconds()/1000)

	// p.Pin(unsafe.SliceData(data2[:]))
	// rl.ReadShaderBuffer(buf, unsafe.Pointer(unsafe.SliceData(data2[:])), uint32(len(data2))*4, 0)
	// p.Unpin()

	// now = time.Now()
	// slices.Sort(data[:])
	// log.Printf("CPU slices.Sort took %d us", time.Since(now).Nanoseconds()/1000)

	// for i := range data {
	// 	val1 := data[i]
	// 	val2 := data2[i]
	// 	if val1 != val2 {
	// 		panic(fmt.Sprintf("values at differ idx %d: %d != %d", i, val1, val2))
	// 	}
	// }
}
