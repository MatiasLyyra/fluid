// Package gsort provides GPU accelerated stable sorting on uint32 keys.
//
// GPU acceleration relies on OpenGL compute shaders and requires O(n) storage for sorting.
//
// Sorting algorithm uses radix sort as described in paper "Fast 4-way parallel radix sorting on GPUs" [1], with slight modifications
// and simplifications. Sorting also relies on calculating prefix sums for arbitrarily large data. For prefix sum calculations,
// algorithm described by NVIDIA's GPU Gems 3 [2] is used.
//
// References:
//
//  1. Ha, Linh & Krüger, Jens & Silva, Claudio. (2009). Fast 4-way parallel radix sorting on GPUs. Comput. Graph. Forum. 28. 2368-2378. 10.1111/j.1467-8659.2009.01542.x.
//  2. https://developer.nvidia.com/gpugems/gpugems3/part-vi-gpu-computing/chapter-39-parallel-prefix-sum-scan-cuda
package gsort

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"strings"
	"text/template"
	"unsafe"

	gl "github.com/go-gl/gl/v4.3-core/gl"

	rl "github.com/gen2brain/raylib-go/raylib"
)

//go:embed shaders/common.glsl
var commonShader string

//go:embed shaders/radix_scan.glsl
var radixScanShader string

//go:embed shaders/prefix_sum.glsl
var prefixSumShader string

//go:embed shaders/add_block.glsl
var addBlockShader string

//go:embed shaders/scatter.glsl
var scatterShader string

var shaderTemplate *template.Template

func init() {
	shaderTemplate = template.Must(template.New("shaders/common.glsl").Parse(commonShader))
	shaderTemplate = template.Must(shaderTemplate.New("shaders/radix_scan.glsl").Parse(radixScanShader))
	shaderTemplate = template.Must(shaderTemplate.New("shaders/prefix_sum.glsl").Parse(prefixSumShader))
	shaderTemplate = template.Must(shaderTemplate.New("shaders/add_block.glsl").Parse(addBlockShader))
	shaderTemplate = template.Must(shaderTemplate.New("shaders/scatter.glsl").Parse(scatterShader))
}

type RadixSort struct {
	shaderRadixScan                   uint32
	shaderRadixScanUniformInput       int32
	shaderRadixScanUniformWorkGroups  int32
	shaderRadixScanUniformOffset      int32
	shaderPrefixSum                   uint32
	shaderPrefixSumUniformInput       int32
	shaderPrefixSumUniformInputOffset int32
	shaderPrefixSumUniformSumOffset   int32
	shaderAddBlock                    uint32
	shaderAddBlockUniformInputOffset  int32
	shaderAddBlockUniformSumOffset    int32
	shaderScatter                     uint32
	shaderScatterUniformInput         int32
	shaderScatterUniformOffset        int32
	shaderScatterUniformWorkGroups    int32
	inputBuffer                       uint32
	localPrefixBuffer                 uint32
	blockSumBuffer                    uint32
	valuesPerWorkGroup                uint32
}

type shaderSettings struct {
	WorkGroupItems uint32
	WorkGroupSize  uint32
}

func loadShader(name string, valuesPerWorkGroup uint32) uint32 {
	var buf bytes.Buffer
	if err := shaderTemplate.ExecuteTemplate(&buf, name, shaderSettings{
		WorkGroupItems: valuesPerWorkGroup,
		WorkGroupSize:  valuesPerWorkGroup / 2,
	}); err != nil {
		panic(fmt.Sprintf("failed to parse embedded shader %v template: %v", name, err))
	}
	shader := rl.CompileShader(buf.String(), rl.ComputeShader)
	shaderProg := rl.LoadComputeShaderProgram(shader)
	if shaderProg == 0 {
		panic(fmt.Sprintf("invalid shader program %v", name))
	}
	return shaderProg
}

type SortSettings struct {
	Capacity           uint32
	ValuesPerWorkGroup uint32
}

func NewSettings(cap uint32) SortSettings {
	return SortSettings{
		Capacity: cap,
	}
}

func (settings SortSettings) WithValuesPerWorkGroup(count uint32) SortSettings {
	settings.ValuesPerWorkGroup = count
	return settings
}

func (settings SortSettings) getValuesPerWorkGroup() uint32 {
	if settings.ValuesPerWorkGroup == 0 {
		return 256
	}
	return nextPow2(settings.ValuesPerWorkGroup)
}
func (settings SortSettings) getCapacity() uint32 {
	if settings.Capacity == 0 {
		panic("SortSettings.Capacity must be defined")
	}
	return multipleOf(settings.Capacity, settings.getValuesPerWorkGroup())
}

func New(settings SortSettings) *RadixSort {
	valuesPerWorkGroup := settings.getValuesPerWorkGroup()
	capacity := settings.getCapacity()
	log.Printf("capacity: %d", capacity)
	radixScanProg := loadShader("shaders/radix_scan.glsl", valuesPerWorkGroup)
	shaderRadixScanUniformInput := rl.GetLocationUniform(radixScanProg, "n_input")
	shaderRadixScanUniformWorkGroups := rl.GetLocationUniform(radixScanProg, "n_workgroups")
	shaderRadixScanUniformOffset := rl.GetLocationUniform(radixScanProg, "offset")
	prefixSumProg := loadShader("shaders/prefix_sum.glsl", valuesPerWorkGroup)
	shaderPrefixSumUniformInput := rl.GetLocationUniform(prefixSumProg, "n_input")
	shaderPrefixSumUniformInputOffset := rl.GetLocationUniform(prefixSumProg, "input_offset")
	shaderPrefixSumUniformSumOffset := rl.GetLocationUniform(prefixSumProg, "sum_offset")
	addBlockProg := loadShader("shaders/add_block.glsl", valuesPerWorkGroup)
	shaderAddBlockUniformInputOffset := rl.GetLocationUniform(addBlockProg, "input_offset")
	shaderAddBlockUniformSumOffset := rl.GetLocationUniform(addBlockProg, "sum_offset")
	scatterProg := loadShader("shaders/scatter.glsl", valuesPerWorkGroup)
	shaderScatterUniformInput := rl.GetLocationUniform(scatterProg, "n_input")
	shaderScatterUniformWorkGroups := rl.GetLocationUniform(scatterProg, "n_workgroups")
	shaderScatterUniformOffset := rl.GetLocationUniform(scatterProg, "offset")

	input := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	localPrefix := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	blockSum := rl.LoadShaderBuffer(max(nextPow2(capacity)/valuesPerWorkGroup, valuesPerWorkGroup)*4*2*4, nil, rl.DynamicCopy)

	return &RadixSort{
		shaderRadixScan:                   radixScanProg,
		shaderRadixScanUniformInput:       shaderRadixScanUniformInput,
		shaderRadixScanUniformWorkGroups:  shaderRadixScanUniformWorkGroups,
		shaderRadixScanUniformOffset:      shaderRadixScanUniformOffset,
		shaderPrefixSum:                   prefixSumProg,
		shaderPrefixSumUniformInput:       shaderPrefixSumUniformInput,
		shaderPrefixSumUniformInputOffset: shaderPrefixSumUniformInputOffset,
		shaderPrefixSumUniformSumOffset:   shaderPrefixSumUniformSumOffset,
		shaderAddBlock:                    addBlockProg,
		shaderAddBlockUniformInputOffset:  shaderAddBlockUniformInputOffset,
		shaderAddBlockUniformSumOffset:    shaderAddBlockUniformSumOffset,
		shaderScatter:                     scatterProg,
		shaderScatterUniformInput:         shaderScatterUniformInput,
		shaderScatterUniformOffset:        shaderScatterUniformOffset,
		shaderScatterUniformWorkGroups:    shaderScatterUniformWorkGroups,
		inputBuffer:                       input,
		localPrefixBuffer:                 localPrefix,
		blockSumBuffer:                    blockSum,
		valuesPerWorkGroup:                valuesPerWorkGroup,
	}
}

func (pfs *RadixSort) Sort(input_buf uint32, length int) {
	dataLen := uint32(length)
	dataLenMultiple := multipleOf(dataLen, pfs.valuesPerWorkGroup)
	workGroups := dataLenMultiple / pfs.valuesPerWorkGroup

	var offset uint32
	buffer1 := input_buf
	buffer2 := pfs.inputBuffer
	for offset = 0; offset < 32; offset += 2 {
		// Scan the input and build local prefix sum for each block, and build block sum 4*workgroups large.
		// Block sum contains count of each possible digit 0-3 layed out as
		// [
		//   [zero_count_for_block0, 	zero_count_for_block1, 	...,  zero_count_for_blockN-1 ]
		//   [one_count_for_block0,  	one_count_for_block1,  	...,  one_count_for_blockN-1  ]
		//   [two_count_for_block0,  	two_count_for_block1,  	...,  two_count_for_blockN-1  ]
		//   [three_count_for_block0,	three_count_for_block1,	...,  three_count_for_blockN-1]
		// ]
		rl.EnableShader(pfs.shaderRadixScan)
		rl.SetUniform(pfs.shaderRadixScanUniformInput, uniformValues(dataLen), int32(rl.ShaderUniformUint))
		rl.SetUniform(pfs.shaderRadixScanUniformWorkGroups, uniformValues(workGroups), int32(rl.ShaderUniformUint))
		rl.SetUniform(pfs.shaderRadixScanUniformOffset, uniformValues(offset), int32(rl.ShaderUniformUint))
		rl.BindShaderBuffer(buffer1, 1)
		rl.BindShaderBuffer(pfs.localPrefixBuffer, 2)
		rl.BindShaderBuffer(pfs.blockSumBuffer, 3)
		rl.ComputeShaderDispatch(workGroups, 1, 1)
		rl.DisableShader()
		gl.MemoryBarrier(gl.SHADER_STORAGE_BARRIER_BIT)

		// Perform prefix sum scan of the block sum memory.
		// This gives us indices for each digit globally two scatter on the next stage.
		pfs.prefixSum(workGroups)
		// printBuffer("BlockSumBuffer", pfs.BlockSumBuffer, 512, 0, 256)

		// Scatter input to the output buffer based on local prefix sum (ordering between same digits within a block)
		// and prefix summed block sum.
		rl.EnableShader(pfs.shaderScatter)
		rl.SetUniform(pfs.shaderScatterUniformInput, uniformValues(dataLen), int32(rl.ShaderUniformUint))
		rl.SetUniform(pfs.shaderScatterUniformWorkGroups, uniformValues(workGroups), int32(rl.ShaderUniformUint))
		rl.SetUniform(pfs.shaderScatterUniformOffset, uniformValues(offset), int32(rl.ShaderUniformUint))
		rl.BindShaderBuffer(buffer1, 1)
		rl.BindShaderBuffer(buffer2, 2)
		rl.BindShaderBuffer(pfs.localPrefixBuffer, 3)
		rl.BindShaderBuffer(pfs.blockSumBuffer, 4)
		rl.ComputeShaderDispatch(workGroups, 1, 1)
		rl.DisableShader()
		gl.MemoryBarrier(gl.SHADER_STORAGE_BARRIER_BIT)
		buffer1, buffer2 = buffer2, buffer1
	}
}

func multipleOf(x, multiple uint32) uint32 {
	if mod := x % multiple; mod > 0 {
		x += multiple - mod
	}
	return x
}

func (pfs *RadixSort) prefixSum(workGroups uint32) {
	initialSize := nextPow2(multipleOf(workGroups*4, pfs.valuesPerWorkGroup))
	sumBufferSize := initialSize
	sumBufferOffset := uint32(0)
	sumBufferSumOffset := sumBufferSize
	inputDataSize := workGroups * 4

	for sumBufferSize >= pfs.valuesPerWorkGroup {
		pfs.prefixSumIteration(sumBufferSize, sumBufferOffset, sumBufferSumOffset, inputDataSize)
		sumBufferOffset += sumBufferSize
		sumBufferSumOffset = sumBufferOffset + (sumBufferSize / pfs.valuesPerWorkGroup)
		sumBufferSize /= pfs.valuesPerWorkGroup
		inputDataSize = sumBufferSize
	}
	if initialSize <= pfs.valuesPerWorkGroup {
		return
	}
	pfs.prefixSumIteration(sumBufferSize, sumBufferOffset, sumBufferSumOffset, sumBufferSize)
	sumBufferSize *= pfs.valuesPerWorkGroup
	sumBufferOffset -= sumBufferSize
	sumBufferSumOffset = sumBufferOffset + (sumBufferSize)

	for sumBufferSize <= initialSize {
		pfs.addBlockIteration(sumBufferSize, sumBufferOffset, sumBufferSumOffset)
		sumBufferSize *= pfs.valuesPerWorkGroup
		sumBufferOffset -= sumBufferSize
		sumBufferSumOffset = sumBufferOffset + (sumBufferSize)
	}
}

func (pfs *RadixSort) prefixSumIteration(sumBufferSize uint32, sumBufferOffset uint32, sumBufferSumOffset uint32, dataLenth uint32) {
	rl.EnableShader(pfs.shaderPrefixSum)
	rl.SetUniform(pfs.shaderPrefixSumUniformInput, uniformValues(dataLenth), int32(rl.ShaderUniformUint))
	rl.SetUniform(pfs.shaderPrefixSumUniformInputOffset, uniformValues(sumBufferOffset), int32(rl.ShaderUniformUint))
	rl.SetUniform(pfs.shaderPrefixSumUniformSumOffset, uniformValues(sumBufferSumOffset), int32(rl.ShaderUniformUint))
	rl.BindShaderBuffer(pfs.blockSumBuffer, 1)
	rl.ComputeShaderDispatch(max(sumBufferSize/pfs.valuesPerWorkGroup, 1), 1, 1)
	rl.DisableShader()
	gl.MemoryBarrier(gl.SHADER_STORAGE_BARRIER_BIT)
}

func (pfs *RadixSort) addBlockIteration(sumBufferSize uint32, sumBufferOffset uint32, sumBufferSumOffset uint32) {
	rl.EnableShader(pfs.shaderAddBlock)
	rl.SetUniform(pfs.shaderAddBlockUniformInputOffset, uniformValues(sumBufferOffset), int32(rl.ShaderUniformUint))
	rl.SetUniform(pfs.shaderAddBlockUniformSumOffset, uniformValues(sumBufferSumOffset), int32(rl.ShaderUniformUint))
	rl.BindShaderBuffer(pfs.blockSumBuffer, 1)
	rl.ComputeShaderDispatch(max(sumBufferSize/pfs.valuesPerWorkGroup, 1), 1, 1)
	rl.DisableShader()
	gl.MemoryBarrier(gl.SHADER_STORAGE_BARRIER_BIT)
}

func (pfs *RadixSort) Free() {
	rl.UnloadShaderProgram(pfs.shaderRadixScan)
	rl.UnloadShaderProgram(pfs.shaderPrefixSum)
	rl.UnloadShaderProgram(pfs.shaderAddBlock)
	rl.UnloadShaderProgram(pfs.shaderScatter)

	rl.UnloadShaderBuffer(pfs.inputBuffer)
	rl.UnloadShaderBuffer(pfs.blockSumBuffer)
	rl.UnloadShaderBuffer(pfs.localPrefixBuffer)
}

func nextPow2(v uint32) uint32 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v++
	return v
}

func uniformValues[T any](values ...T) []float32 {
	ret := make([]float32, len(values))
	for i, v := range values {
		ret[i] = *(*float32)(unsafe.Pointer(&v))
	}
	return ret
}

func printBuffer(name string, buf uint32, length uint32, offset uint32, split int) {
	temp := make([]uint32, length)
	if split <= 0 {
		split = int(length)
	}
	rl.ReadShaderBuffer(buf, unsafe.Pointer(unsafe.SliceData(temp)), length*4, offset*4)
	log.Printf("Buffer %v\n%v", name, splitBuffer(temp, split))
}

func splitBuffer(buf []uint32, split int) string {
	var sb strings.Builder
	for i := 0; i < len(buf); i += split {
		end := min(i+split, len(buf))
		sb.WriteString(fmt.Sprintf("%+v\n", buf[i:end]))
	}
	return sb.String()
}
