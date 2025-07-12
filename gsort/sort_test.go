//go:build opengl43

package gsort_test

import (
	"cmp"
	"iter"
	"math"
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

func TestSortSmall(t *testing.T) {
	const capacity = 8192
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	for td := range generateTestData(initializeRandomValues, r, linear(capacity)) {
		gpuSort(gs, td.actual, sb)
		arraysEqual(t, td.expected, td.actual)
	}
}
func TestSortSmallWorkGroup(t *testing.T) {
	const capacity = 1024
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity).WithValuesPerWorkGroup(2))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	for td := range generateTestData(initializeRandomValues, r, linear(capacity)) {
		gpuSort(gs, td.actual, sb)
		arraysEqual(t, td.expected, td.actual)
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
		arraysEqual(t, td.expected, td.actual)
	}
}

func TestSortLarge(t *testing.T) {
	const capacity = 1 << 24
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	for td := range generateTestData(initializeRandomValues, r, linearBetween(capacity-1, capacity)) {
		gpuSort(gs, td.actual, sb)
		arraysEqual(t, td.expected, td.actual)
	}
}

func TestSortLargeSameValue(t *testing.T) {
	const capacity = 1 << 24
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	for td := range generateTestData(initializeSameValue, r, linearBetween(capacity-1, capacity)) {
		gpuSort(gs, td.actual, sb)
		arraysEqual(t, td.expected, td.actual)
	}
}

func TestSortLargeSorted(t *testing.T) {
	const capacity = 1 << 24
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	for td := range generateTestData(initializeRandomValuesSorted, r, linearBetween(capacity-1, capacity)) {
		gpuSort(gs, td.actual, sb)
		arraysEqual(t, td.expected, td.actual)
	}
}

func TestSortLargeSortedReverse(t *testing.T) {
	const capacity = 1 << 24
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	for td := range generateTestData(initializeRandomValuesSortedReverse, r, linearBetween(capacity-1, capacity)) {
		gpuSort(gs, td.actual, sb)
		arraysEqual(t, td.expected, td.actual)
	}
}

func TestSortLargeWithMinAndMaxValues(t *testing.T) {
	const capacity = 1 << 24
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	for td := range generateTestData(initializeRandomValuesWithMinAndMax, r, linearBetween(capacity-1, capacity)) {
		gpuSort(gs, td.actual, sb)
		arraysEqual(t, td.expected, td.actual)
	}
}

func TestSortDoesNotWriteOutOfBounds(t *testing.T) {
	const capacity = 1 << 10
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*4, nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	sentinelValues := [10]uint32{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	var actualSentinelValues [10]uint32
	for td := range generateTestData(initializeRandomValuesWithMinAndMax, r, values(246, 502, 758, 1014)) {
		rl.UpdateShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(sentinelValues[:])), uint32(len(sentinelValues)*4), uint32(len(td.actual)*4))
		gpuSort(gs, td.actual, sb)
		arraysEqual(t, td.expected, td.actual)
		rl.ReadShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(actualSentinelValues[:])), uint32(len(sentinelValues)*4), uint32(len(td.actual)*4))
		arraysEqual(t, sentinelValues[:], actualSentinelValues[:])
	}
}

func TestSortStabilityPaddingBefore(t *testing.T) {
	type TestData struct {
		data1 uint32
		key   uint32
	}
	const capacity = 1 << 20
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity).WithKeyOffset(4).WithInputDataSize(8))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*uint32(unsafe.Sizeof(TestData{})), nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	testDataExpected := make([]TestData, capacity)
	testDataActual := make([]TestData, capacity)
	for i := range testDataExpected {
		testDataExpected[i] = TestData{
			data1: uint32(i),
			key:   r.Uint32(),
		}
		testDataActual[i] = testDataExpected[i]
	}
	slices.SortStableFunc(testDataExpected, func(a, b TestData) int {
		return cmp.Compare(a.key, b.key)
	})
	var p runtime.Pinner
	p.Pin(unsafe.SliceData(testDataActual))
	rl.UpdateShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(testDataActual)), capacity*uint32(unsafe.Sizeof(TestData{})), 0)
	p.Unpin()

	gs.Sort(sb, capacity)

	p.Pin(unsafe.SliceData(testDataActual))
	rl.ReadShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(testDataActual)), capacity*uint32(unsafe.Sizeof(TestData{})), 0)
	p.Unpin()

	for i := range testDataExpected {
		if testDataExpected[i] != testDataActual[i] {
			t.Fatalf("actual value differs at index %d, actual %d != %d expected", i, testDataActual[i], testDataExpected[i])
		}
	}
}
func TestSortStabilityPaddingAfter(t *testing.T) {
	type TestData struct {
		key   uint32
		data1 uint32
	}
	const capacity = 1 << 20
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity).WithKeyOffset(0).WithInputDataSize(8))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*uint32(unsafe.Sizeof(TestData{})), nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	testDataExpected := make([]TestData, capacity)
	testDataActual := make([]TestData, capacity)
	for i := range testDataExpected {
		testDataExpected[i] = TestData{
			key:   r.Uint32(),
			data1: uint32(i),
		}
		testDataActual[i] = testDataExpected[i]
	}
	slices.SortStableFunc(testDataExpected, func(a, b TestData) int {
		return cmp.Compare(a.key, b.key)
	})
	var p runtime.Pinner
	p.Pin(unsafe.SliceData(testDataActual))
	rl.UpdateShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(testDataActual)), capacity*uint32(unsafe.Sizeof(TestData{})), 0)
	p.Unpin()

	gs.Sort(sb, capacity)

	p.Pin(unsafe.SliceData(testDataActual))
	rl.ReadShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(testDataActual)), capacity*uint32(unsafe.Sizeof(TestData{})), 0)
	p.Unpin()

	for i := range testDataExpected {
		if testDataExpected[i] != testDataActual[i] {
			t.Fatalf("actual value differs at index %d, actual %d != %d expected", i, testDataActual[i], testDataExpected[i])
		}
	}
}
func TestSortStabilityPaddingBeforeAndAfter(t *testing.T) {
	type TestData struct {
		data1 uint32
		key   uint32
		data2 uint32
	}
	const capacity = 1 << 20
	initialize(t)

	r := rand.New(rand.NewSource(0))
	gs := gsort.New(gsort.NewSettings(capacity).WithKeyOffset(4).WithInputDataSize(12))
	defer gs.Free()

	sb := rl.LoadShaderBuffer(capacity*uint32(unsafe.Sizeof(TestData{})), nil, rl.DynamicCopy)
	defer rl.UnloadShaderBuffer(sb)

	testDataExpected := make([]TestData, capacity)
	testDataActual := make([]TestData, capacity)
	for i := range testDataExpected {
		testDataExpected[i] = TestData{
			data1: uint32(i),
			key:   r.Uint32(),
			data2: uint32(capacity - i),
		}
		testDataActual[i] = testDataExpected[i]
	}
	slices.SortStableFunc(testDataExpected, func(a, b TestData) int {
		return cmp.Compare(a.key, b.key)
	})
	var p runtime.Pinner
	p.Pin(unsafe.SliceData(testDataActual))
	rl.UpdateShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(testDataActual)), capacity*uint32(unsafe.Sizeof(TestData{})), 0)
	p.Unpin()

	gs.Sort(sb, capacity)

	p.Pin(unsafe.SliceData(testDataActual))
	rl.ReadShaderBuffer(sb, unsafe.Pointer(unsafe.SliceData(testDataActual)), capacity*uint32(unsafe.Sizeof(TestData{})), 0)
	p.Unpin()

	for i := range testDataExpected {
		if testDataExpected[i] != testDataActual[i] {
			t.Fatalf("actual value differs at index %d, actual %d != %d expected", i, testDataActual[i], testDataExpected[i])
		}
	}
}

func arraysEqual(t *testing.T, expected, actual []uint32) {
	for i := range expected {
		if expected[i] != actual[i] {
			t.Fatalf("actual value differs at index %d, actual %d != %d expected", i, actual[i], expected[i])
		}
	}
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

type generateValuesFunc func(cap int, r *rand.Rand) testData

func generateTestData(fn generateValuesFunc, r *rand.Rand, sizes iter.Seq[int]) iter.Seq[testData] {
	return func(yield func(testData) bool) {
		for v := range sizes {
			td := fn(v, r)
			if !yield(td) {
				break
			}
		}
	}
}

func initializeRandomValuesSorted(capacity int, r *rand.Rand) testData {
	expected := make([]uint32, capacity)
	actual := make([]uint32, capacity)

	for i := range expected {
		expected[i] = r.Uint32()
	}
	slices.Sort(expected)
	copy(actual, expected)
	return testData{actual: actual, expected: expected}
}

func initializeRandomValuesSortedReverse(capacity int, r *rand.Rand) testData {
	expected := make([]uint32, capacity)
	actual := make([]uint32, capacity)

	for i := range expected {
		expected[i] = r.Uint32()
	}
	slices.Sort(expected)
	copy(actual, expected)
	slices.Reverse(actual)
	return testData{actual: actual, expected: expected}
}

func initializeRandomValues(capacity int, r *rand.Rand) testData {
	expected := make([]uint32, capacity)
	actual := make([]uint32, capacity)

	for i := range expected {
		expected[i] = r.Uint32()
		actual[i] = expected[i]
	}
	slices.Sort(expected)
	return testData{actual: actual, expected: expected}
}

func initializeRandomValuesWithMinAndMax(capacity int, r *rand.Rand) testData {
	expected := make([]uint32, capacity)
	actual := make([]uint32, capacity)

	for i := range expected {
		v := r.Uint32()
		if v < math.MaxUint32/2 {
			if v < math.MaxUint32/4 {
				v = 0
			} else {
				v = math.MaxUint32
			}
		}
		expected[i] = v
		actual[i] = expected[i]
	}
	slices.Sort(expected)
	return testData{actual: actual, expected: expected}
}

func initializeSameValue(capacity int, r *rand.Rand) testData {
	expected := make([]uint32, capacity)
	actual := make([]uint32, capacity)
	v := r.Uint32()
	for i := range expected {
		expected[i] = v
		actual[i] = v
	}
	return testData{actual: actual, expected: expected}
}

func initialize(t testing.TB) {
	runtime.GOMAXPROCS(1)
	runtime.LockOSThread()
	rl.SetTraceLogLevel(rl.LogWarning)
	rl.InitWindow(600, 600, "Radix Sort Test")
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

func linearBetween(start, end int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := start; i < end; i++ {
			if !yield(i) {
				break
			}
		}
	}
}

func values(values ...int) iter.Seq[int] {
	return func(yield func(int) bool) {
		for _, v := range values {
			if !yield(v) {
				break
			}
		}
	}
}
