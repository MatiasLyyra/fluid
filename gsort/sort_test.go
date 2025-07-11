//go:build opengl43

package gsort_test

import (
	"iter"
	"math/rand"
	"runtime"
	"slices"
	"testing"
	"unsafe"

	"github.com/MatiasLyyra/fluid/gsort"
	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/go-gl/gl/v4.3-core/gl"
	"github.com/stretchr/testify/assert"
)

func TestSortSamllRandomValue(t *testing.T) {
	const capacity = 8192
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	for td := range generateTestData(initializeSameValue, r, linear(capacity)) {
		gpuSort(gs, td.actual, sb)
		assert.Equal(t, td.actual, td.expected, "Sorted arrays should be equal")

	}
}
func TestSortRandomSmallWorkGroup(t *testing.T) {
	const capacity = 1024
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity).WithValuesPerWorkGroup(2))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	for td := range generateTestData(initializeSameValue, r, linear(capacity)) {
		gpuSort(gs, td.actual, sb)
		assert.Equal(t, td.actual, td.expected, "Sorted arrays should be equal")
	}
}
func TestSortSmallSameValue(t *testing.T) {
	const capacity = 8192
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	for td := range generateTestData(initializeSameValue, r, linear(capacity)) {
		gpuSort(gs, td.actual, sb)
		assert.Equal(t, td.actual, td.expected, "Sorted arrays should be equal")

	}
}

func TestSortLarge(t *testing.T) {
	const capacity = 256 * 256 * 256
	initialize(t)

	r := rand.New(rand.NewSource(0))
	expected, actual := initializeRandomTestValues(capacity, r)
	gs := gsort.New(gsort.NewSettings(capacity))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	gpuSort(gs, actual, sb)
	assert.Equal(t, expected, actual, "Sorted arrays should be equal")
}

func gpuSort(gs *gsort.RadixSort, data []uint32, sb uint32) {
	var p runtime.Pinner
	p.Pin(unsafe.SliceData(data))
	rl.UpdateShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(data)), uint32(len(data))*4, 0)
	p.Unpin()

	gs.Sort(sb, len(data))

	p.Pin(unsafe.SliceData(data))
	rl.ReadShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(data)), uint32(len(data))*4, 0)
	p.Unpin()
}

type testData struct {
	actual   []uint32
	expected []uint32
}

type generateValuesFunc func(cap int, r *rand.Rand) ([]uint32, []uint32)

func generateTestData(fn generateValuesFunc, r *rand.Rand, sizes iter.Seq[int]) iter.Seq[testData] {
	return func(yield func(testData) bool) {
		for v := range sizes {
			actual, expected := fn(v, r)
			if !yield(testData{actual: actual, expected: expected}) {
				break
			}
		}
	}
}

func initializeRandomTestValues(capacity int, r *rand.Rand) ([]uint32, []uint32) {
	expected := make([]uint32, capacity)
	actual := make([]uint32, capacity)

	for i := range expected {
		expected[i] = r.Uint32()
		actual[i] = expected[i]
	}
	slices.Sort(expected)
	return expected, actual
}

func initializeSameValue(capacity int, r *rand.Rand) ([]uint32, []uint32) {
	expected := make([]uint32, capacity)
	actual := make([]uint32, capacity)
	v := r.Uint32()
	for i := range expected {
		expected[i] = v
		actual[i] = v
	}
	return expected, actual
}

func initialize(t *testing.T) {
	runtime.LockOSThread()
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.InitWindow(1, 1, "Prefix sum")
	if err := gl.Init(); err != nil {
		assert.NoError(t, err, "gl.Init() should succeed")
	}
}

func linear(cap int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := range cap {
			if !yield(i) {
				break
			}
		}
	}
}
